# Mach-O Writer Refactoring Plan

This document describes the planned refactoring of `assembler/slasm/macho.go` to improve maintainability, testability, and reliability.

## Current State

The file is ~1165 lines with a monolithic `WriteExecutable` function (634 lines) that:
- Calculates all offsets and sizes inline
- Builds Mach-O structures
- Writes everything to disk
- Generates code signatures

Key problems:
1. Single massive function is hard to test and maintain
2. ~50 repetitive error handling blocks
3. Magic numbers scattered throughout
4. Fragile exports trie generation with hardcoded byte arrays
5. No atomic writes (partial files on failure)
6. Unused `dyldInfoCommand` struct
7. Hardcoded UUID instead of content-based

## Phase 1: Extract Constants

Replace magic numbers with named constants. Add these near the existing constants (after line 724):

```go
// Memory layout constants
const (
    // VMBaseAddress is the standard base address for ARM64 macOS executables
    VMBaseAddress = 0x100000000

    // PageSize is the memory page size on ARM64 macOS (16KB)
    PageSize = 0x4000

    // PageZeroSize is the size of the __PAGEZERO segment (4GB null guard)
    PageZeroSize = 0x100000000

    // MinSegmentFileSize is the minimum file size for segments (matches system linker)
    MinSegmentFileSize = 0x4000
)

// Structure sizes (in bytes)
const (
    MachHeader64Size       = 32
    SegmentCommand64Size   = 72
    Section64Size          = 80
    EntryPointCmdSize      = 24
    DylinkerCmdBaseSize    = 12  // Without path
    DylibCmdBaseSize       = 24  // Without path
    UUIDCmdSize            = 24
    BuildVersionCmdSize    = 32  // With 1 tool entry
    SourceVersionCmdSize   = 16
    LinkeditDataCmdSize    = 16  // For chained fixups, exports trie, etc.
    SymtabCmdSize          = 24
    DysymtabCmdSize        = 80
    CodeSignatureCmdSize   = 16
    BuildToolVersionSize   = 8
    Nlist64Size            = 16
)

// Version constants
const (
    // MacOSMinVersion is the minimum macOS version (11.0.0 Big Sur)
    MacOSMinVersion = 0x000b0000

    // MacOSSDKVersion is the SDK version (15.0.0)
    MacOSSDKVersion = 0x000f0000

    // LibSystemVersion is the current libSystem version (1292.100.0)
    LibSystemVersion = 0x050c6400

    // LibSystemCompatVersion is the libSystem compatibility version (1.0.0)
    LibSystemCompatVersion = 0x00010000

    // LinkerToolVersion is the LD tool version for LC_BUILD_VERSION (1167.0)
    LinkerToolVersion = 0x048f0000

    // SourceVersion is the source version (1.0.0.0.0)
    SourceVersion = 0x0001000000000000
)

// Section flags
const (
    // SectionFlagPureInstructions marks a section as containing only machine instructions
    SectionFlagPureInstructions = 0x80000400
)

// Export symbol flags
const (
    ExportSymbolFlagsRegular = 0x00
)

// Dyld paths
const (
    DylinkerPath = "/usr/lib/dyld"
    LibSystemPath = "/usr/lib/libSystem.B.dylib"
)
```

### Update References

Replace all magic numbers in `WriteExecutable` with these constants. Examples:

```go
// Before:
vmAddr := uint64(0x100000000)

// After:
vmAddr := uint64(VMBaseAddress)
```

```go
// Before:
textSegmentFileSize := uint64(0x4000)

// After:
textSegmentFileSize := uint64(MinSegmentFileSize)
```

```go
// Before:
headerSize := uint64(32)

// After:
headerSize := uint64(MachHeader64Size)
```

---

## Phase 2: Add Binary Writer Helper

Add this helper type after the struct definitions (around line 890):

```go
// binaryWriter wraps an io.Writer with deferred error handling for binary writes.
// Once an error occurs, subsequent writes become no-ops.
type binaryWriter struct {
    w   io.Writer
    err error
}

// newBinaryWriter creates a new binaryWriter wrapping the given writer.
func newBinaryWriter(w io.Writer) *binaryWriter {
    return &binaryWriter{w: w}
}

// write writes a fixed-size value in little-endian format.
// If a previous write failed, this is a no-op.
func (b *binaryWriter) write(data any) {
    if b.err != nil {
        return
    }
    b.err = binary.Write(b.w, binary.LittleEndian, data)
}

// writeBytes writes raw bytes.
// If a previous write failed, this is a no-op.
func (b *binaryWriter) writeBytes(data []byte) {
    if b.err != nil {
        return
    }
    _, b.err = b.w.Write(data)
}

// writePadding writes n zero bytes.
// If a previous write failed, this is a no-op.
func (b *binaryWriter) writePadding(n int) {
    if b.err != nil || n <= 0 {
        return
    }
    _, b.err = b.w.Write(make([]byte, n))
}

// error returns the first error that occurred, or nil.
func (b *binaryWriter) error() error {
    return b.err
}
```

### Update Write Calls

Replace the repetitive error handling pattern:

```go
// Before (lines 459-553):
if err := writeStruct(file, &header); err != nil {
    return err
}
if err := writeStruct(file, &pagezero); err != nil {
    return err
}
// ... 48 more times

// After:
bw := newBinaryWriter(file)
bw.write(&header)
bw.write(&pagezero)
bw.write(&segment)
bw.write(&section)
if hasDataSection {
    bw.write(&dataSegment)
    bw.write(&dataSection)
}
bw.write(&linkedit)
bw.write(&dylinkerCmd)
bw.writeBytes(dylinkerPathBytes)
bw.write(&dylibCmd)
bw.writeBytes(dylibPathBytes)
bw.write(&entryCmd)
bw.write(&uuidCmd)
bw.write(&buildVersionCmd)
bw.write(&buildToolEntry)
bw.write(&sourceVersionCmd)
bw.write(&chainedFixupsCmd)
bw.write(&exportsTrieCmd)
bw.write(&symtabCmd)
bw.write(&dysymtabCmd)
bw.write(&functionStartsCmd)
bw.write(&dataInCodeCmd)
bw.write(&codeSignatureCmd)

if err := bw.error(); err != nil {
    return err
}
```

---

## Phase 3: Extract Layout Calculation

Add a layout type and calculation function. Place after the constants:

```go
// executableLayout holds all computed offsets and sizes for a Mach-O executable.
type executableLayout struct {
    // Header and load commands
    headerSize     uint64
    loadCmdsSize   uint64
    numLoadCmds    uint32

    // __TEXT segment
    codeOffset         uint64
    codeSize           uint64
    textSegmentFileOff uint64
    textSegmentFileSize uint64
    textVMAddr         uint64
    textVMSize         uint64

    // __DATA segment (zero if no data)
    hasDataSection      bool
    dataOffset          uint64
    dataSize            uint64
    dataSegmentFileSize uint64
    dataVMAddr          uint64
    dataVMSize          uint64

    // __LINKEDIT segment
    linkeditOffset     uint64
    linkeditSize       uint64
    linkeditVMAddr     uint64
    linkeditVMSize     uint64

    // Linkedit contents
    chainedFixupsOffset uint64
    chainedFixupsSize   uint64
    exportsTrieOffset   uint64
    exportsTrieSize     uint64
    symtabOffset        uint64
    symtabSize          uint64
    stringTableOffset   uint64
    stringTableSize     uint64
    functionStartsOffset uint64
    functionStartsSize   uint64

    // Code signature
    signatureOffset uint64
    signatureSize   uint64

    // Dynamic library paths (for size calculation)
    dylinkerCmdSize uint64
    dylibCmdSize    uint64
}

// calculateLayout computes all offsets and sizes for the executable.
func (w *MachOWriter) calculateLayout(code, data []byte, relocations []DataRelocation) *executableLayout {
    layout := &executableLayout{}

    // Fixed sizes
    layout.headerSize = MachHeader64Size
    layout.codeSize = uint64(len(code))
    layout.dataSize = uint64(len(data))
    layout.hasDataSection = len(data) > 0

    // Dynamic library command sizes (path + null + alignment)
    layout.dylinkerCmdSize = uint64(DylinkerCmdBaseSize + len(DylinkerPath) + 1)
    layout.dylinkerCmdSize = (layout.dylinkerCmdSize + 7) &^ 7 // Align to 8 bytes

    layout.dylibCmdSize = uint64(DylibCmdBaseSize + len(LibSystemPath) + 1)
    layout.dylibCmdSize = (layout.dylibCmdSize + 7) &^ 7

    // Calculate total load commands size
    // Base commands: __PAGEZERO, __TEXT+section, __LINKEDIT, dylinker, dylib,
    //                entry, uuid, build_version, source_version,
    //                chained_fixups, exports_trie, symtab, dysymtab,
    //                function_starts, data_in_code, code_signature
    layout.loadCmdsSize = SegmentCommand64Size +                    // __PAGEZERO
        SegmentCommand64Size + Section64Size +                       // __TEXT + __text
        SegmentCommand64Size +                                       // __LINKEDIT
        layout.dylinkerCmdSize +
        layout.dylibCmdSize +
        EntryPointCmdSize +
        UUIDCmdSize +
        BuildVersionCmdSize +
        SourceVersionCmdSize +
        LinkeditDataCmdSize + // chained fixups
        LinkeditDataCmdSize + // exports trie
        SymtabCmdSize +
        DysymtabCmdSize +
        LinkeditDataCmdSize + // function starts
        LinkeditDataCmdSize + // data in code
        CodeSignatureCmdSize

    layout.numLoadCmds = 16

    // Add __DATA segment if needed
    if layout.hasDataSection {
        layout.loadCmdsSize += SegmentCommand64Size + Section64Size
        layout.numLoadCmds++
    }

    // Code placement (after header + load commands, 8-byte aligned)
    layout.codeOffset = layout.headerSize + layout.loadCmdsSize
    layout.codeOffset = (layout.codeOffset + 7) &^ 7

    // __TEXT segment: starts at file offset 0, includes header + code
    layout.textSegmentFileOff = 0
    layout.textVMAddr = VMBaseAddress
    layout.textSegmentFileSize = MinSegmentFileSize
    if (layout.codeOffset + layout.codeSize) > layout.textSegmentFileSize {
        layout.textSegmentFileSize = ((layout.codeOffset + layout.codeSize + 0xFFF) / 0x1000) * 0x1000
    }
    layout.textVMSize = layout.textSegmentFileSize

    // __DATA segment (if present)
    if layout.hasDataSection {
        layout.dataOffset = layout.textSegmentFileSize
        layout.dataVMAddr = layout.textVMAddr + layout.textVMSize
        layout.dataSegmentFileSize = MinSegmentFileSize
        if layout.dataSize > layout.dataSegmentFileSize {
            layout.dataSegmentFileSize = ((layout.dataSize + PageSize - 1) / PageSize) * PageSize
        }
        layout.dataVMSize = layout.dataSegmentFileSize
    }

    // __LINKEDIT segment
    layout.linkeditOffset = layout.textSegmentFileSize + layout.dataSegmentFileSize
    layout.linkeditVMAddr = layout.textVMAddr + layout.textVMSize + layout.dataVMSize

    // Linkedit contents
    chainedFixupsData := generateChainedFixupsWithRelocations(
        make([]byte, len(data)), // Don't modify actual data yet
        relocations,
        layout.dataVMAddr,
        layout.textVMAddr,
    )
    layout.chainedFixupsSize = uint64(len(chainedFixupsData))
    layout.chainedFixupsOffset = layout.linkeditOffset

    layout.exportsTrieSize = 48
    layout.exportsTrieOffset = layout.chainedFixupsOffset + layout.chainedFixupsSize

    layout.symtabSize = Nlist64Size // 1 symbol
    layout.symtabOffset = layout.exportsTrieOffset + layout.exportsTrieSize

    layout.stringTableSize = 16
    layout.stringTableOffset = layout.symtabOffset + layout.symtabSize

    layout.functionStartsSize = 8
    layout.functionStartsOffset = layout.stringTableOffset + layout.stringTableSize

    // Code signature
    linkeditDataBeforeSig := layout.chainedFixupsSize + layout.exportsTrieSize +
        layout.symtabSize + layout.stringTableSize + layout.functionStartsSize
    fileSizeForSig := int64(layout.linkeditOffset + linkeditDataBeforeSig)
    signatureID := "slasm-binary"
    layout.signatureSize = uint64(codesign.Size(fileSizeForSig, signatureID))
    layout.signatureSize = (layout.signatureSize + 15) &^ 15 // Align to 16
    layout.signatureOffset = layout.linkeditOffset + linkeditDataBeforeSig

    // Total linkedit size
    layout.linkeditSize = linkeditDataBeforeSig + layout.signatureSize
    layout.linkeditVMSize = ((layout.linkeditSize + 0xFFF) / 0x1000) * 0x1000
    if layout.linkeditVMSize < 0x1000 {
        layout.linkeditVMSize = 0x1000
    }

    return layout
}
```

---

## Phase 4: Extract Command Building

Add a type to hold all commands and a builder function:

```go
// loadCommands holds all Mach-O load commands for an executable.
type loadCommands struct {
    header          machHeader64
    pagezero        segmentCommand64
    textSegment     segmentCommand64
    textSection     section64
    dataSegment     segmentCommand64
    dataSection     section64
    linkedit        segmentCommand64
    dylinker        dylinkerCommand
    dylinkerPath    []byte
    dylib           dylibCommand
    dylibPath       []byte
    entryPoint      entryPointCommand
    uuid            uuidCommand
    buildVersion    buildVersionCommand
    buildTool       buildToolVersion
    sourceVersion   sourceVersionCommand
    chainedFixups   linkeditDataCommand
    exportsTrie     linkeditDataCommand
    symtab          symtabCommand
    dysymtab        dysymtabCommand
    functionStarts  linkeditDataCommand
    dataInCode      linkeditDataCommand
    codeSignature   codeSignatureCommand
}

// buildLoadCommands constructs all Mach-O structures from the layout.
func (w *MachOWriter) buildLoadCommands(layout *executableLayout, code, data []byte) *loadCommands {
    cmds := &loadCommands{}

    // Header
    cmds.header = machHeader64{
        Magic:      MH_MAGIC_64,
        CPUType:    CPU_TYPE_ARM64,
        CPUSubtype: CPU_SUBTYPE_ARM64,
        FileType:   MH_EXECUTE,
        NCmds:      layout.numLoadCmds,
        SizeofCmds: uint32(layout.loadCmdsSize),
        Flags:      MH_NOUNDEFS | MH_DYLDLINK | MH_TWOLEVEL | MH_PIE,
        Reserved:   0,
    }

    // __PAGEZERO
    var pagezeroName [16]byte
    copy(pagezeroName[:], "__PAGEZERO")
    cmds.pagezero = segmentCommand64{
        Cmd:      LC_SEGMENT_64,
        Cmdsize:  SegmentCommand64Size,
        Segname:  pagezeroName,
        VMAddr:   0,
        VMSize:   PageZeroSize,
        FileOff:  0,
        FileSize: 0,
        MaxProt:  0,
        InitProt: 0,
        NSects:   0,
        Flags:    0,
    }

    // __TEXT segment
    var textSegname [16]byte
    copy(textSegname[:], "__TEXT")
    cmds.textSegment = segmentCommand64{
        Cmd:      LC_SEGMENT_64,
        Cmdsize:  uint32(SegmentCommand64Size + Section64Size),
        Segname:  textSegname,
        VMAddr:   layout.textVMAddr,
        VMSize:   layout.textVMSize,
        FileOff:  0,
        FileSize: layout.textSegmentFileSize,
        MaxProt:  VM_PROT_READ | VM_PROT_EXECUTE,
        InitProt: VM_PROT_READ | VM_PROT_EXECUTE,
        NSects:   1,
        Flags:    0,
    }

    // __text section
    var textSectname [16]byte
    copy(textSectname[:], "__text")
    cmds.textSection = section64{
        Sectname:  textSectname,
        Segname:   textSegname,
        Addr:      layout.textVMAddr + layout.codeOffset,
        Size:      layout.codeSize,
        Offset:    uint32(layout.codeOffset),
        Align:     2,
        Reloff:    0,
        Nreloc:    0,
        Flags:     SectionFlagPureInstructions,
        Reserved1: 0,
        Reserved2: 0,
        Reserved3: 0,
    }

    // __DATA segment (if present)
    if layout.hasDataSection {
        var dataSegname [16]byte
        copy(dataSegname[:], "__DATA")
        var dataSectname [16]byte
        copy(dataSectname[:], "__data")

        cmds.dataSegment = segmentCommand64{
            Cmd:      LC_SEGMENT_64,
            Cmdsize:  uint32(SegmentCommand64Size + Section64Size),
            Segname:  dataSegname,
            VMAddr:   layout.dataVMAddr,
            VMSize:   layout.dataVMSize,
            FileOff:  layout.dataOffset,
            FileSize: layout.dataSegmentFileSize,
            MaxProt:  VM_PROT_READ | VM_PROT_WRITE,
            InitProt: VM_PROT_READ | VM_PROT_WRITE,
            NSects:   1,
            Flags:    0,
        }

        cmds.dataSection = section64{
            Sectname:  dataSectname,
            Segname:   dataSegname,
            Addr:      layout.dataVMAddr,
            Size:      layout.dataSize,
            Offset:    uint32(layout.dataOffset),
            Align:     3,
            Reloff:    0,
            Nreloc:    0,
            Flags:     0,
            Reserved1: 0,
            Reserved2: 0,
            Reserved3: 0,
        }
    }

    // __LINKEDIT segment
    var linkeditName [16]byte
    copy(linkeditName[:], "__LINKEDIT")
    cmds.linkedit = segmentCommand64{
        Cmd:      LC_SEGMENT_64,
        Cmdsize:  SegmentCommand64Size,
        Segname:  linkeditName,
        VMAddr:   layout.linkeditVMAddr,
        VMSize:   layout.linkeditVMSize,
        FileOff:  layout.linkeditOffset,
        FileSize: layout.linkeditSize,
        MaxProt:  VM_PROT_READ,
        InitProt: VM_PROT_READ,
        NSects:   0,
        Flags:    0,
    }

    // Dylinker
    cmds.dylinker = dylinkerCommand{
        Cmd:     LC_LOAD_DYLINKER,
        Cmdsize: uint32(layout.dylinkerCmdSize),
        NameOff: DylinkerCmdBaseSize,
    }
    cmds.dylinkerPath = make([]byte, layout.dylinkerCmdSize-DylinkerCmdBaseSize)
    copy(cmds.dylinkerPath, DylinkerPath)

    // Dylib
    cmds.dylib = dylibCommand{
        Cmd:                  LC_LOAD_DYLIB,
        Cmdsize:              uint32(layout.dylibCmdSize),
        NameOff:              DylibCmdBaseSize,
        Timestamp:            2,
        CurrentVersion:       LibSystemVersion,
        CompatibilityVersion: LibSystemCompatVersion,
    }
    cmds.dylibPath = make([]byte, layout.dylibCmdSize-DylibCmdBaseSize)
    copy(cmds.dylibPath, LibSystemPath)

    // Entry point
    cmds.entryPoint = entryPointCommand{
        Cmd:       LC_MAIN,
        Cmdsize:   EntryPointCmdSize,
        EntryOff:  layout.codeOffset,
        StackSize: 0,
    }

    // UUID (content-based)
    cmds.uuid = uuidCommand{
        Cmd:     LC_UUID,
        Cmdsize: UUIDCmdSize,
        UUID:    generateContentUUID(code, data),
    }

    // Build version
    cmds.buildVersion = buildVersionCommand{
        Cmd:      LC_BUILD_VERSION,
        Cmdsize:  BuildVersionCmdSize,
        Platform: PLATFORM_MACOS,
        Minos:    MacOSMinVersion,
        Sdk:      MacOSSDKVersion,
        Ntools:   1,
    }
    cmds.buildTool = buildToolVersion{
        Tool:    3, // LD
        Version: LinkerToolVersion,
    }

    // Source version
    cmds.sourceVersion = sourceVersionCommand{
        Cmd:     LC_SOURCE_VERSION,
        Cmdsize: SourceVersionCmdSize,
        Version: SourceVersion,
    }

    // Chained fixups
    cmds.chainedFixups = linkeditDataCommand{
        Cmd:      LC_DYLD_CHAINED_FIXUPS,
        Cmdsize:  LinkeditDataCmdSize,
        DataOff:  uint32(layout.chainedFixupsOffset),
        DataSize: uint32(layout.chainedFixupsSize),
    }

    // Exports trie
    cmds.exportsTrie = linkeditDataCommand{
        Cmd:      LC_DYLD_EXPORTS_TRIE,
        Cmdsize:  LinkeditDataCmdSize,
        DataOff:  uint32(layout.exportsTrieOffset),
        DataSize: uint32(layout.exportsTrieSize),
    }

    // Symtab
    cmds.symtab = symtabCommand{
        Cmd:     LC_SYMTAB,
        Cmdsize: SymtabCmdSize,
        Symoff:  uint32(layout.symtabOffset),
        Nsyms:   1,
        Stroff:  uint32(layout.stringTableOffset),
        Strsize: uint32(layout.stringTableSize),
    }

    // Dysymtab
    cmds.dysymtab = dysymtabCommand{
        Cmd:        LC_DYSYMTAB,
        Cmdsize:    DysymtabCmdSize,
        Ilocalsym:  0,
        Nlocalsym:  0,
        Iextdefsym: 0,
        Nextdefsym: 1,
        Iundefsym:  1,
        Nundefsym:  0,
        // All other fields zero
    }

    // Function starts
    cmds.functionStarts = linkeditDataCommand{
        Cmd:      LC_FUNCTION_STARTS,
        Cmdsize:  LinkeditDataCmdSize,
        DataOff:  uint32(layout.functionStartsOffset),
        DataSize: uint32(layout.functionStartsSize),
    }

    // Data in code
    cmds.dataInCode = linkeditDataCommand{
        Cmd:      LC_DATA_IN_CODE,
        Cmdsize:  LinkeditDataCmdSize,
        DataOff:  uint32(layout.functionStartsOffset + layout.functionStartsSize),
        DataSize: 0,
    }

    // Code signature
    cmds.codeSignature = codeSignatureCommand{
        Cmd:      LC_CODE_SIGNATURE,
        Cmdsize:  CodeSignatureCmdSize,
        DataOff:  uint32(layout.signatureOffset),
        DataSize: uint32(layout.signatureSize),
    }

    return cmds
}

// generateContentUUID generates a deterministic UUID from code and data content.
func generateContentUUID(code, data []byte) [16]byte {
    // Use a simple hash-based approach
    // In production, consider using crypto/sha256
    var uuid [16]byte

    // Simple FNV-1a inspired hash
    hash := uint64(14695981039346656037)
    for _, b := range code {
        hash ^= uint64(b)
        hash *= 1099511628211
    }
    for _, b := range data {
        hash ^= uint64(b)
        hash *= 1099511628211
    }

    // Fill UUID with hash bytes
    binary.LittleEndian.PutUint64(uuid[0:8], hash)
    binary.LittleEndian.PutUint64(uuid[8:16], hash*1099511628211)

    // Set UUID version 4 and variant 1
    uuid[6] = (uuid[6] & 0x0f) | 0x40
    uuid[8] = (uuid[8] & 0x3f) | 0x80

    return uuid
}
```

---

## Phase 5: Rewrite WriteExecutable

Replace the entire `WriteExecutable` function with this cleaner version:

```go
// WriteExecutable writes a Mach-O executable to the specified path.
func (w *MachOWriter) WriteExecutable(outputPath string, code []byte, data []byte, relocations []DataRelocation, symbols *SymbolTable, entryPoint string) error {
    // Calculate layout
    layout := w.calculateLayout(code, data, relocations)

    // Build load commands
    cmds := w.buildLoadCommands(layout, code, data)

    // Create temp file for atomic write
    tmpPath := outputPath + ".tmp"
    file, err := os.Create(tmpPath)
    if err != nil {
        return err
    }
    defer func() {
        file.Close()
        if err != nil {
            os.Remove(tmpPath)
        }
    }()

    // Write header and load commands
    if err = w.writeHeaderAndCommands(file, layout, cmds); err != nil {
        return err
    }

    // Write segment data
    if err = w.writeSegmentData(file, layout, code, data, relocations); err != nil {
        return err
    }

    // Generate and write code signature
    if err = w.writeCodeSignature(file, layout); err != nil {
        return err
    }

    // Log structure info
    w.logLayout(layout)

    // Close and rename atomically
    if err = file.Close(); err != nil {
        return err
    }
    return os.Rename(tmpPath, outputPath)
}

// writeHeaderAndCommands writes the Mach-O header and all load commands.
func (w *MachOWriter) writeHeaderAndCommands(file *os.File, layout *executableLayout, cmds *loadCommands) error {
    bw := newBinaryWriter(file)

    // Header
    bw.write(&cmds.header)

    // Segments and sections
    bw.write(&cmds.pagezero)
    bw.write(&cmds.textSegment)
    bw.write(&cmds.textSection)
    if layout.hasDataSection {
        bw.write(&cmds.dataSegment)
        bw.write(&cmds.dataSection)
    }
    bw.write(&cmds.linkedit)

    // Dylinker
    bw.write(&cmds.dylinker)
    bw.writeBytes(cmds.dylinkerPath)

    // Dylib
    bw.write(&cmds.dylib)
    bw.writeBytes(cmds.dylibPath)

    // Other commands
    bw.write(&cmds.entryPoint)
    bw.write(&cmds.uuid)
    bw.write(&cmds.buildVersion)
    bw.write(&cmds.buildTool)
    bw.write(&cmds.sourceVersion)
    bw.write(&cmds.chainedFixups)
    bw.write(&cmds.exportsTrie)
    bw.write(&cmds.symtab)
    bw.write(&cmds.dysymtab)
    bw.write(&cmds.functionStarts)
    bw.write(&cmds.dataInCode)
    bw.write(&cmds.codeSignature)

    return bw.error()
}

// writeSegmentData writes the actual segment content (code, data, linkedit).
func (w *MachOWriter) writeSegmentData(file *os.File, layout *executableLayout, code, data []byte, relocations []DataRelocation) error {
    // Seek to code offset
    if _, err := file.Seek(int64(layout.codeOffset), 0); err != nil {
        return err
    }

    bw := newBinaryWriter(file)

    // Write code
    bw.writeBytes(code)

    // Pad __TEXT segment
    textPadding := int(layout.textSegmentFileSize - layout.codeOffset - layout.codeSize)
    bw.writePadding(textPadding)

    // Write __DATA segment
    if layout.hasDataSection {
        bw.writeBytes(data)
        dataPadding := int(layout.dataSegmentFileSize - layout.dataSize)
        bw.writePadding(dataPadding)
    }

    // Write __LINKEDIT contents
    chainedFixupsData := generateChainedFixupsWithRelocations(data, relocations, layout.dataVMAddr, layout.textVMAddr)
    bw.writeBytes(chainedFixupsData)

    exportsTrieData := generateMinimalExportsTrie(layout.codeOffset)
    bw.writeBytes(exportsTrieData)

    // Symbol table entry
    symbolEntry := make([]byte, Nlist64Size)
    binary.LittleEndian.PutUint32(symbolEntry[0:4], 1)
    symbolEntry[4] = 0x0f
    symbolEntry[5] = 1
    binary.LittleEndian.PutUint64(symbolEntry[8:16], layout.textVMAddr+layout.codeOffset)
    bw.writeBytes(symbolEntry)

    // String table
    stringTable := make([]byte, 16)
    stringTable[0] = ' '
    copy(stringTable[1:], "_start\x00")
    bw.writeBytes(stringTable)

    // Function starts
    functionStartsData := generateFunctionStarts(layout.codeOffset)
    bw.writeBytes(functionStartsData)

    return bw.error()
}

// writeCodeSignature generates and writes the code signature.
func (w *MachOWriter) writeCodeSignature(file *os.File, layout *executableLayout) error {
    // Read file content for signing
    if _, err := file.Seek(0, 0); err != nil {
        return err
    }

    fileSizeForSig := int64(layout.signatureOffset)
    fileContent := make([]byte, fileSizeForSig)
    if _, err := io.ReadFull(file, fileContent); err != nil {
        return err
    }

    // Generate signature
    signatureID := "slasm-binary"
    signatureData := make([]byte, layout.signatureSize)
    codesign.Sign(signatureData, bytes.NewReader(fileContent), signatureID, fileSizeForSig, 0, int64(layout.textSegmentFileSize), true)

    // Write signature
    if _, err := file.Seek(int64(layout.signatureOffset), 0); err != nil {
        return err
    }
    _, err := file.Write(signatureData)
    return err
}

// logLayout logs the Mach-O structure information.
func (w *MachOWriter) logLayout(layout *executableLayout) {
    w.logger.Printf("\nMach-O Structure:\n")
    w.logger.Printf("  Header:            size=%d bytes\n", layout.headerSize)
    w.logger.Printf("  Load commands:     size=%d bytes, count=%d\n", layout.loadCmdsSize, layout.numLoadCmds)
    w.logger.Printf("  Code offset:       0x%x (%d bytes)\n", layout.codeOffset, layout.codeOffset)
    w.logger.Printf("  Code size:         %d bytes\n", layout.codeSize)
    w.logger.Printf("\nSegments:\n")
    w.logger.Printf("  __PAGEZERO:        vm=0x%x-0x%x (size=0x%x)\n", uint64(0), uint64(PageZeroSize), uint64(PageZeroSize))
    w.logger.Printf("  __TEXT:            vm=0x%x-0x%x (size=0x%x), file=0x%x-0x%x\n",
        layout.textVMAddr, layout.textVMAddr+layout.textVMSize, layout.textVMSize, uint64(0), layout.textSegmentFileSize)
    w.logger.Printf("    __text section:  vm=0x%x-0x%x (size=0x%x), file=0x%x\n",
        layout.textVMAddr+layout.codeOffset, layout.textVMAddr+layout.codeOffset+layout.codeSize, layout.codeSize, layout.codeOffset)
    if layout.hasDataSection {
        w.logger.Printf("  __DATA:            vm=0x%x-0x%x (size=0x%x), file=0x%x-0x%x\n",
            layout.dataVMAddr, layout.dataVMAddr+layout.dataVMSize, layout.dataVMSize, layout.dataOffset, layout.dataOffset+layout.dataSegmentFileSize)
        w.logger.Printf("    __data section:  vm=0x%x-0x%x (size=0x%x), file=0x%x\n",
            layout.dataVMAddr, layout.dataVMAddr+layout.dataSize, layout.dataSize, layout.dataOffset)
    }
    w.logger.Printf("  __LINKEDIT:        vm=0x%x-0x%x (size=0x%x), file=0x%x\n",
        layout.linkeditVMAddr, layout.linkeditVMAddr+layout.linkeditVMSize, layout.linkeditVMSize, layout.linkeditOffset)
    w.logger.Printf("\nEntry point:         0x%x (file offset 0x%x)\n", layout.textVMAddr+layout.codeOffset, layout.codeOffset)
    w.logger.Printf("Total file size:     %d bytes\n", layout.linkeditOffset+layout.linkeditSize)
    w.logger.Printf("  Code signature:    offset=0x%x, size=%d bytes\n", layout.signatureOffset, layout.signatureSize)
}
```

---

## Phase 6: Cleanup

### Remove Unused Code

Delete the `dyldInfoCommand` struct (lines 874-888) - it's never used.

### Delete Old writeStruct Helper

The `writeStruct` function (line 1162) can be removed once the binaryWriter is in use.

---

## Testing Strategy

After each phase, run the full test suite:

```bash
go run cmd/slm/main.go test
go run cmd/slm/main.go test-integration
```

Specifically verify:
1. E2E tests in `test/sl/` still pass
2. Generated executables run correctly
3. `codesign -v` validates the signature

### Additional Tests to Add

```go
func TestCalculateLayout(t *testing.T) {
    w := NewMachOWriter("arm64", nil)

    tests := []struct {
        name     string
        codeSize int
        dataSize int
    }{
        {"empty", 0, 0},
        {"code only", 100, 0},
        {"code and data", 100, 50},
        {"large code", 0x5000, 0},
        {"large data", 100, 0x5000},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            code := make([]byte, tt.codeSize)
            data := make([]byte, tt.dataSize)
            layout := w.calculateLayout(code, data, nil)

            // Verify invariants
            if layout.codeOffset < layout.headerSize+layout.loadCmdsSize {
                t.Error("code offset overlaps header")
            }
            if layout.hasDataSection != (tt.dataSize > 0) {
                t.Error("hasDataSection mismatch")
            }
            // ... more invariant checks
        })
    }
}

func TestBinaryWriter(t *testing.T) {
    var buf bytes.Buffer
    bw := newBinaryWriter(&buf)

    bw.write(uint32(0x12345678))
    bw.write(uint16(0xABCD))

    if err := bw.error(); err != nil {
        t.Fatal(err)
    }

    expected := []byte{0x78, 0x56, 0x34, 0x12, 0xCD, 0xAB}
    if !bytes.Equal(buf.Bytes(), expected) {
        t.Errorf("got %x, want %x", buf.Bytes(), expected)
    }
}
```

---

## Implementation Order

1. **Phase 1**: Add constants (low risk, immediate benefit)
2. **Phase 2**: Add binaryWriter helper (isolated, testable)
3. **Phase 3**: Add layout calculation (new code, doesn't change existing)
4. **Phase 4**: Add command building (new code, doesn't change existing)
5. **Phase 5**: Rewrite WriteExecutable to use new helpers
6. **Phase 6**: Remove dead code

Each phase should be a separate commit for easy rollback.

---

## Rollback Plan

If issues arise:
1. Each phase is a separate commit - `git revert` the problematic commit
2. The original `WriteExecutable` function should be preserved in git history
3. Run `codesign -v` on generated binaries to verify signature validity

---

## Future Improvements (Out of Scope)

These are noted but not part of this refactoring:

1. **Implement WriteObjectFile** - Currently a stub
2. **Proper exports trie builder** - Replace hardcoded bytes with structured generation
3. **Multiple entry point support** - Currently hardcoded to `_start`
4. **Dynamic symbol support** - For linking with external libraries
5. **Universal binary support** - Fat Mach-O for multiple architectures

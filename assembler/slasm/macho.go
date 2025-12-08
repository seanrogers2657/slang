package slasm

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"

	"github.com/seanrogers2657/slang/assembler/slasm/codesign"
)

// MachOWriter writes Mach-O object files and executables
type MachOWriter struct {
	arch        string
	objectCode  []byte
	dataSection []byte
	symbolTable *SymbolTable
	logger      *Logger
}

// NewMachOWriter creates a new Mach-O file writer
func NewMachOWriter(arch string, logger *Logger) *MachOWriter {
	if logger == nil {
		logger = NewSilentLogger()
	}
	return &MachOWriter{
		arch:   arch,
		logger: logger,
	}
}

// WriteObjectFile writes a Mach-O object file (.o)
func (w *MachOWriter) WriteObjectFile(outputPath string, code []byte, data []byte, symbols *SymbolTable) error {
	// Object file structure:
	// 1. mach_header_64 (FileType = MH_OBJECT)
	// 2. LC_SEGMENT_64 (unnamed, containing all sections)
	//    - __text section header
	//    - __data section header (if data present)
	// 3. LC_SYMTAB
	// 4. LC_DYSYMTAB
	// 5. Section data (__text, __data)
	// 6. Relocation entries (after each section's data, referenced by section header)
	// 7. Symbol table (nlist64 entries)
	// 8. String table

	hasData := len(data) > 0

	// Calculate number of sections
	numSections := uint32(1) // __text
	if hasData {
		numSections++
	}

	// Calculate sizes and offsets
	headerSize := uint32(MachHeader64Size)

	// Segment command size includes section headers
	segmentCmdSize := uint32(SegmentCommand64Size) + numSections*uint32(Section64Size)
	symtabCmdSize := uint32(SymtabCmdSize)
	dysymtabCmdSize := uint32(DysymtabCmdSize)

	loadCmdsSize := segmentCmdSize + symtabCmdSize + dysymtabCmdSize
	numLoadCmds := uint32(3) // segment, symtab, dysymtab

	// Section data starts after header + load commands, aligned to 4 bytes
	sectionDataStart := align(headerSize+loadCmdsSize, 4)
	textOffset := sectionDataStart
	textSize := uint32(len(code))

	// Data section follows text (if present)
	dataOffset := uint32(0)
	dataSize := uint32(0)
	if hasData {
		dataOffset = align(textOffset+textSize, 4)
		dataSize = uint32(len(data))
	}

	// Calculate where symbol table and string table go
	var symbolsOffset uint32
	if hasData {
		symbolsOffset = align(dataOffset+dataSize, 8)
	} else {
		symbolsOffset = align(textOffset+textSize, 8)
	}

	// Build symbol table entries
	symEntries, stringTable := w.buildObjectSymbols(symbols)
	numSymbols := uint32(len(symEntries))
	symbolTableSize := numSymbols * uint32(Nlist64Size)

	// String table follows symbol table
	stringsOffset := symbolsOffset + symbolTableSize
	stringTableSize := uint32(len(stringTable))

	// Total file size
	totalSize := stringsOffset + stringTableSize

	// Create file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	bw := newBinaryWriter(file)

	// Write Mach-O header
	header := machHeader64{
		Magic:      MH_MAGIC_64,
		CPUType:    CPU_TYPE_ARM64,
		CPUSubtype: CPU_SUBTYPE_ARM64,
		FileType:   MH_OBJECT,
		NCmds:      numLoadCmds,
		SizeofCmds: loadCmdsSize,
		Flags:      MH_NOUNDEFS, // No undefined symbols in our simple case
		Reserved:   0,
	}
	bw.write(&header)

	// Write segment command (unnamed for object files)
	var segmentSize uint64
	if hasData {
		segmentSize = uint64(dataOffset + dataSize - textOffset)
	} else {
		segmentSize = uint64(textSize)
	}

	segment := segmentCommand64{
		Cmd:      LC_SEGMENT_64,
		Cmdsize:  segmentCmdSize,
		VMAddr:   0,
		VMSize:   segmentSize,
		FileOff:  uint64(textOffset),
		FileSize: segmentSize,
		MaxProt:  VM_PROT_READ | VM_PROT_WRITE | VM_PROT_EXECUTE,
		InitProt: VM_PROT_READ | VM_PROT_WRITE | VM_PROT_EXECUTE,
		NSects:   numSections,
		Flags:    0,
	}
	// Leave Segname empty (unnamed segment for object files)
	bw.write(&segment)

	// Write __text section header
	textSection := section64{
		Addr:      0,
		Size:      uint64(textSize),
		Offset:    textOffset,
		Align:     2, // 4-byte aligned (2^2)
		Reloff:    0, // No relocations for now
		Nreloc:    0,
		Flags:     SectionFlagPureInstructions,
		Reserved1: 0,
		Reserved2: 0,
		Reserved3: 0,
	}
	copy(textSection.Sectname[:], "__text")
	copy(textSection.Segname[:], "__TEXT")
	bw.write(&textSection)

	// Write __data section header (if present)
	if hasData {
		dataSection := section64{
			Addr:      uint64(dataOffset - textOffset), // Relative to segment start
			Size:      uint64(dataSize),
			Offset:    dataOffset,
			Align:     3, // 8-byte aligned (2^3)
			Reloff:    0, // No relocations for now
			Nreloc:    0,
			Flags:     0, // S_REGULAR
			Reserved1: 0,
			Reserved2: 0,
			Reserved3: 0,
		}
		copy(dataSection.Sectname[:], "__data")
		copy(dataSection.Segname[:], "__DATA")
		bw.write(&dataSection)
	}

	// Write LC_SYMTAB
	symtab := symtabCommand{
		Cmd:     LC_SYMTAB,
		Cmdsize: symtabCmdSize,
		Symoff:  symbolsOffset,
		Nsyms:   numSymbols,
		Stroff:  stringsOffset,
		Strsize: stringTableSize,
	}
	bw.write(&symtab)

	// Write LC_DYSYMTAB
	dysymtab := dysymtabCommand{
		Cmd:            LC_DYSYMTAB,
		Cmdsize:        dysymtabCmdSize,
		Ilocalsym:      0,
		Nlocalsym:      0,
		Iextdefsym:     0,
		Nextdefsym:     numSymbols,
		Iundefsym:      numSymbols,
		Nundefsym:      0,
		Tocoff:         0,
		Ntoc:           0,
		Modtaboff:      0,
		Nmodtab:        0,
		Extrefsymoff:   0,
		Nextrefsyms:    0,
		Indirectsymoff: 0,
		Nindirectsyms:  0,
		Extreloff:      0,
		Nextrel:        0,
		Locreloff:      0,
		Nlocrel:        0,
	}
	bw.write(&dysymtab)

	// Pad to section data start
	currentPos := headerSize + loadCmdsSize
	if currentPos < sectionDataStart {
		padding := make([]byte, sectionDataStart-currentPos)
		bw.writeBytes(padding)
	}

	// Write __text section data
	bw.writeBytes(code)

	// Pad to data section (if present)
	if hasData {
		currentPos = textOffset + textSize
		if currentPos < dataOffset {
			padding := make([]byte, dataOffset-currentPos)
			bw.writeBytes(padding)
		}
		bw.writeBytes(data)
		currentPos = dataOffset + dataSize
	} else {
		currentPos = textOffset + textSize
	}

	// Pad to symbol table
	if currentPos < symbolsOffset {
		padding := make([]byte, symbolsOffset-currentPos)
		bw.writeBytes(padding)
	}

	// Write symbol table entries
	for _, sym := range symEntries {
		bw.write(&sym)
	}

	// Write string table
	bw.writeBytes(stringTable)

	// Pad to total size if needed
	currentPos = stringsOffset + stringTableSize
	if currentPos < totalSize {
		padding := make([]byte, totalSize-currentPos)
		bw.writeBytes(padding)
	}

	w.logger.Printf("Object file structure:\n")
	w.logger.Printf("  Header: %d bytes\n", headerSize)
	w.logger.Printf("  Load commands: %d bytes (%d commands)\n", loadCmdsSize, numLoadCmds)
	w.logger.Printf("  __text: offset=0x%x, size=%d bytes\n", textOffset, textSize)
	if hasData {
		w.logger.Printf("  __data: offset=0x%x, size=%d bytes\n", dataOffset, dataSize)
	}
	w.logger.Printf("  Symbol table: offset=0x%x, %d entries\n", symbolsOffset, numSymbols)
	w.logger.Printf("  String table: offset=0x%x, size=%d bytes\n", stringsOffset, stringTableSize)
	w.logger.Printf("  Total size: %d bytes\n", totalSize)

	return bw.error()
}

// buildObjectSymbols creates symbol table entries and string table for object file
func (w *MachOWriter) buildObjectSymbols(symbols *SymbolTable) ([]nlist64, []byte) {
	if symbols == nil {
		// Return minimal string table (just null byte)
		return nil, []byte{0}
	}

	var entries []nlist64
	var stringTable []byte
	stringTable = append(stringTable, 0) // String table starts with null

	// Add each symbol
	symbols.ForEach(func(name string, sym *Symbol) {
		strIdx := uint32(len(stringTable))
		stringTable = append(stringTable, []byte(name)...)
		stringTable = append(stringTable, 0) // Null terminator

		var symType uint8 = N_SECT // Defined in section
		if sym.Global {
			symType |= N_EXT // External symbol
		}

		var sect uint8 = 1 // Section 1 is __text
		if sym.Section == SectionData {
			sect = 2 // Section 2 is __data
		}

		entry := nlist64{
			Strx:  strIdx,
			Type:  symType,
			Sect:  sect,
			Desc:  0,
			Value: sym.Address,
		}
		entries = append(entries, entry)
	})

	return entries, stringTable
}

// align rounds up n to the nearest multiple of alignment
func align(n, alignment uint32) uint32 {
	if alignment == 0 {
		return n
	}
	return (n + alignment - 1) &^ (alignment - 1)
}

// WriteExecutable writes a Mach-O executable to the specified path.
func (w *MachOWriter) WriteExecutable(outputPath string, code []byte, data []byte, relocations []DataRelocation, symbols *SymbolTable, entryPoint string) error {
	// Calculate layout
	layout := w.calculateLayout(code, data, relocations)

	// Build load commands
	cmds := w.buildLoadCommands(layout, code, data)

	// Create file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header and load commands
	if err := w.writeHeaderAndCommands(file, layout, cmds); err != nil {
		return err
	}

	// Write segment data
	if err := w.writeSegmentData(file, layout, code, data, relocations); err != nil {
		return err
	}

	// Generate and write code signature
	if err := w.writeCodeSignature(file, layout); err != nil {
		return err
	}

	// Log structure info
	w.logLayout(layout)

	return nil
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
		// Generate chained fixups (this modifies data in-place)
		chainedFixupsData := generateChainedFixupsWithRelocations(data, relocations, layout.dataVMAddr, layout.textVMAddr)
		bw.writeBytes(data)
		dataPadding := int(layout.dataSegmentFileSize - layout.dataSize)
		bw.writePadding(dataPadding)

		// Write __LINKEDIT contents
		bw.writeBytes(chainedFixupsData)
	} else {
		// Write minimal chained fixups
		chainedFixupsData := generateMinimalChainedFixups()
		bw.writeBytes(chainedFixupsData)
	}

	// Write exports trie
	exportsTrieData := generateMinimalExportsTrie(layout.codeOffset)
	bw.writeBytes(exportsTrieData)

	// Write symbol table entry
	symbolEntry := make([]byte, Nlist64Size)
	binary.LittleEndian.PutUint32(symbolEntry[0:4], 1)
	symbolEntry[4] = 0x0f
	symbolEntry[5] = 1
	binary.LittleEndian.PutUint64(symbolEntry[8:16], layout.textVMAddr+layout.codeOffset)
	bw.writeBytes(symbolEntry)

	// Write string table
	stringTable := make([]byte, StringTableSize)
	stringTable[0] = ' '
	copy(stringTable[1:], "_start\x00")
	bw.writeBytes(stringTable)

	// Write function starts
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

// Mach-O constants and structures

const (
	MH_MAGIC_64 = 0xfeedfacf
	MH_OBJECT   = 0x1
	MH_EXECUTE  = 0x2

	CPU_TYPE_ARM64    = 0x0100000c
	CPU_SUBTYPE_ARM64 = 0x00000000

	LC_SEGMENT_64          = 0x19
	LC_SYMTAB              = 0x2
	LC_DYSYMTAB            = 0xb
	LC_LOAD_DYLINKER       = 0xe
	LC_LOAD_DYLIB          = 0xc
	LC_UUID                = 0x1b
	LC_BUILD_VERSION       = 0x32
	LC_SOURCE_VERSION      = 0x2a
	LC_CODE_SIGNATURE      = 0x1d
	LC_MAIN                = 0x80000028
	LC_DYLD_CHAINED_FIXUPS = 0x80000034
	LC_DYLD_EXPORTS_TRIE   = 0x80000033
	LC_FUNCTION_STARTS     = 0x26
	LC_DATA_IN_CODE        = 0x29

	// Mach-O header flags
	MH_NOUNDEFS = 0x1
	MH_DYLDLINK = 0x4
	MH_TWOLEVEL = 0x80
	MH_PIE      = 0x200000

	VM_PROT_READ    = 0x1
	VM_PROT_WRITE   = 0x2
	VM_PROT_EXECUTE = 0x4

	// Platform types for LC_BUILD_VERSION
	PLATFORM_MACOS = 0x1
)

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
	MachHeader64Size     = 32
	SegmentCommand64Size = 72
	Section64Size        = 80
	EntryPointCmdSize    = 24
	DylinkerCmdBaseSize  = 12 // Without path
	DylibCmdBaseSize     = 24 // Without path
	UUIDCmdSize          = 24
	BuildVersionCmdSize  = 32 // With 1 tool entry
	SourceVersionCmdSize = 16
	LinkeditDataCmdSize  = 16 // For chained fixups, exports trie, etc.
	SymtabCmdSize        = 24
	DysymtabCmdSize      = 80
	CodeSignatureCmdSize = 16
	BuildToolVersionSize = 8
	Nlist64Size          = 16
	FunctionStartsSize   = 8
	StringTableSize      = 16
	ExportsTrieSize      = 48
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

	// SourceVersionValue is the source version (1.0.0.0.0)
	SourceVersionValue = 0x0001000000000000
)

// Section flags
const (
	// SectionFlagPureInstructions marks a section as containing only machine instructions
	SectionFlagPureInstructions = 0x80000400
)

// Dyld paths
const (
	DylinkerPath  = "/usr/lib/dyld"
	LibSystemPath = "/usr/lib/libSystem.B.dylib"
)

// machHeader64 represents the Mach-O file header
type machHeader64 struct {
	Magic      uint32
	CPUType    uint32
	CPUSubtype uint32
	FileType   uint32
	NCmds      uint32
	SizeofCmds uint32
	Flags      uint32
	Reserved   uint32
}

// segmentCommand64 represents a segment load command
type segmentCommand64 struct {
	Cmd      uint32
	Cmdsize  uint32
	Segname  [16]byte
	VMAddr   uint64
	VMSize   uint64
	FileOff  uint64
	FileSize uint64
	MaxProt  uint32
	InitProt uint32
	NSects   uint32
	Flags    uint32
}

// section64 represents a section within a segment
type section64 struct {
	Sectname  [16]byte
	Segname   [16]byte
	Addr      uint64
	Size      uint64
	Offset    uint32
	Align     uint32
	Reloff    uint32
	Nreloc    uint32
	Flags     uint32
	Reserved1 uint32
	Reserved2 uint32
	Reserved3 uint32
}

// entryPointCommand represents the LC_MAIN load command
type entryPointCommand struct {
	Cmd       uint32
	Cmdsize   uint32
	EntryOff  uint64
	StackSize uint64
}

// dylinkerCommand represents the LC_LOAD_DYLINKER load command
type dylinkerCommand struct {
	Cmd     uint32
	Cmdsize uint32
	NameOff uint32 // Offset of the name string from start of command
}

// dylibCommand represents the LC_LOAD_DYLIB load command
type dylibCommand struct {
	Cmd                  uint32
	Cmdsize              uint32
	NameOff              uint32 // Offset of the library path from start of command
	Timestamp            uint32 // Library's build timestamp
	CurrentVersion       uint32 // Library's current version number
	CompatibilityVersion uint32 // Library's compatibility version number
}

// uuidCommand represents the LC_UUID load command
type uuidCommand struct {
	Cmd     uint32
	Cmdsize uint32
	UUID    [16]byte
}

// buildVersionCommand represents the LC_BUILD_VERSION load command
type buildVersionCommand struct {
	Cmd      uint32
	Cmdsize  uint32
	Platform uint32 // Platform (1 = macOS)
	Minos    uint32 // Minimum OS version (e.g., 11.0.0)
	Sdk      uint32 // SDK version (e.g., 15.0.0)
	Ntools   uint32 // Number of tool entries
}

// buildToolVersion represents a tool entry in LC_BUILD_VERSION
type buildToolVersion struct {
	Tool    uint32 // Tool identifier (e.g., 3 = LD)
	Version uint32 // Tool version
}

// sourceVersionCommand represents the LC_SOURCE_VERSION load command
type sourceVersionCommand struct {
	Cmd     uint32
	Cmdsize uint32
	Version uint64 // A.B.C.D.E packed as a24.b10.c10.d10.e10
}

// codeSignatureCommand represents the LC_CODE_SIGNATURE load command
type codeSignatureCommand struct {
	Cmd      uint32
	Cmdsize  uint32
	DataOff  uint32 // File offset of signature data
	DataSize uint32 // Size of signature data
}

// linkeditDataCommand represents LC_DYLD_CHAINED_FIXUPS and similar commands
type linkeditDataCommand struct {
	Cmd      uint32
	Cmdsize  uint32
	DataOff  uint32 // File offset of data in __LINKEDIT
	DataSize uint32 // Size of data
}

// symtabCommand represents the LC_SYMTAB load command
type symtabCommand struct {
	Cmd     uint32
	Cmdsize uint32
	Symoff  uint32 // File offset of symbol table
	Nsyms   uint32 // Number of symbol table entries
	Stroff  uint32 // File offset of string table
	Strsize uint32 // Size of string table
}

// dysymtabCommand represents the LC_DYSYMTAB load command
type dysymtabCommand struct {
	Cmd            uint32
	Cmdsize        uint32
	Ilocalsym      uint32 // Index of first local symbol
	Nlocalsym      uint32 // Number of local symbols
	Iextdefsym     uint32 // Index of first external defined symbol
	Nextdefsym     uint32 // Number of external defined symbols
	Iundefsym      uint32 // Index of first undefined symbol
	Nundefsym      uint32 // Number of undefined symbols
	Tocoff         uint32 // File offset of table of contents
	Ntoc           uint32 // Number of entries in TOC
	Modtaboff      uint32 // File offset of module table
	Nmodtab        uint32 // Number of entries in module table
	Extrefsymoff   uint32 // File offset of external reference table
	Nextrefsyms    uint32 // Number of entries in external reference table
	Indirectsymoff uint32 // File offset of indirect symbol table
	Nindirectsyms  uint32 // Number of entries in indirect symbol table
	Extreloff      uint32 // File offset of external relocation entries
	Nextrel        uint32 // Number of external relocation entries
	Locreloff      uint32 // File offset of local relocation entries
	Nlocrel        uint32 // Number of local relocation entries
}

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

// executableLayout holds all computed offsets and sizes for a Mach-O executable.
type executableLayout struct {
	// Header and load commands
	headerSize   uint64
	loadCmdsSize uint64
	numLoadCmds  uint32

	// __TEXT segment
	codeOffset          uint64
	codeSize            uint64
	textSegmentFileOff  uint64
	textSegmentFileSize uint64
	textVMAddr          uint64
	textVMSize          uint64

	// __DATA segment (zero if no data)
	hasDataSection      bool
	dataOffset          uint64
	dataSize            uint64
	dataSegmentFileSize uint64
	dataVMAddr          uint64
	dataVMSize          uint64

	// __LINKEDIT segment
	linkeditOffset uint64
	linkeditSize   uint64
	linkeditVMAddr uint64
	linkeditVMSize uint64

	// Linkedit contents
	chainedFixupsOffset  uint64
	chainedFixupsSize    uint64
	exportsTrieOffset    uint64
	exportsTrieSize      uint64
	symtabOffset         uint64
	symtabSize           uint64
	stringTableOffset    uint64
	stringTableSize      uint64
	functionStartsOffset uint64
	functionStartsSize   uint64

	// Code signature
	signatureOffset uint64
	signatureSize   uint64

	// Dynamic library command sizes (for path alignment)
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
	layout.loadCmdsSize = SegmentCommand64Size + // __PAGEZERO
		SegmentCommand64Size + Section64Size + // __TEXT + __text
		SegmentCommand64Size + // __LINKEDIT
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

	// Calculate chained fixups size (without modifying data)
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	chainedFixupsData := generateChainedFixupsWithRelocations(dataCopy, relocations, layout.dataVMAddr, layout.textVMAddr)
	layout.chainedFixupsSize = uint64(len(chainedFixupsData))
	layout.chainedFixupsOffset = layout.linkeditOffset

	layout.exportsTrieSize = ExportsTrieSize
	layout.exportsTrieOffset = layout.chainedFixupsOffset + layout.chainedFixupsSize

	layout.symtabSize = Nlist64Size // 1 symbol
	layout.symtabOffset = layout.exportsTrieOffset + layout.exportsTrieSize

	layout.stringTableSize = StringTableSize
	layout.stringTableOffset = layout.symtabOffset + layout.symtabSize

	layout.functionStartsSize = FunctionStartsSize
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

// loadCommands holds all Mach-O load commands for an executable.
type loadCommands struct {
	header         machHeader64
	pagezero       segmentCommand64
	textSegment    segmentCommand64
	textSection    section64
	dataSegment    segmentCommand64
	dataSection    section64
	linkedit       segmentCommand64
	dylinker       dylinkerCommand
	dylinkerPath   []byte
	dylib          dylibCommand
	dylibPath      []byte
	entryPoint     entryPointCommand
	uuid           uuidCommand
	buildVersion   buildVersionCommand
	buildTool      buildToolVersion
	sourceVersion  sourceVersionCommand
	chainedFixups  linkeditDataCommand
	exportsTrie    linkeditDataCommand
	symtab         symtabCommand
	dysymtab       dysymtabCommand
	functionStarts linkeditDataCommand
	dataInCode     linkeditDataCommand
	codeSignature  codeSignatureCommand
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
		Version: SourceVersionValue,
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

// generateMinimalChainedFixups creates a minimal chained fixups data structure
// This is required by LC_DYLD_CHAINED_FIXUPS for modern macOS
func generateMinimalChainedFixups() []byte {
	// Chained fixups header structure (matches working binary exactly)
	// 56 bytes total
	buf := make([]byte, 56)

	// dyld_chained_fixups_header
	binary.LittleEndian.PutUint32(buf[0:4], 0)    // fixups_version = 0
	binary.LittleEndian.PutUint32(buf[4:8], 32)   // starts_offset = 0x20
	binary.LittleEndian.PutUint32(buf[8:12], 48)  // imports_offset = 0x30
	binary.LittleEndian.PutUint32(buf[12:16], 48) // symbols_offset = 0x30
	binary.LittleEndian.PutUint32(buf[16:20], 0)  // imports_count = 0
	binary.LittleEndian.PutUint32(buf[20:24], 1)  // imports_format = 1 (DYLD_CHAINED_IMPORT)
	// bytes 24-31: zeros (padding)

	// At offset 32: dyld_chained_starts_in_image
	binary.LittleEndian.PutUint32(buf[32:36], 3) // seg_count = 3 (PAGEZERO, TEXT, LINKEDIT)
	// bytes 36-48: zeros (no fixups in any segment)

	// bytes 48-55: zeros (empty imports/symbols)

	return buf
}

// Chained fixup constants
const (
	DYLD_CHAINED_PTR_64_OFFSET = 6 // pointer_format for 64-bit rebase-only fixups
)

// generateChainedFixupsWithRelocations creates chained fixups data for data section relocations.
// It modifies the data bytes in-place to use chained pointer format (DYLD_CHAINED_PTR_64_OFFSET)
// and generates proper chained fixups metadata for dyld to process at load time.
//
// The chained pointer format encodes:
//   - bits 0-35:  target offset from image base
//   - bits 36-43: high8 (high 8 bits of target, for addresses > 36 bits)
//   - bits 44-50: reserved (must be 0)
//   - bits 51-62: next (delta to next pointer in chain, in 4-byte units)
//   - bit 63:     bind (0 for rebase, 1 for bind)
func generateChainedFixupsWithRelocations(data []byte, relocations []DataRelocation, dataVMAddr uint64, imageBase uint64) []byte {
	// If no relocations, return minimal fixups
	if len(relocations) == 0 {
		return generateMinimalChainedFixups()
	}

	// Sort relocations by offset to build the chain correctly
	sortedRelocs := make([]DataRelocation, len(relocations))
	copy(sortedRelocs, relocations)
	sortDataRelocations(sortedRelocs)

	// Group relocations by page (16KB pages on ARM64 macOS)
	pageRelocs := make(map[uint64][]DataRelocation)
	for _, reloc := range sortedRelocs {
		pageNum := reloc.Offset / PageSize
		pageRelocs[pageNum] = append(pageRelocs[pageNum], reloc)
	}

	// Find max page number
	var maxPage uint64
	for pageNum := range pageRelocs {
		if pageNum > maxPage {
			maxPage = pageNum
		}
	}
	pageCount := int(maxPage + 1)

	// Calculate sizes for the chained fixups structure
	// Header: 32 bytes (dyld_chained_fixups_header with padding)
	// starts_in_image: 4 + 4*4 = 20 bytes (seg_count + 4 segment offsets)
	// starts_in_segment for DATA: 22 + 2*pageCount bytes (struct size)
	headerSize := 32
	startsInImageSize := 4 + 4*4 // seg_count + 4 segment offsets

	// dyld_chained_starts_in_segment structure:
	// size (4) + page_size (2) + pointer_format (2) + segment_offset (8) +
	// max_valid_pointer (4) + page_count (2) + page_start[pageCount] (2*n)
	// = 22 + 2*pageCount bytes
	startsInSegmentSize := 22 + 2*pageCount

	// Align startsInSegmentSize to 4 bytes for the structure padding
	startsInSegmentSizeAligned := startsInSegmentSize
	if startsInSegmentSizeAligned%4 != 0 {
		startsInSegmentSizeAligned += 4 - (startsInSegmentSizeAligned % 4)
	}

	totalSize := headerSize + startsInImageSize + startsInSegmentSizeAligned
	// Add padding to align to 8 bytes
	if totalSize%8 != 0 {
		totalSize += 8 - (totalSize % 8)
	}

	buf := make([]byte, totalSize)

	// Offsets within the buffer
	startsOffset := uint32(headerSize)
	dataSegmentStartsOffset := uint32(startsInImageSize) // Relative to starts_in_image
	importsOffset := uint32(totalSize - 8)
	symbolsOffset := importsOffset

	// dyld_chained_fixups_header (offset 0)
	binary.LittleEndian.PutUint32(buf[0:4], 0)            // fixups_version = 0
	binary.LittleEndian.PutUint32(buf[4:8], startsOffset) // starts_offset
	binary.LittleEndian.PutUint32(buf[8:12], importsOffset)
	binary.LittleEndian.PutUint32(buf[12:16], symbolsOffset)
	binary.LittleEndian.PutUint32(buf[16:20], 0) // imports_count = 0
	binary.LittleEndian.PutUint32(buf[20:24], 1) // imports_format = 1 (DYLD_CHAINED_IMPORT)
	// bytes 24-31: zeros (symbols_format = 0, padding)

	// dyld_chained_starts_in_image (at startsOffset)
	off := int(startsOffset)
	binary.LittleEndian.PutUint32(buf[off:off+4], 4) // seg_count = 4 (PAGEZERO, TEXT, DATA, LINKEDIT)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:off+4], 0) // seg_info_offset[0] = 0 (PAGEZERO - no fixups)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:off+4], 0) // seg_info_offset[1] = 0 (TEXT - no fixups)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:off+4], dataSegmentStartsOffset) // seg_info_offset[2] = offset to DATA starts
	off += 4
	binary.LittleEndian.PutUint32(buf[off:off+4], 0) // seg_info_offset[3] = 0 (LINKEDIT - no fixups)
	off += 4

	// dyld_chained_starts_in_segment for DATA (follows starts_in_image)
	segmentStartsOff := int(startsOffset) + int(dataSegmentStartsOffset)
	// size field is the actual structure size (not aligned)
	binary.LittleEndian.PutUint32(buf[segmentStartsOff:segmentStartsOff+4], uint32(startsInSegmentSize))  // size
	binary.LittleEndian.PutUint16(buf[segmentStartsOff+4:segmentStartsOff+6], PageSize)                   // page_size
	binary.LittleEndian.PutUint16(buf[segmentStartsOff+6:segmentStartsOff+8], DYLD_CHAINED_PTR_64_OFFSET) // pointer_format
	// segment_offset: VM offset from image base to the data segment
	// For ARM64 macOS, the __DATA segment typically starts at 0x4000 (16KB) after __TEXT
	binary.LittleEndian.PutUint64(buf[segmentStartsOff+8:segmentStartsOff+16], dataVMAddr-imageBase) // segment_offset
	binary.LittleEndian.PutUint32(buf[segmentStartsOff+16:segmentStartsOff+20], 0)                   // max_valid_pointer (0 for 64-bit)
	binary.LittleEndian.PutUint16(buf[segmentStartsOff+20:segmentStartsOff+22], uint16(pageCount))   // page_count

	// page_start array
	pageStartOff := segmentStartsOff + 22
	for i := 0; i < pageCount; i++ {
		relocs, hasRelocs := pageRelocs[uint64(i)]
		if hasRelocs && len(relocs) > 0 {
			// First relocation's offset within the page
			firstOffset := relocs[0].Offset % PageSize
			binary.LittleEndian.PutUint16(buf[pageStartOff+i*2:pageStartOff+i*2+2], uint16(firstOffset))
		} else {
			// DYLD_CHAINED_PTR_START_NONE = 0xFFFF
			binary.LittleEndian.PutUint16(buf[pageStartOff+i*2:pageStartOff+i*2+2], 0xFFFF)
		}
	}

	// Now modify the data bytes to use chained pointer format
	// and link the pointers in each page together
	for pageNum, relocs := range pageRelocs {
		for i, reloc := range relocs {
			// Calculate the target offset from image base
			// The original TargetAddr is an absolute VM address
			targetOffset := reloc.TargetAddr - imageBase

			// Calculate delta to next pointer (in 4-byte units)
			var nextDelta uint64 = 0
			if i+1 < len(relocs) {
				nextReloc := relocs[i+1]
				// Both should be in the same page
				if nextReloc.Offset/PageSize == pageNum {
					delta := nextReloc.Offset - reloc.Offset
					nextDelta = delta / 4 // Convert to 4-byte units
				}
			}

			// Encode as DYLD_CHAINED_PTR_64_OFFSET
			// bits 0-35:  target (36 bits)
			// bits 36-43: high8 (8 bits)
			// bits 44-50: reserved (7 bits, must be 0)
			// bits 51-62: next (12 bits)
			// bit 63:     bind (1 bit, 0 for rebase)
			var encoded uint64
			encoded |= targetOffset & 0xFFFFFFFFF        // bits 0-35: target (36 bits)
			encoded |= (targetOffset >> 36) << 36 & 0xFF // bits 36-43: high8 (but target should fit in 36 bits)
			// bits 44-50: reserved = 0
			encoded |= (nextDelta & 0xFFF) << 51 // bits 51-62: next
			// bit 63: bind = 0

			// Write the encoded pointer back to data
			binary.LittleEndian.PutUint64(data[reloc.Offset:reloc.Offset+8], encoded)
		}
	}

	return buf
}

// sortDataRelocations sorts relocations by offset in ascending order
func sortDataRelocations(relocs []DataRelocation) {
	// Simple insertion sort (relocations list is typically small)
	for i := 1; i < len(relocs); i++ {
		key := relocs[i]
		j := i - 1
		for j >= 0 && relocs[j].Offset > key.Offset {
			relocs[j+1] = relocs[j]
			j--
		}
		relocs[j+1] = key
	}
}

// generateMinimalExportsTrie creates a minimal exports trie for _start and _mh_execute_header
// This format is used by LC_DYLD_EXPORTS_TRIE
// entryOffset is the file offset to the entry point (code start)
func generateMinimalExportsTrie(entryOffset uint64) []byte {
	// The exports trie encodes:
	// - _mh_execute_header at address 0 (relative to __TEXT base)
	// - _start at the entry point offset (relative to __TEXT base)
	//
	// Trie structure:
	// - Root node (no terminal): 1 edge "_" -> child
	// - Child node: 2 edges "mh_execute_header\0" and "start\0"
	// - Terminal for _mh_execute_header: flags=0, address=0
	// - Terminal for _start: flags=0, address=entryOffset

	// Encode start address as ULEB128
	startAddrULEB := encodeULEB128(entryOffset)

	// Build terminal nodes first to know their sizes
	// Terminal for _mh_execute_header: terminalSize=2 (flags + addr), flags=0, addr=0, edgeCount=0
	mhTerminal := []byte{0x02, 0x00, 0x00, 0x00}

	// Terminal for _start: terminalSize, flags=0, addr=ULEB, edgeCount=0
	startTerminalSize := byte(1 + len(startAddrULEB)) // flags + addr
	startTerminal := append([]byte{startTerminalSize, 0x00}, startAddrULEB...)
	startTerminal = append(startTerminal, 0x00) // edgeCount=0

	// Calculate sizes and offsets
	// Root node: terminalSize(1) + edgeCount(1) + "_\0"(2) + childOffset(1) = 5 bytes
	rootNodeSize := 5

	// Child node: terminalSize(1) + edgeCount(1) + "mh_execute_header\0"(18) + offset(1)
	//             + "start\0"(6) + offset(1) = 28 bytes
	mhSuffix := "mh_execute_header"
	startSuffix := "start"
	childNodeSize := 2 + len(mhSuffix) + 1 + 1 + len(startSuffix) + 1 + 1

	// Offsets from trie start
	childOffset := rootNodeSize
	mhOffset := rootNodeSize + childNodeSize
	startOffset := mhOffset + len(mhTerminal)

	// Build root node
	rootNode := []byte{
		0x00,                  // terminalSize = 0 (not a terminal)
		0x01,                  // edgeCount = 1
		'_', 0x00,             // edge label "_\0"
		byte(childOffset),     // child node offset
	}

	// Build child node
	childNode := []byte{0x00, 0x02} // terminalSize=0, edgeCount=2
	childNode = append(childNode, []byte(mhSuffix)...)
	childNode = append(childNode, 0x00)           // null terminator
	childNode = append(childNode, byte(mhOffset)) // offset to mh terminal
	childNode = append(childNode, []byte(startSuffix)...)
	childNode = append(childNode, 0x00)              // null terminator
	childNode = append(childNode, byte(startOffset)) // offset to start terminal

	// Assemble full trie
	var buf []byte
	buf = append(buf, rootNode...)
	buf = append(buf, childNode...)
	buf = append(buf, mhTerminal...)
	buf = append(buf, startTerminal...)

	// Pad to 8-byte alignment
	for len(buf)%8 != 0 {
		buf = append(buf, 0x00)
	}

	return buf
}

// encodeULEB128 encodes a uint64 as ULEB128
func encodeULEB128(value uint64) []byte {
	var result []byte
	for {
		b := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		result = append(result, b)
		if value == 0 {
			break
		}
	}
	return result
}

// generateFunctionStarts generates the function starts data
// This is a ULEB128-encoded list of function start offsets relative to __TEXT
func generateFunctionStarts(entryOffset uint64) []byte {
	buf := make([]byte, 8)
	// First (and only) function starts at entryOffset
	// ULEB128 encode the offset
	if entryOffset <= 0x7f {
		buf[0] = byte(entryOffset)
	} else {
		buf[0] = byte(entryOffset&0x7f) | 0x80
		buf[1] = byte((entryOffset >> 7) & 0x7f)
		if entryOffset >= 0x4000 {
			buf[1] |= 0x80
			buf[2] = byte((entryOffset >> 14) & 0x7f)
		}
	}
	// Terminator (0) is already zero-initialized
	return buf
}

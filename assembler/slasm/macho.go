package slasm

import (
	"encoding/binary"
	"os"
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
	// TODO: Implement Mach-O object file generation
	// Structure:
	// 1. mach_header_64
	// 2. Load commands:
	//    - LC_SEGMENT_64 for __TEXT segment
	//    - LC_SEGMENT_64 for __DATA segment
	//    - LC_SYMTAB for symbol table
	// 3. Section data (__text, __data)
	// 4. Relocations
	// 5. Symbol table
	// 6. String table

	return nil
}

// WriteExecutable writes a Mach-O executable
func (w *MachOWriter) WriteExecutable(outputPath string, code []byte, data []byte, symbols *SymbolTable, entryPoint string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Calculate offsets and sizes
	headerSize := uint64(32)        // mach_header_64
	segmentCmdSize := uint64(72)    // segment_command_64
	sectionHeaderSize := uint64(80) // section_64
	entryPointCmdSize := uint64(24) // entry_point_command
	dylinkerPath := "/usr/lib/dyld"
	dylinkerCmdSize := uint64(12 + len(dylinkerPath) + 1) // command header + path + null
	dylinkerCmdSize = (dylinkerCmdSize + 7) & ^uint64(7)  // Align to 8 bytes

	dylibPath := "/usr/lib/libSystem.B.dylib"
	dylibCmdSize := uint64(24 + len(dylibPath) + 1) // LC_LOAD_DYLIB header (24 bytes) + path + null
	dylibCmdSize = (dylibCmdSize + 7) & ^uint64(7)  // Align to 8 bytes

	uuidCmdSize := uint64(24)             // LC_UUID command (8 + 16 bytes)
	buildVersionCmdSize := uint64(32)     // LC_BUILD_VERSION command (with 1 tool entry)
	sourceVersionCmdSize := uint64(16)    // LC_SOURCE_VERSION command
	chainedFixupsCmdSize := uint64(16)    // LC_DYLD_CHAINED_FIXUPS command
	exportsTrieCmdSize := uint64(16)      // LC_DYLD_EXPORTS_TRIE command
	symtabCmdSize := uint64(24)           // LC_SYMTAB command
	dysymtabCmdSize := uint64(80)         // LC_DYSYMTAB command
	functionStartsCmdSize := uint64(16)   // LC_FUNCTION_STARTS command
	dataInCodeCmdSize := uint64(16)       // LC_DATA_IN_CODE command

	// Add __PAGEZERO segment (important for memory protection)
	pagezeroSize := uint64(72) // segment_command_64 without sections

	// Add __LINKEDIT segment (required for code signatures)
	linkeditSegmentSize := uint64(72) // segment_command_64 without sections

	// Reserve space for LC_CODE_SIGNATURE which codesign will add
	codeSignatureCmdSize := uint64(16) // LC_CODE_SIGNATURE load command size
	// Calculate total size of all load commands (including symbol tables)
	loadCmdsSize := pagezeroSize + segmentCmdSize + sectionHeaderSize + linkeditSegmentSize +
		dylinkerCmdSize + dylibCmdSize + entryPointCmdSize + uuidCmdSize +
		buildVersionCmdSize + sourceVersionCmdSize +
		chainedFixupsCmdSize + exportsTrieCmdSize +
		symtabCmdSize + dysymtabCmdSize +
		functionStartsCmdSize + dataInCodeCmdSize +
		codeSignatureCmdSize // Reserved space for codesign

	// Place code right after load commands, aligned to 8 bytes
	// (like the system linker does - no page alignment for code offset)
	codeOffset := headerSize + loadCmdsSize
	codeOffset = ((codeOffset + 7) / 8) * 8 // Align to 8 bytes

	// Calculate code size
	codeSize := uint64(len(code))

	// Virtual memory base address
	vmAddr := uint64(0x100000000) // Standard base for ARM64 executables

	// Calculate __TEXT segment file size and VM size
	// Match the system linker layout: use 0x4000 (16KB) minimum for __TEXT
	// This ensures proper alignment and matches macOS conventions
	textSegmentFileSize := uint64(0x4000) // 16KB like system linker
	if (codeOffset + codeSize) > textSegmentFileSize {
		textSegmentFileSize = ((codeOffset + codeSize + 0xFFF) / 0x1000) * 0x1000
	}
	vmSize := textSegmentFileSize // VM size matches file size

	// Calculate __LINKEDIT segment location
	// It comes right after the __TEXT segment in both file and VM
	linkeditOffset := textSegmentFileSize                   // Right after __TEXT in file
	linkeditVMAddr := vmAddr + vmSize                       // Right after __TEXT in VM

	// Reserve space for code signature (8KB should be enough for a simple binary)
	linkeditSize := uint64(0x2000) // 8KB for signature data
	linkeditVMSize := linkeditSize

	// Build Mach-O header
	// Load commands: __PAGEZERO, __TEXT, __LINKEDIT (3 segments)
	// + LC_LOAD_DYLINKER, LC_LOAD_DYLIB, LC_MAIN, LC_UUID
	// + LC_BUILD_VERSION, LC_SOURCE_VERSION
	// + LC_DYLD_CHAINED_FIXUPS, LC_DYLD_EXPORTS_TRIE
	// + LC_SYMTAB, LC_DYSYMTAB
	// + LC_FUNCTION_STARTS, LC_DATA_IN_CODE
	// Total: 15 load commands (codesign adds LC_CODE_SIGNATURE, making it 16)
	// SizeofCmds includes space reserved for LC_CODE_SIGNATURE
	header := machHeader64{
		Magic:      MH_MAGIC_64,
		CPUType:    CPU_TYPE_ARM64,
		CPUSubtype: CPU_SUBTYPE_ARM64,
		FileType:   MH_EXECUTE,
		NCmds:      15, // Not counting LC_CODE_SIGNATURE yet
		SizeofCmds: uint32(loadCmdsSize - codeSignatureCmdSize), // Our actual commands size
		Flags:      MH_NOUNDEFS | MH_DYLDLINK | MH_TWOLEVEL | MH_PIE,
		Reserved:   0,
	}

	// Build __TEXT segment command with __text section
	// The __TEXT segment should start at file offset 0 and include the header and load commands
	var segname [16]byte
	copy(segname[:], "__TEXT")

	segment := segmentCommand64{
		Cmd:      LC_SEGMENT_64,
		Cmdsize:  uint32(segmentCmdSize + sectionHeaderSize),
		Segname:  segname,
		VMAddr:   vmAddr,
		VMSize:   vmSize,
		FileOff:  0,                       // Start at beginning of file
		FileSize: textSegmentFileSize,     // Include header, load commands, and code
		MaxProt:  VM_PROT_READ | VM_PROT_EXECUTE,
		InitProt: VM_PROT_READ | VM_PROT_EXECUTE,
		NSects:   1,
		Flags:    0,
	}

	// Build __text section
	var sectname [16]byte
	copy(sectname[:], "__text")

	section := section64{
		Sectname:  sectname,
		Segname:   segname,
		Addr:      vmAddr + codeOffset, // VM address where code will be loaded
		Size:      codeSize,
		Offset:    uint32(codeOffset),
		Align:     2, // 2^2 = 4 byte alignment
		Reloff:    0,
		Nreloc:    0,
		Flags:     0x80000400, // S_REGULAR | S_ATTR_PURE_INSTRUCTIONS | S_ATTR_SOME_INSTRUCTIONS
		Reserved1: 0,
		Reserved2: 0,
		Reserved3: 0,
	}

	// Build LC_MAIN command (entry point)
	entryCmd := entryPointCommand{
		Cmd:       LC_MAIN,
		Cmdsize:   uint32(entryPointCmdSize),
		EntryOff:  codeOffset, // Entry point file offset
		StackSize: 0,          // Use default stack size
	}

	// Build __PAGEZERO segment (protects against null pointer dereferences)
	var pagezeroName [16]byte
	copy(pagezeroName[:], "__PAGEZERO")

	pagezero := segmentCommand64{
		Cmd:      LC_SEGMENT_64,
		Cmdsize:  uint32(pagezeroSize),
		Segname:  pagezeroName,
		VMAddr:   0,
		VMSize:   0x100000000, // 4GB
		FileOff:  0,
		FileSize: 0,
		MaxProt:  0,
		InitProt: 0,
		NSects:   0,
		Flags:    0,
	}

	// Build __LINKEDIT segment (for code signatures and other link-edit data)
	var linkeditName [16]byte
	copy(linkeditName[:], "__LINKEDIT")

	linkedit := segmentCommand64{
		Cmd:      LC_SEGMENT_64,
		Cmdsize:  uint32(linkeditSegmentSize),
		Segname:  linkeditName,
		VMAddr:   linkeditVMAddr,
		VMSize:   linkeditVMSize,
		FileOff:  linkeditOffset,
		FileSize: linkeditSize,
		MaxProt:  VM_PROT_READ,
		InitProt: VM_PROT_READ,
		NSects:   0,
		Flags:    0,
	}

	// Build dylinker load command
	dylinkerCmd := dylinkerCommand{
		Cmd:     LC_LOAD_DYLINKER,
		Cmdsize: uint32(dylinkerCmdSize),
		NameOff: 12, // Offset from start of command (after cmd, cmdsize, nameoff)
	}

	// Build dylib load command for libSystem
	dylibCmd := dylibCommand{
		Cmd:                  LC_LOAD_DYLIB,
		Cmdsize:              uint32(dylibCmdSize),
		NameOff:              24,         // Offset from start of command (after the header fields)
		Timestamp:            2,          // Standard value
		CurrentVersion:       0x050c6400, // 1292.100.0 (common libSystem version)
		CompatibilityVersion: 0x00010000, // 1.0.0
	}

	// Build LC_UUID command
	// Generate a simple UUID (in production, this should be unique)
	// For now, we'll use a deterministic UUID based on code content
	uuidCmd := uuidCommand{
		Cmd:     LC_UUID,
		Cmdsize: uint32(uuidCmdSize),
		UUID: [16]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		},
	}

	// Build LC_BUILD_VERSION command
	// Platform: macOS (1)
	// Minimum OS: 11.0.0 (Big Sur) encoded as 11 << 16 | 0 << 8 | 0
	// SDK: 15.0.0 (macOS 15) encoded as 15 << 16 | 0 << 8 | 0
	buildVersionCmd := buildVersionCommand{
		Cmd:      LC_BUILD_VERSION,
		Cmdsize:  uint32(buildVersionCmdSize),
		Platform: PLATFORM_MACOS,
		Minos:    0x000b0000, // 11.0.0
		Sdk:      0x000f0000, // 15.0.0
		Ntools:   1,          // Include 1 tool entry
	}

	// Tool entry: LD (linker) version 1167.0
	buildToolEntry := buildToolVersion{
		Tool:    3,          // 3 = LD (linker)
		Version: 0x048f0000, // 1167.0 encoded as (1167 << 16)
	}

	// Build LC_SOURCE_VERSION command
	// Version 1.0.0.0.0 encoded as (1 << 40)
	sourceVersionCmd := sourceVersionCommand{
		Cmd:     LC_SOURCE_VERSION,
		Cmdsize: uint32(sourceVersionCmdSize),
		Version: 0x0001000000000000, // Version 1.0.0.0.0
	}

	// Build LC_DYLD_CHAINED_FIXUPS command
	// Modern format required by newer macOS versions
	chainedFixupsDataSize := uint64(56) // Minimal chained fixups data structure
	chainedFixupsCmd := linkeditDataCommand{
		Cmd:      LC_DYLD_CHAINED_FIXUPS,
		Cmdsize:  uint32(chainedFixupsCmdSize),
		DataOff:  uint32(linkeditOffset),
		DataSize: uint32(chainedFixupsDataSize),
	}

	// Build LC_DYLD_EXPORTS_TRIE command
	exportsTrieSize := uint64(48) // Minimal exports trie with _start and _mh_execute_header
	exportsTrieCmd := linkeditDataCommand{
		Cmd:      LC_DYLD_EXPORTS_TRIE,
		Cmdsize:  uint32(exportsTrieCmdSize),
		DataOff:  uint32(linkeditOffset + chainedFixupsDataSize),
		DataSize: uint32(exportsTrieSize),
	}

	// Build LC_SYMTAB command
	// Place symbol table after chained fixups and exports trie in __LINKEDIT
	symtabOffset := linkeditOffset + chainedFixupsDataSize + exportsTrieSize
	// Minimal symbol table with one symbol (_start)
	numSymbols := uint32(1)
	symbolSize := uint64(16)                         // nlist_64 size
	symbolDataSize := uint64(numSymbols) * symbolSize
	stringTableSize := uint64(16) // "_start\0" + padding
	symtabCmd := symtabCommand{
		Cmd:     LC_SYMTAB,
		Cmdsize: uint32(symtabCmdSize),
		Symoff:  uint32(symtabOffset),
		Nsyms:   numSymbols,
		Stroff:  uint32(symtabOffset + symbolDataSize),
		Strsize: uint32(stringTableSize),
	}

	// Build LC_DYSYMTAB command
	dysymtabCmd := dysymtabCommand{
		Cmd:            LC_DYSYMTAB,
		Cmdsize:        uint32(dysymtabCmdSize),
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

	// Build LC_FUNCTION_STARTS command
	// Function starts data comes after string table
	functionStartsOffset := symtabOffset + symbolDataSize + stringTableSize
	functionStartsDataSize := uint64(8) // Minimal function starts data (just points to _start)
	functionStartsCmd := linkeditDataCommand{
		Cmd:      LC_FUNCTION_STARTS,
		Cmdsize:  uint32(functionStartsCmdSize),
		DataOff:  uint32(functionStartsOffset),
		DataSize: uint32(functionStartsDataSize),
	}

	// Build LC_DATA_IN_CODE command
	// Data in code comes after function starts, but we have no data in code (size 0)
	dataInCodeOffset := functionStartsOffset + functionStartsDataSize
	dataInCodeCmd := linkeditDataCommand{
		Cmd:      LC_DATA_IN_CODE,
		Cmdsize:  uint32(dataInCodeCmdSize),
		DataOff:  uint32(dataInCodeOffset),
		DataSize: 0, // No data in code
	}

	// Note: LC_CODE_SIGNATURE will be added by codesign
	// We leave space for it in the load commands area (16 bytes) and in __LINKEDIT

	// Write everything to file
	if err := writeStruct(file, &header); err != nil {
		return err
	}
	// Write __PAGEZERO first
	if err := writeStruct(file, &pagezero); err != nil {
		return err
	}
	if err := writeStruct(file, &segment); err != nil {
		return err
	}
	if err := writeStruct(file, &section); err != nil {
		return err
	}
	// Write __LINKEDIT segment
	if err := writeStruct(file, &linkedit); err != nil {
		return err
	}
	// Write dylinker command
	if err := writeStruct(file, &dylinkerCmd); err != nil {
		return err
	}
	// Write dylinker path (null-terminated, padded to alignment)
	dylinkerPathBytes := make([]byte, dylinkerCmdSize-12)
	copy(dylinkerPathBytes, dylinkerPath)
	if _, err := file.Write(dylinkerPathBytes); err != nil {
		return err
	}

	// Write dylib command
	if err := writeStruct(file, &dylibCmd); err != nil {
		return err
	}
	// Write dylib path (null-terminated, padded to alignment)
	dylibPathBytes := make([]byte, dylibCmdSize-24)
	copy(dylibPathBytes, dylibPath)
	if _, err := file.Write(dylibPathBytes); err != nil {
		return err
	}

	if err := writeStruct(file, &entryCmd); err != nil {
		return err
	}
	// Write LC_UUID command
	if err := writeStruct(file, &uuidCmd); err != nil {
		return err
	}
	// Write LC_BUILD_VERSION command
	if err := writeStruct(file, &buildVersionCmd); err != nil {
		return err
	}
	// Write tool entry for LC_BUILD_VERSION
	if err := writeStruct(file, &buildToolEntry); err != nil {
		return err
	}
	// Write LC_SOURCE_VERSION command
	if err := writeStruct(file, &sourceVersionCmd); err != nil {
		return err
	}
	// Write LC_DYLD_CHAINED_FIXUPS command
	if err := writeStruct(file, &chainedFixupsCmd); err != nil {
		return err
	}
	// Write LC_DYLD_EXPORTS_TRIE command
	if err := writeStruct(file, &exportsTrieCmd); err != nil {
		return err
	}
	// Write LC_SYMTAB command
	if err := writeStruct(file, &symtabCmd); err != nil {
		return err
	}
	// Write LC_DYSYMTAB command
	if err := writeStruct(file, &dysymtabCmd); err != nil {
		return err
	}
	// Write LC_FUNCTION_STARTS command
	if err := writeStruct(file, &functionStartsCmd); err != nil {
		return err
	}
	// Write LC_DATA_IN_CODE command
	if err := writeStruct(file, &dataInCodeCmd); err != nil {
		return err
	}
	// Note: We leave 16 bytes of space here for codesign to add LC_CODE_SIGNATURE

	// Print Mach-O structure information (only if verbose logging is enabled)
	w.logger.Printf("\nMach-O Structure:\n")
	w.logger.Printf("  Header:            size=%d bytes\n", headerSize)
	w.logger.Printf("  Load commands:     size=%d bytes, count=%d\n", loadCmdsSize, header.NCmds)
	w.logger.Printf("  Code offset:       0x%x (%d bytes)\n", codeOffset, codeOffset)
	w.logger.Printf("  Code size:         %d bytes\n", codeSize)
	w.logger.Printf("\nSegments:\n")
	w.logger.Printf("  __PAGEZERO:        vm=0x%x-0x%x (size=0x%x)\n", uint64(0), uint64(0x100000000), uint64(0x100000000))
	w.logger.Printf("  __TEXT:            vm=0x%x-0x%x (size=0x%x), file=0x%x-0x%x\n",
		vmAddr, vmAddr+vmSize, vmSize, uint64(0), textSegmentFileSize)
	w.logger.Printf("    __text section:  vm=0x%x-0x%x (size=0x%x), file=0x%x\n",
		vmAddr+codeOffset, vmAddr+codeOffset+codeSize, codeSize, codeOffset)
	w.logger.Printf("  __LINKEDIT:        vm=0x%x-0x%x (size=0x%x), file=0x%x\n",
		linkeditVMAddr, linkeditVMAddr+linkeditVMSize, linkeditVMSize, linkeditOffset)
	w.logger.Printf("\nEntry point:         0x%x (file offset 0x%x)\n", vmAddr+codeOffset, codeOffset)
	w.logger.Printf("Total file size:     %d bytes\n", linkeditOffset+linkeditSize)

	// Seek to code offset (leave room for codesign to add LC_CODE_SIGNATURE)
	if _, err := file.Seek(int64(codeOffset), 0); err != nil {
		return err
	}

	// Write code
	_, err = file.Write(code)
	if err != nil {
		return err
	}

	// Pad the __TEXT segment to match vmSize
	// The __TEXT segment starts at file offset 0 and should have filesize = vmSize
	currentPos := codeOffset + codeSize
	textPadding := vmSize - currentPos
	if textPadding > 0 {
		padding := make([]byte, textPadding)
		if _, err := file.Write(padding); err != nil {
			return err
		}
	}

	// Write __LINKEDIT segment data
	// First, write the chained fixups data (56 bytes)
	chainedFixupsData := generateMinimalChainedFixups()
	if _, err := file.Write(chainedFixupsData); err != nil {
		return err
	}

	// Write the exports trie data
	exportsTrieData := generateMinimalExportsTrie(codeOffset)
	if _, err := file.Write(exportsTrieData); err != nil {
		return err
	}

	// Write symbol table (nlist_64 entries)
	// For _start symbol at address 0 in __text section
	symbolEntry := make([]byte, 16)
	binary.LittleEndian.PutUint32(symbolEntry[0:4], 1) // n_strx = 1 (offset in string table, after initial \0)
	symbolEntry[4] = 0x0f                               // n_type = N_SECT | N_EXT (external symbol in a section)
	symbolEntry[5] = 1                                  // n_sect = 1 (__text section)
	binary.LittleEndian.PutUint16(symbolEntry[6:8], 0)  // n_desc = 0
	binary.LittleEndian.PutUint64(symbolEntry[8:16], vmAddr+codeOffset) // n_value = VM address of _start
	if _, err := file.Write(symbolEntry); err != nil {
		return err
	}

	// Write string table
	stringTable := make([]byte, 16)
	stringTable[0] = ' '                // Initial space (standard practice)
	copy(stringTable[1:], "_start\x00") // Symbol name + null terminator
	// Rest is padding (already zero)
	if _, err := file.Write(stringTable); err != nil {
		return err
	}

	// Write function starts data
	// ULEB128 encoding of offset from __TEXT segment to first function
	// For our case, the first (and only) function is at offset codeOffset
	functionStartsData := generateFunctionStarts(codeOffset)
	if _, err := file.Write(functionStartsData); err != nil {
		return err
	}

	// Pad the remaining __LINKEDIT space (codesign will use this for signature)
	totalLinkeditDataSize := chainedFixupsDataSize + exportsTrieSize + symbolDataSize + stringTableSize + functionStartsDataSize
	remainingLinkeditSize := linkeditSize - totalLinkeditDataSize
	linkeditPadding := make([]byte, remainingLinkeditSize)
	if _, err := file.Write(linkeditPadding); err != nil {
		return err
	}

	return nil
}

// Mach-O constants and structures

const (
	MH_MAGIC_64 = 0xfeedfacf
	MH_OBJECT   = 0x1
	MH_EXECUTE  = 0x2

	CPU_TYPE_ARM64    = 0x0100000c
	CPU_SUBTYPE_ARM64 = 0x00000000

	LC_SEGMENT_64           = 0x19
	LC_SYMTAB               = 0x2
	LC_DYSYMTAB             = 0xb
	LC_LOAD_DYLINKER        = 0xe
	LC_LOAD_DYLIB           = 0xc
	LC_UUID                 = 0x1b
	LC_BUILD_VERSION        = 0x32
	LC_SOURCE_VERSION       = 0x2a
	LC_CODE_SIGNATURE       = 0x1d
	LC_MAIN                 = 0x80000028
	LC_DYLD_CHAINED_FIXUPS  = 0x80000034
	LC_DYLD_EXPORTS_TRIE    = 0x80000033
	LC_DYLD_INFO_ONLY       = 0x80000022
	LC_FUNCTION_STARTS      = 0x26
	LC_DATA_IN_CODE         = 0x29

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

// dyldInfoCommand represents the LC_DYLD_INFO_ONLY load command
type dyldInfoCommand struct {
	Cmd          uint32
	Cmdsize      uint32
	RebaseOff    uint32 // File offset to rebase info
	RebaseSize   uint32 // Size of rebase info
	BindOff      uint32 // File offset to binding info
	BindSize     uint32 // Size of binding info
	WeakBindOff  uint32 // File offset to weak binding info
	WeakBindSize uint32 // Size of weak binding info
	LazyBindOff  uint32 // File offset to lazy binding info
	LazyBindSize uint32 // Size of lazy binding info
	ExportOff    uint32 // File offset to export info
	ExportSize   uint32 // Size of export info
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

// generateMinimalExportsTrie creates a minimal exports trie for _start and _mh_execute_header
// This format is used by LC_DYLD_EXPORTS_TRIE
// entryOffset is the file offset to the entry point (code start)
func generateMinimalExportsTrie(entryOffset uint64) []byte {
	// The exports trie encodes:
	// - _mh_execute_header at address 0 (relative to __TEXT base)
	// - start at the entry point offset (relative to __TEXT base)

	// Use exact bytes from working system binary, with adjusted offset for start
	// The entry offset is encoded at byte 14-15 as a ULEB128 value
	buf := []byte{
		0x00, 0x01, 0x5f, 0x00, 0x12, 0x00, 0x00, 0x00,
		0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, // byte 14-15 will be patched
		0x00, 0x00, 0x02, 0x5f, 0x6d, 0x68, 0x5f, 0x65,
		0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x5f, 0x68,
		0x65, 0x61, 0x64, 0x65, 0x72, 0x00, 0x09, 0x73,
		0x74, 0x61, 0x72, 0x74, 0x00, 0x0d, 0x00, 0x00,
	}

	// Encode the entry offset as ULEB128 at the correct position
	// For offset 0x1000 (4096), the ULEB128 is 0x80 0x20
	// For offset 0x2d8 (728), the ULEB128 is 0xd8 0x05
	if entryOffset <= 0x7f {
		buf[14] = byte(entryOffset)
		buf[15] = 0x05
		buf[16] = 0x00
	} else {
		// ULEB128 encoding for values > 127
		buf[14] = byte(entryOffset&0x7f) | 0x80
		buf[15] = byte((entryOffset >> 7) & 0x7f)
		if entryOffset >= 0x4000 {
			buf[15] |= 0x80
			buf[16] = byte((entryOffset >> 14) & 0x7f)
		}
	}

	// Fix the offset in the trie for where _start address is encoded
	// Looking at byte 44-45, this should also have the encoded address
	if entryOffset <= 0x7f {
		buf[45] = byte(entryOffset)
	} else {
		buf[45] = byte(entryOffset&0x7f) | 0x80
		buf[46] = byte((entryOffset >> 7) & 0x7f)
	}

	return buf
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

// Helper function to write structs in little-endian format
func writeStruct(file *os.File, data any) error {
	return binary.Write(file, binary.LittleEndian, data)
}

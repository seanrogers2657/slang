package slasm

// ARM64 Mach-O relocation types
// Reference: https://github.com/apple/llvm-project/blob/apple/main/llvm/include/llvm/BinaryFormat/MachO.h

const (
	// ARM64_RELOC_UNSIGNED is an absolute address (pointer-sized)
	ARM64_RELOC_UNSIGNED = 0

	// ARM64_RELOC_SUBTRACTOR is used with UNSIGNED for pointer differences
	ARM64_RELOC_SUBTRACTOR = 1

	// ARM64_RELOC_BRANCH26 is a 26-bit branch displacement (B, BL instructions)
	ARM64_RELOC_BRANCH26 = 2

	// ARM64_RELOC_PAGE21 is a 21-bit page-relative offset (ADRP instruction)
	ARM64_RELOC_PAGE21 = 3

	// ARM64_RELOC_PAGEOFF12 is a 12-bit page offset (ADD, LDR, STR with page offset)
	ARM64_RELOC_PAGEOFF12 = 4

	// ARM64_RELOC_GOT_LOAD_PAGE21 is for GOT-relative ADRP
	ARM64_RELOC_GOT_LOAD_PAGE21 = 5

	// ARM64_RELOC_GOT_LOAD_PAGEOFF12 is for GOT-relative LDR
	ARM64_RELOC_GOT_LOAD_PAGEOFF12 = 6

	// ARM64_RELOC_POINTER_TO_GOT is a 32-bit pointer to GOT entry
	ARM64_RELOC_POINTER_TO_GOT = 7

	// ARM64_RELOC_TLVP_LOAD_PAGE21 is for thread-local ADRP
	ARM64_RELOC_TLVP_LOAD_PAGE21 = 8

	// ARM64_RELOC_TLVP_LOAD_PAGEOFF12 is for thread-local LDR
	ARM64_RELOC_TLVP_LOAD_PAGEOFF12 = 9

	// ARM64_RELOC_ADDEND is an addend for the previous relocation
	ARM64_RELOC_ADDEND = 10
)

// TextRelocation represents a relocation entry for the __text section.
// This tracks where in the machine code we need to patch an address.
type TextRelocation struct {
	// Offset is the byte offset within the section where the relocation applies
	Offset uint32

	// SymbolName is the name of the symbol being referenced
	SymbolName string

	// Type is the ARM64 relocation type (ARM64_RELOC_*)
	Type uint8

	// PCRel indicates if the relocation is PC-relative
	PCRel bool

	// Length is log2 of the size: 0=1byte, 1=2byte, 2=4byte, 3=8byte
	Length uint8

	// Extern indicates if this references an external symbol (vs section)
	Extern bool
}

// MachORelocation is the raw Mach-O relocation_info structure (8 bytes)
// Reference: mach-o/reloc.h
type MachORelocation struct {
	// Address is the offset in the section to the item to be relocated
	Address int32

	// Packed contains symbolnum (24 bits), pcrel (1 bit), length (2 bits),
	// extern (1 bit), type (4 bits)
	Packed uint32
}

// PackRelocation packs relocation fields into the Mach-O format
func PackRelocation(symbolNum uint32, pcrel bool, length uint8, extern bool, relocType uint8) uint32 {
	packed := symbolNum & 0x00FFFFFF // 24 bits for symbol number

	if pcrel {
		packed |= 1 << 24
	}

	packed |= uint32(length&0x3) << 25 // 2 bits for length

	if extern {
		packed |= 1 << 27
	}

	packed |= uint32(relocType&0xF) << 28 // 4 bits for type

	return packed
}

// nlist64 is the symbol table entry structure for 64-bit Mach-O
// Reference: mach-o/nlist.h
type nlist64 struct {
	// Strx is the index into the string table
	Strx uint32

	// Type contains n_type flags:
	// N_UNDF (0x0) - undefined symbol
	// N_ABS  (0x2) - absolute symbol
	// N_SECT (0xe) - defined in section n_sect
	// N_EXT  (0x1) - external symbol (OR'd with above)
	Type uint8

	// Sect is the section number (1-based) or NO_SECT (0)
	Sect uint8

	// Desc contains additional flags (N_WEAK_DEF, etc.)
	Desc uint16

	// Value is the symbol value (address for defined symbols)
	Value uint64
}

// Symbol types for n_type field
const (
	N_UNDF = 0x0  // Undefined symbol
	N_ABS  = 0x2  // Absolute symbol
	N_SECT = 0xe  // Defined in section
	N_EXT  = 0x1  // External symbol (can be OR'd)
	N_PEXT = 0x10 // Private external
)

// NO_SECT indicates the symbol is not in any section
const NO_SECT = 0

// ObjectSymbol represents a symbol in an object file
type ObjectSymbol struct {
	Name   string
	Type   uint8  // N_UNDF, N_SECT, etc.
	Sect   uint8  // Section number (1-based) or NO_SECT
	Extern bool   // Is this an external symbol?
	Value  uint64 // Address/value
	StrIdx uint32 // String table index (set during output)
}

// ObjectSection represents a section in an object file
type ObjectSection struct {
	Name        string
	SegmentName string
	Data        []byte
	Align       uint32 // Power of 2 alignment
	Relocations []TextRelocation
	Flags       uint32
}

// ObjectFile represents a complete object file ready for output
type ObjectFile struct {
	TextSection *ObjectSection
	DataSection *ObjectSection
	Symbols     []ObjectSymbol
	StringTable []byte
}

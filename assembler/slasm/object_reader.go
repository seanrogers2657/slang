package slasm

import (
	"encoding/binary"
	"fmt"
	"os"
)

// ParsedObject represents a parsed Mach-O object file
type ParsedObject struct {
	// Header information
	Magic      uint32
	CPUType    uint32
	CPUSubtype uint32
	FileType   uint32
	NCmds      uint32

	// Sections
	TextSection *ParsedSection
	DataSection *ParsedSection

	// Symbols
	Symbols []ParsedSymbol

	// String table
	StringTable []byte

	// Source path (for error messages)
	SourcePath string
}

// ParsedSection represents a section from an object file
type ParsedSection struct {
	Name    string
	Segment string
	Data    []byte
	Addr    uint64
	Size    uint64
	Offset  uint32
	Align   uint32
	Relocs  []ParsedRelocation
}

// ParsedRelocation represents a relocation entry
type ParsedRelocation struct {
	Address   int32
	SymbolNum uint32
	PCRel     bool
	Length    uint8
	Extern    bool
	Type      uint8
}

// ParsedSymbol represents a symbol from the symbol table
type ParsedSymbol struct {
	Name    string
	Type    uint8
	Sect    uint8
	Desc    uint16
	Value   uint64
	Extern  bool
	Defined bool
}

// ReadObjectFile parses a Mach-O object file and returns its contents
func ReadObjectFile(path string) (*ParsedObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read object file: %w", err)
	}

	if len(data) < MachHeader64Size {
		return nil, fmt.Errorf("file too small to be a valid Mach-O file")
	}

	obj := &ParsedObject{
		SourcePath: path,
	}

	// Parse header
	obj.Magic = binary.LittleEndian.Uint32(data[0:4])
	if obj.Magic != MH_MAGIC_64 {
		return nil, fmt.Errorf("not a 64-bit Mach-O file (magic: 0x%x)", obj.Magic)
	}

	obj.CPUType = binary.LittleEndian.Uint32(data[4:8])
	obj.CPUSubtype = binary.LittleEndian.Uint32(data[8:12])
	obj.FileType = binary.LittleEndian.Uint32(data[12:16])
	obj.NCmds = binary.LittleEndian.Uint32(data[16:20])

	if obj.FileType != MH_OBJECT {
		return nil, fmt.Errorf("not an object file (type: %d)", obj.FileType)
	}

	// Parse load commands
	offset := uint32(MachHeader64Size)
	var symtabOff, symtabNsyms, strOff, strSize uint32

	for i := uint32(0); i < obj.NCmds; i++ {
		if int(offset+8) > len(data) {
			return nil, fmt.Errorf("truncated load command %d", i)
		}

		cmd := binary.LittleEndian.Uint32(data[offset : offset+4])
		cmdsize := binary.LittleEndian.Uint32(data[offset+4 : offset+8])

		switch cmd {
		case LC_SEGMENT_64:
			// Parse segment
			if err := obj.parseSegment(data, offset); err != nil {
				return nil, fmt.Errorf("failed to parse segment: %w", err)
			}

		case LC_SYMTAB:
			// Parse symbol table command
			symtabOff = binary.LittleEndian.Uint32(data[offset+8 : offset+12])
			symtabNsyms = binary.LittleEndian.Uint32(data[offset+12 : offset+16])
			strOff = binary.LittleEndian.Uint32(data[offset+16 : offset+20])
			strSize = binary.LittleEndian.Uint32(data[offset+20 : offset+24])
		}

		offset += cmdsize
	}

	// Load string table
	if strOff > 0 && strSize > 0 {
		if int(strOff+strSize) > len(data) {
			return nil, fmt.Errorf("string table extends beyond file")
		}
		obj.StringTable = data[strOff : strOff+strSize]
	}

	// Parse symbol table
	if symtabOff > 0 && symtabNsyms > 0 {
		if err := obj.parseSymbols(data, symtabOff, symtabNsyms); err != nil {
			return nil, fmt.Errorf("failed to parse symbols: %w", err)
		}
	}

	return obj, nil
}

// parseSegment parses an LC_SEGMENT_64 command
func (obj *ParsedObject) parseSegment(data []byte, offset uint32) error {
	// Skip cmd (4) and cmdsize (4)
	// segname is at offset+8, 16 bytes
	// vmaddr is at offset+24, 8 bytes
	// vmsize is at offset+32, 8 bytes
	// fileoff is at offset+40, 8 bytes
	// filesize is at offset+48, 8 bytes
	// maxprot is at offset+56, 4 bytes
	// initprot is at offset+60, 4 bytes
	// nsects is at offset+64, 4 bytes
	// flags is at offset+68, 4 bytes
	// Total segment command size: 72 bytes

	nsects := binary.LittleEndian.Uint32(data[offset+64 : offset+68])

	// Parse sections (each section header is 80 bytes)
	sectionOffset := offset + SegmentCommand64Size
	for i := uint32(0); i < nsects; i++ {
		section, err := obj.parseSection(data, sectionOffset)
		if err != nil {
			return fmt.Errorf("failed to parse section %d: %w", i, err)
		}

		// Categorize section
		switch section.Name {
		case "__text":
			obj.TextSection = section
		case "__data":
			obj.DataSection = section
		}

		sectionOffset += Section64Size
	}

	return nil
}

// parseSection parses a section64 structure
func (obj *ParsedObject) parseSection(data []byte, offset uint32) (*ParsedSection, error) {
	if int(offset+Section64Size) > len(data) {
		return nil, fmt.Errorf("section header extends beyond file")
	}

	section := &ParsedSection{}

	// sectname is at offset+0, 16 bytes
	sectname := data[offset : offset+16]
	section.Name = trimNull(sectname)

	// segname is at offset+16, 16 bytes
	segname := data[offset+16 : offset+32]
	section.Segment = trimNull(segname)

	// addr is at offset+32, 8 bytes
	section.Addr = binary.LittleEndian.Uint64(data[offset+32 : offset+40])

	// size is at offset+40, 8 bytes
	section.Size = binary.LittleEndian.Uint64(data[offset+40 : offset+48])

	// offset is at offset+48, 4 bytes
	section.Offset = binary.LittleEndian.Uint32(data[offset+48 : offset+52])

	// align is at offset+52, 4 bytes
	section.Align = binary.LittleEndian.Uint32(data[offset+52 : offset+56])

	// reloff is at offset+56, 4 bytes
	reloff := binary.LittleEndian.Uint32(data[offset+56 : offset+60])

	// nreloc is at offset+60, 4 bytes
	nreloc := binary.LittleEndian.Uint32(data[offset+60 : offset+64])

	// Load section data
	if section.Offset > 0 && section.Size > 0 {
		if int(uint64(section.Offset)+section.Size) > len(data) {
			return nil, fmt.Errorf("section data extends beyond file")
		}
		section.Data = make([]byte, section.Size)
		copy(section.Data, data[section.Offset:uint64(section.Offset)+section.Size])
	}

	// Load relocations
	if reloff > 0 && nreloc > 0 {
		for i := uint32(0); i < nreloc; i++ {
			reloc, err := parseRelocation(data, reloff+i*8)
			if err != nil {
				return nil, fmt.Errorf("failed to parse relocation %d: %w", i, err)
			}
			section.Relocs = append(section.Relocs, reloc)
		}
	}

	return section, nil
}

// parseRelocation parses a relocation_info structure (8 bytes)
func parseRelocation(data []byte, offset uint32) (ParsedRelocation, error) {
	if int(offset+8) > len(data) {
		return ParsedRelocation{}, fmt.Errorf("relocation extends beyond file")
	}

	reloc := ParsedRelocation{}
	reloc.Address = int32(binary.LittleEndian.Uint32(data[offset : offset+4]))

	packed := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
	reloc.SymbolNum = packed & 0x00FFFFFF
	reloc.PCRel = (packed>>24)&1 != 0
	reloc.Length = uint8((packed >> 25) & 0x3)
	reloc.Extern = (packed>>27)&1 != 0
	reloc.Type = uint8((packed >> 28) & 0xF)

	return reloc, nil
}

// parseSymbols parses the symbol table
func (obj *ParsedObject) parseSymbols(data []byte, symoff, nsyms uint32) error {
	for i := uint32(0); i < nsyms; i++ {
		offset := symoff + i*Nlist64Size
		if int(offset+Nlist64Size) > len(data) {
			return fmt.Errorf("symbol %d extends beyond file", i)
		}

		sym := ParsedSymbol{}

		// n_strx is at offset+0, 4 bytes
		strx := binary.LittleEndian.Uint32(data[offset : offset+4])

		// n_type is at offset+4, 1 byte
		sym.Type = data[offset+4]

		// n_sect is at offset+5, 1 byte
		sym.Sect = data[offset+5]

		// n_desc is at offset+6, 2 bytes
		sym.Desc = binary.LittleEndian.Uint16(data[offset+6 : offset+8])

		// n_value is at offset+8, 8 bytes
		sym.Value = binary.LittleEndian.Uint64(data[offset+8 : offset+16])

		// Get symbol name from string table
		if strx < uint32(len(obj.StringTable)) {
			sym.Name = readCString(obj.StringTable, strx)
		}

		// Determine if symbol is external and defined
		sym.Extern = sym.Type&N_EXT != 0
		sym.Defined = sym.Type&N_SECT != 0

		obj.Symbols = append(obj.Symbols, sym)
	}

	return nil
}

// trimNull removes null bytes from the end of a byte slice and returns a string
func trimNull(b []byte) string {
	for i := range b {
		if b[i] == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// readCString reads a null-terminated string from data starting at offset
func readCString(data []byte, offset uint32) string {
	end := offset
	for int(end) < len(data) && data[end] != 0 {
		end++
	}
	return string(data[offset:end])
}

package slasm

import (
	"encoding/binary"
	"fmt"
)

// Linker handles linking multiple object files into an executable
type Linker struct {
	// Objects are the parsed object files to link
	Objects []*ParsedObject

	// Combined output
	TextData []byte
	DataData []byte

	// Symbol resolution
	GlobalSymbols map[string]*LinkedSymbol

	// Logger for output
	Logger *Logger
}

// LinkedSymbol represents a resolved symbol
type LinkedSymbol struct {
	Name       string
	Value      uint64 // Final address in output
	Section    uint8  // MachOSectText or MachOSectData
	Defined    bool   // Is this symbol defined?
	SourceFile string // Which object file defined it
	ObjectIdx  int    // Index of object that defined it
}

// NewLinker creates a new linker
func NewLinker(logger *Logger) *Linker {
	if logger == nil {
		logger = NewSilentLogger()
	}
	return &Linker{
		GlobalSymbols: make(map[string]*LinkedSymbol),
		Logger:        logger,
	}
}

// LoadObjects loads and parses all object files
func (l *Linker) LoadObjects(paths []string) error {
	for _, path := range paths {
		l.Logger.Printf("Loading object file: %s\n", path)

		obj, err := ReadObjectFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		l.Objects = append(l.Objects, obj)
	}

	l.Logger.Printf("Loaded %d object file(s)\n", len(l.Objects))
	return nil
}

// CollectSymbols collects and resolves symbols from all object files
func (l *Linker) CollectSymbols() error {
	l.Logger.Printf("Collecting symbols...\n")

	// First pass: collect all defined symbols
	for objIdx, obj := range l.Objects {
		for _, sym := range obj.Symbols {
			if sym.Defined && sym.Extern {
				// Check for duplicate definition
				if existing, ok := l.GlobalSymbols[sym.Name]; ok {
					return fmt.Errorf("duplicate symbol '%s': defined in both %s and %s",
						sym.Name, existing.SourceFile, obj.SourcePath)
				}

				l.GlobalSymbols[sym.Name] = &LinkedSymbol{
					Name:       sym.Name,
					Value:      sym.Value, // Will be adjusted during layout
					Section:    sym.Sect,
					Defined:    true,
					SourceFile: obj.SourcePath,
					ObjectIdx:  objIdx,
				}
				l.Logger.Printf("  Found symbol: %s (section %d, value 0x%x)\n",
					sym.Name, sym.Sect, sym.Value)
			}
		}
	}

	// Second pass: check for undefined symbols
	for _, obj := range l.Objects {
		for _, sym := range obj.Symbols {
			if !sym.Defined && sym.Extern {
				if _, ok := l.GlobalSymbols[sym.Name]; !ok {
					return fmt.Errorf("undefined symbol '%s' referenced in %s",
						sym.Name, obj.SourcePath)
				}
			}
		}
	}

	l.Logger.Printf("Resolved %d global symbol(s)\n", len(l.GlobalSymbols))
	return nil
}

// MergeSections combines sections from all object files
func (l *Linker) MergeSections() error {
	l.Logger.Printf("Merging sections...\n")

	// Track offsets for each object's text section
	textOffsets := make([]uint64, len(l.Objects))
	dataOffsets := make([]uint64, len(l.Objects))

	// Merge text sections
	currentTextOffset := uint64(0)
	for i, obj := range l.Objects {
		textOffsets[i] = currentTextOffset

		if obj.TextSection != nil {
			// Align to 4 bytes
			if currentTextOffset%4 != 0 {
				padding := 4 - (currentTextOffset % 4)
				l.TextData = append(l.TextData, make([]byte, padding)...)
				currentTextOffset += padding
				textOffsets[i] = currentTextOffset
			}

			l.TextData = append(l.TextData, obj.TextSection.Data...)
			currentTextOffset += uint64(len(obj.TextSection.Data))

			l.Logger.Printf("  Object %d __text: %d bytes at offset 0x%x\n",
				i, len(obj.TextSection.Data), textOffsets[i])
		}
	}

	// Merge data sections
	currentDataOffset := uint64(0)
	for i, obj := range l.Objects {
		dataOffsets[i] = currentDataOffset

		if obj.DataSection != nil {
			// Align to 8 bytes
			if currentDataOffset%8 != 0 {
				padding := 8 - (currentDataOffset % 8)
				l.DataData = append(l.DataData, make([]byte, padding)...)
				currentDataOffset += padding
				dataOffsets[i] = currentDataOffset
			}

			l.DataData = append(l.DataData, obj.DataSection.Data...)
			currentDataOffset += uint64(len(obj.DataSection.Data))

			l.Logger.Printf("  Object %d __data: %d bytes at offset 0x%x\n",
				i, len(obj.DataSection.Data), dataOffsets[i])
		}
	}

	// Update symbol values with merged offsets
	for _, sym := range l.GlobalSymbols {
		objIdx := sym.ObjectIdx
		if sym.Section == MachOSectText {
			sym.Value += textOffsets[objIdx]
		} else if sym.Section == MachOSectData {
			sym.Value += dataOffsets[objIdx]
		}
		l.Logger.Printf("  Symbol %s -> 0x%x\n", sym.Name, sym.Value)
	}

	l.Logger.Printf("Merged: %d bytes text, %d bytes data\n",
		len(l.TextData), len(l.DataData))

	return nil
}

// ApplyRelocations patches the merged code with final addresses
func (l *Linker) ApplyRelocations(textVMAddr, dataVMAddr uint64) error {
	l.Logger.Printf("Applying relocations...\n")

	// Track current offset in merged text section
	textOffset := uint64(0)

	for objIdx, obj := range l.Objects {
		if obj.TextSection == nil {
			continue
		}

		objTextSize := uint64(len(obj.TextSection.Data))

		for _, reloc := range obj.TextSection.Relocs {
			// Calculate position in merged text
			patchOffset := textOffset + uint64(reloc.Address)

			// Get target symbol
			if !reloc.Extern {
				// Local relocations are resolved within the object file itself
				// Log for debugging/transparency
				l.Logger.Printf("  Skipping local relocation at offset 0x%x (type %d, symbol %d)\n",
					patchOffset, reloc.Type, reloc.SymbolNum)
				continue
			}

			if int(reloc.SymbolNum) >= len(obj.Symbols) {
				return fmt.Errorf("invalid symbol number %d in relocation", reloc.SymbolNum)
			}

			symName := obj.Symbols[reloc.SymbolNum].Name
			targetSym, ok := l.GlobalSymbols[symName]
			if !ok {
				return fmt.Errorf("undefined symbol '%s' in relocation", symName)
			}

			// Calculate target address
			var targetAddr uint64
			if targetSym.Section == MachOSectText {
				targetAddr = textVMAddr + targetSym.Value
			} else {
				targetAddr = dataVMAddr + targetSym.Value
			}

			// Apply relocation based on type
			switch reloc.Type {
			case ARM64_RELOC_BRANCH26:
				// 26-bit PC-relative branch
				pcAddr := textVMAddr + patchOffset
				offset := int64(targetAddr) - int64(pcAddr)
				if offset < -(1<<27) || offset >= (1<<27) {
					return fmt.Errorf("branch target out of range: %d", offset)
				}

				// Read current instruction
				instr := binary.LittleEndian.Uint32(l.TextData[patchOffset:])

				// Patch the imm26 field
				imm26 := uint32(offset>>2) & ARM64_IMM26_MASK
				instr = (instr & ARM64_BRANCH_OP_MASK) | imm26

				// Write back
				binary.LittleEndian.PutUint32(l.TextData[patchOffset:], instr)

				l.Logger.Printf("  BRANCH26 at 0x%x -> %s (0x%x)\n",
					patchOffset, symName, targetAddr)

			case ARM64_RELOC_PAGE21:
				// ADRP page-relative
				return fmt.Errorf("ARM64_RELOC_PAGE21 relocation not yet implemented at offset 0x%x for symbol '%s'",
					patchOffset, symName)

			case ARM64_RELOC_PAGEOFF12:
				// ADD/LDR/STR page offset
				return fmt.Errorf("ARM64_RELOC_PAGEOFF12 relocation not yet implemented at offset 0x%x for symbol '%s'",
					patchOffset, symName)

			case ARM64_RELOC_UNSIGNED:
				// Absolute address (8 bytes for 64-bit)
				binary.LittleEndian.PutUint64(l.TextData[patchOffset:], targetAddr)
				l.Logger.Printf("  UNSIGNED at 0x%x -> %s (0x%x)\n",
					patchOffset, symName, targetAddr)

			default:
				return fmt.Errorf("unsupported relocation type %d at offset 0x%x for symbol '%s'",
					reloc.Type, patchOffset, symName)
			}
		}

		textOffset += objTextSize
		// Align for next object
		if textOffset%4 != 0 {
			textOffset += 4 - (textOffset % 4)
		}

		_ = objIdx // Silence unused variable warning
	}

	return nil
}

// Link performs the complete linking process
func (l *Linker) Link(objectPaths []string, outputPath string, entryPoint string) error {
	// Load all object files
	if err := l.LoadObjects(objectPaths); err != nil {
		return err
	}

	// Collect and resolve symbols
	if err := l.CollectSymbols(); err != nil {
		return err
	}

	// Check for entry point
	entrySym, ok := l.GlobalSymbols[entryPoint]
	if !ok {
		return fmt.Errorf("entry point '%s' not found", entryPoint)
	}
	l.Logger.Printf("Entry point: %s\n", entryPoint)

	// Merge sections
	if err := l.MergeSections(); err != nil {
		return err
	}

	// Build symbol table for executable
	symTable := NewSymbolTable()
	for name, sym := range l.GlobalSymbols {
		sectionType := SectionText
		if sym.Section == MachOSectData {
			sectionType = SectionData
		}
		symTable.Define(name, sym.Value, sectionType, 0, 0) // line/col 0 for linked symbols
		symTable.MarkGlobal(name)
	}

	// Calculate VM addresses for relocation
	// These will be recalculated by WriteExecutable, but we need approximations
	textVMAddr := uint64(VMBaseAddress + 0x4000) // After header
	dataVMAddr := uint64(VMBaseAddress + 0x8000) // After text segment

	// Apply relocations
	if err := l.ApplyRelocations(textVMAddr, dataVMAddr); err != nil {
		return err
	}

	// Write executable
	l.Logger.Printf("Writing executable: %s\n", outputPath)

	writer := NewMachOWriter("arm64", l.Logger)
	err := writer.WriteExecutable(outputPath, l.TextData, l.DataData, nil, symTable, entryPoint)
	if err != nil {
		return fmt.Errorf("failed to write executable: %w", err)
	}

	// Make executable
	if err := makeExecutable(outputPath); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	l.Logger.Printf("Successfully linked %d object(s) -> %s\n", len(l.Objects), outputPath)
	l.Logger.Printf("  Entry point: %s at 0x%x\n", entryPoint, entrySym.Value)

	return nil
}

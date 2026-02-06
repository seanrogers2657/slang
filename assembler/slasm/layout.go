package slasm

import (
	"fmt"
)

// Layout calculates addresses for all instructions and data
type Layout struct {
	program     *Program
	symbolTable *SymbolTable
	constants   map[string]int64
}

// NewLayout creates a new layout calculator
func NewLayout(program *Program) *Layout {
	return &Layout{
		program:     program,
		symbolTable: NewSymbolTable(),
		constants:   make(map[string]int64),
	}
}

// Calculate performs two-pass layout calculation
// First pass: collect all label definitions and calculate addresses
// Second pass: resolve label references
func (l *Layout) Calculate() error {
	// Track current address for each section
	textAddr := uint64(0)
	dataAddr := uint64(0)

	// Process each section
	for _, section := range l.program.Sections {
		currentAddr := &textAddr
		sectionType := section.Type

		if section.Type == SectionData {
			currentAddr = &dataAddr
		}

		// Process each item in the section
		for _, item := range section.Items {
			switch v := item.(type) {
			case *Label:
				// Define symbol at current address
				err := l.symbolTable.Define(v.Name, *currentAddr, sectionType, v.Line, v.Column)
				if err != nil {
					return err
				}

			case *Instruction:
				// Each instruction is 4 bytes
				*currentAddr += uint64(instructionSize(v))

			case *Directive:
				// Handle alignment directives
				if v.Name == "align" && len(v.Args) > 0 {
					alignValue, err := parseAlignment(v.Args[0])
					if err != nil {
						return fmt.Errorf("line %d: %w", v.Line, err)
					}
					alignment := uint64(1 << alignValue) // 2^n
					padding := alignmentPadding(*currentAddr, alignment)
					*currentAddr += padding
				}
				// Mark symbols as global
				if v.Name == "global" && len(v.Args) > 0 {
					for _, arg := range v.Args {
						l.symbolTable.MarkGlobal(arg)
					}
				}
				// Mark symbols as extern (imported from another object file)
				if v.Name == "extern" && len(v.Args) > 0 {
					for _, arg := range v.Args {
						l.symbolTable.MarkExtern(arg)
					}
				}

			case *DataDeclaration:
				// Add size of data
				*currentAddr += uint64(dataSize(v))

			case *ConstantDef:
				// Validate constant name
				if v.Name == "" {
					return fmt.Errorf("line %d: constant definition has empty name", v.Line)
				}
				// Check for duplicate constant names
				if _, exists := l.constants[v.Name]; exists {
					return fmt.Errorf("line %d: duplicate constant '%s'", v.Line, v.Name)
				}
				// Store constant value (doesn't take address space)
				l.constants[v.Name] = v.Value
			}
		}
	}

	return nil
}

// GetSymbolTable returns the populated symbol table
func (l *Layout) GetSymbolTable() *SymbolTable {
	return l.symbolTable
}

// GetConstants returns the constants map
func (l *Layout) GetConstants() map[string]int64 {
	return l.constants
}

// Helper functions

// instructionSize returns the size in bytes of an instruction
func instructionSize(inst *Instruction) int {
	// All ARM64 instructions are 4 bytes
	return 4
}

// dataSize returns the size in bytes of a data declaration
func dataSize(data *DataDeclaration) int {
	switch data.Type {
	case "byte":
		// Count comma-separated values
		return countValues(data.Value)
	case "2byte", "hword":
		return countValues(data.Value) * 2
	case "4byte", "word":
		return countValues(data.Value) * 4
	case "8byte", "quad":
		return countValues(data.Value) * 8
	case "space", "zero":
		// Parse the size with validation (negative, overflow, max size checks)
		size, err := ParseSpaceSize(data.Value)
		if err != nil {
			return 0 // Layout phase - errors will be caught in encoding
		}
		return size
	case "asciz", "string":
		// String length + 1 for null terminator
		return len(UnescapeString(data.Value)) + 1
	case "ascii":
		// String length without null terminator
		return len(UnescapeString(data.Value))
	default:
		return 0
	}
}

// countValues counts comma-separated values in a string
func countValues(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, ch := range s {
		if ch == ',' {
			count++
		}
	}
	return count
}

// parseAlignment parses an alignment value and returns an error if invalid.
// The alignment value is the power of 2 (e.g., .align 4 means 16-byte alignment).
func parseAlignment(s string) (int, error) {
	// Use ParseInt for proper validation
	value, err := ParseInt(s)
	if err != nil {
		return 0, fmt.Errorf("invalid alignment value '%s': %w", s, err)
	}

	// ARM64 alignment values are typically 0-12 (1 to 4096 bytes)
	if value < 0 || value > 12 {
		return 0, fmt.Errorf("alignment value %d out of range (must be 0-12)", value)
	}

	return value, nil
}

// alignmentPadding calculates padding needed for alignment
func alignmentPadding(currentAddr uint64, alignment uint64) uint64 {
	if alignment == 0 {
		return 0
	}
	remainder := currentAddr % alignment
	if remainder == 0 {
		return 0
	}
	return alignment - remainder
}

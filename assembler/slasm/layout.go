package slasm

// Layout calculates addresses for all instructions and data
type Layout struct {
	program     *Program
	symbolTable *SymbolTable
}

// NewLayout creates a new layout calculator
func NewLayout(program *Program) *Layout {
	return &Layout{
		program:     program,
		symbolTable: NewSymbolTable(),
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
					// Parse alignment value
					alignment := uint64(4) // default
					if len(v.Args) > 0 {
						// Simple parsing - assume it's a number
						alignValue := parseAlignment(v.Args[0])
						if alignValue > 0 {
							alignment = uint64(1 << alignValue) // 2^n
						}
					}
					padding := alignmentPadding(*currentAddr, alignment)
					*currentAddr += padding
				}
				// Mark symbols as global
				if v.Name == "global" && len(v.Args) > 0 {
					for _, arg := range v.Args {
						l.symbolTable.MarkGlobal(arg)
					}
				}

			case *DataDeclaration:
				// Add size of data
				*currentAddr += uint64(dataSize(v))
			}
		}
	}

	return nil
}

// GetSymbolTable returns the populated symbol table
func (l *Layout) GetSymbolTable() *SymbolTable {
	return l.symbolTable
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
		return 1
	case "space":
		// Parse the size from Value (simple decimal parsing)
		size := 0
		for _, ch := range data.Value {
			if ch >= '0' && ch <= '9' {
				size = size*10 + int(ch-'0')
			}
		}
		return size
	case "asciz":
		// String length + 1 for null terminator
		return len(data.Value) + 1
	default:
		return 0
	}
}

// parseAlignment parses an alignment value
func parseAlignment(s string) int {
	// Simple decimal parsing
	result := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			result = result*10 + int(ch-'0')
		}
	}
	return result
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

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
	// TODO: Implement layout calculation
	// 1. First pass: iterate through all sections
	// 2. For each instruction/data, calculate its size
	// 3. Assign addresses to labels
	// 4. Track section offsets
	// 5. Build symbol table
	return nil
}

// GetSymbolTable returns the populated symbol table
func (l *Layout) GetSymbolTable() *SymbolTable {
	return l.symbolTable
}

// Helper functions

// instructionSize returns the size in bytes of an instruction
func instructionSize(inst *Instruction) int {
	// TODO: Implement instruction size calculation
	// Most ARM64 instructions are 4 bytes
	// Some pseudo-instructions expand to multiple instructions
	return 4
}

// dataSize returns the size in bytes of a data declaration
func dataSize(data *DataDeclaration) int {
	// TODO: Implement data size calculation
	// .byte = 1 byte
	// .space N = N bytes
	// .asciz = string length + 1 (null terminator)
	return 0
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

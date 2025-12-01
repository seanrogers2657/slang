package slasm

// Program represents a complete assembly program
type Program struct {
	Sections []*Section
}

// SectionType represents the type of section
type SectionType int

const (
	SectionData SectionType = iota
	SectionText
)

// Section represents a section in the assembly (.data, .text)
type Section struct {
	Type  SectionType
	Items []Item
}

// Item represents an item in a section (directive, label, instruction, data)
type Item interface {
	isItem()
}

// Directive represents an assembler directive (.global, .align, etc.)
type Directive struct {
	Name   string   // e.g., "global", "align"
	Args   []string // directive arguments
	Line   int      // source line number
	Column int      // source column number
}

func (d *Directive) isItem() {}

// Label represents a label definition
type Label struct {
	Name   string
	Line   int // source line number
	Column int // source column number
}

func (l *Label) isItem() {}

// Instruction represents an assembly instruction
type Instruction struct {
	Mnemonic string     // e.g., "mov", "add", "sub"
	Operands []*Operand // instruction operands
	Line     int        // source line number
	Column   int        // source column number
}

func (i *Instruction) isItem() {}

// OperandType represents the type of operand
type OperandType int

const (
	OperandRegister OperandType = iota
	OperandImmediate
	OperandLabel
	OperandMemory
)

// Operand represents an instruction operand
type Operand struct {
	Type  OperandType
	Value string // register name, immediate value, or label name

	// For memory operands [base, offset]
	Base            string // base register
	Offset          string // offset value or register
	Writeback       bool   // true for pre-indexed [base, #offset]!
	PostIndexOffset string // for post-indexed [base], #offset
}

// DataDeclaration represents a data declaration (.byte, .space, .asciz, etc.)
type DataDeclaration struct {
	Type  string // "byte", "space", "asciz", etc.
	Value string // the value or size
}

func (d *DataDeclaration) isItem() {}

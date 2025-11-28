package slasm

import (
	"testing"
)

func TestInstructionSize(t *testing.T) {
	tests := []struct {
		name         string
		instruction  *Instruction
		expectedSize int
	}{
		{
			name: "mov instruction",
			instruction: &Instruction{
				Mnemonic: "mov",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
					{Type: OperandImmediate, Value: "1"},
				},
			},
			expectedSize: 4,
		},
		{
			name: "add instruction",
			instruction: &Instruction{
				Mnemonic: "add",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x2"},
					{Type: OperandRegister, Value: "x0"},
					{Type: OperandRegister, Value: "x1"},
				},
			},
			expectedSize: 4,
		},
		{
			name: "branch instruction",
			instruction: &Instruction{
				Mnemonic: "b",
				Operands: []*Operand{
					{Type: OperandLabel, Value: "main"},
				},
			},
			expectedSize: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := instructionSize(tt.instruction)
			if size != tt.expectedSize {
				t.Errorf("expected size %d, got %d", tt.expectedSize, size)
			}
		})
	}
}

func TestDataSize(t *testing.T) {
	tests := []struct {
		name         string
		data         *DataDeclaration
		expectedSize int
	}{
		{
			name: ".byte declaration",
			data: &DataDeclaration{
				Type:  "byte",
				Value: "10",
			},
			expectedSize: 1,
		},
		{
			name: ".space 32",
			data: &DataDeclaration{
				Type:  "space",
				Value: "32",
			},
			expectedSize: 32,
		},
		{
			name: ".asciz with short string",
			data: &DataDeclaration{
				Type:  "asciz",
				Value: "Hello",
			},
			expectedSize: 6, // 5 chars + null terminator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := dataSize(tt.data)
			if size != tt.expectedSize {
				t.Errorf("expected size %d, got %d", tt.expectedSize, size)
			}
		})
	}
}

func TestAlignmentPadding(t *testing.T) {
	tests := []struct {
		name            string
		currentAddr     uint64
		alignment       uint64
		expectedPadding uint64
	}{
		{
			name:            "already aligned",
			currentAddr:     0x1000,
			alignment:       4,
			expectedPadding: 0,
		},
		{
			name:            "needs 1 byte padding",
			currentAddr:     0x1001,
			alignment:       4,
			expectedPadding: 3,
		},
		{
			name:            "needs 2 byte padding",
			currentAddr:     0x1002,
			alignment:       4,
			expectedPadding: 2,
		},
		{
			name:            "needs 3 byte padding",
			currentAddr:     0x1003,
			alignment:       4,
			expectedPadding: 1,
		},
		{
			name:            "8-byte alignment from 0",
			currentAddr:     0,
			alignment:       8,
			expectedPadding: 0,
		},
		{
			name:            "8-byte alignment from 4",
			currentAddr:     4,
			alignment:       8,
			expectedPadding: 4,
		},
		{
			name:            "no alignment (0)",
			currentAddr:     0x1234,
			alignment:       0,
			expectedPadding: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padding := alignmentPadding(tt.currentAddr, tt.alignment)
			if padding != tt.expectedPadding {
				t.Errorf("expected padding %d, got %d", tt.expectedPadding, padding)
			}
		})
	}
}

func TestLayout_SimpleProgram(t *testing.T) {
	// Create a simple program with one text section
	program := &Program{
		Sections: []*Section{
			{
				Type: SectionText,
				Items: []Item{
					&Label{Name: "_start"},
					&Instruction{
						Mnemonic: "mov",
						Operands: []*Operand{
							{Type: OperandRegister, Value: "x0"},
							{Type: OperandImmediate, Value: "1"},
						},
					},
					&Instruction{
						Mnemonic: "mov",
						Operands: []*Operand{
							{Type: OperandRegister, Value: "x16"},
							{Type: OperandImmediate, Value: "1"},
						},
					},
					&Instruction{
						Mnemonic: "svc",
						Operands: []*Operand{
							{Type: OperandImmediate, Value: "0"},
						},
					},
				},
			},
		},
	}

	layout := NewLayout(program)
	err := layout.Calculate()

	if err != nil {
		t.Fatalf("layout calculation failed: %v", err)
	}

	// Check that the symbol table has _start
	st := layout.GetSymbolTable()
	symbol, exists := st.Lookup("_start")

	if !exists {
		t.Fatal("expected _start symbol to exist")
	}

	if symbol.Address != 0 {
		t.Errorf("expected _start at address 0, got 0x%x", symbol.Address)
	}
}

func TestLayout_MultipleLabels(t *testing.T) {
	// Create a program with multiple labels
	program := &Program{
		Sections: []*Section{
			{
				Type: SectionText,
				Items: []Item{
					&Label{Name: "_start"},
					&Instruction{Mnemonic: "b", Operands: []*Operand{{Type: OperandLabel, Value: "main"}}},
					&Label{Name: "main"},
					&Instruction{Mnemonic: "mov", Operands: []*Operand{
						{Type: OperandRegister, Value: "x0"},
						{Type: OperandImmediate, Value: "0"},
					}},
					&Instruction{Mnemonic: "ret"},
				},
			},
		},
	}

	layout := NewLayout(program)
	err := layout.Calculate()

	if err != nil {
		t.Fatalf("layout calculation failed: %v", err)
	}

	st := layout.GetSymbolTable()

	// Check _start at address 0
	startSym, exists := st.Lookup("_start")
	if !exists {
		t.Fatal("expected _start symbol to exist")
	}
	if startSym.Address != 0 {
		t.Errorf("expected _start at 0, got 0x%x", startSym.Address)
	}

	// Check main at address 4 (after one branch instruction)
	mainSym, exists := st.Lookup("main")
	if !exists {
		t.Fatal("expected main symbol to exist")
	}
	if mainSym.Address != 4 {
		t.Errorf("expected main at 4, got 0x%x", mainSym.Address)
	}
}

func TestLayout_DataSection(t *testing.T) {
	// Create a program with data section
	program := &Program{
		Sections: []*Section{
			{
				Type: SectionData,
				Items: []Item{
					&Label{Name: "buffer"},
					&DataDeclaration{Type: "space", Value: "32"},
					&Label{Name: "newline"},
					&DataDeclaration{Type: "byte", Value: "10"},
				},
			},
		},
	}

	layout := NewLayout(program)
	err := layout.Calculate()

	if err != nil {
		t.Fatalf("layout calculation failed: %v", err)
	}

	st := layout.GetSymbolTable()

	// Check buffer at address 0
	bufferSym, exists := st.Lookup("buffer")
	if !exists {
		t.Fatal("expected buffer symbol to exist")
	}
	if bufferSym.Address != 0 {
		t.Errorf("expected buffer at 0, got 0x%x", bufferSym.Address)
	}
	if bufferSym.Section != SectionData {
		t.Errorf("expected buffer in data section, got %v", bufferSym.Section)
	}

	// Check newline at address 32 (after 32-byte buffer)
	newlineSym, exists := st.Lookup("newline")
	if !exists {
		t.Fatal("expected newline symbol to exist")
	}
	if newlineSym.Address != 32 {
		t.Errorf("expected newline at 32, got 0x%x", newlineSym.Address)
	}
}

func TestLayout_WithAlignment(t *testing.T) {
	// Create a program with alignment directives
	program := &Program{
		Sections: []*Section{
			{
				Type: SectionText,
				Items: []Item{
					&Directive{Name: "align", Args: []string{"4"}},
					&Label{Name: "_start"},
					&Instruction{Mnemonic: "mov", Operands: []*Operand{
						{Type: OperandRegister, Value: "x0"},
						{Type: OperandImmediate, Value: "0"},
					}},
				},
			},
		},
	}

	layout := NewLayout(program)
	err := layout.Calculate()

	if err != nil {
		t.Fatalf("layout calculation failed: %v", err)
	}

	st := layout.GetSymbolTable()
	symbol, exists := st.Lookup("_start")

	if !exists {
		t.Fatal("expected _start symbol to exist")
	}

	// Address should be aligned to 4 bytes
	if symbol.Address%4 != 0 {
		t.Errorf("expected _start to be 4-byte aligned, got address 0x%x", symbol.Address)
	}
}

func TestLayout_MixedSections(t *testing.T) {
	// Create a program with both data and text sections
	program := &Program{
		Sections: []*Section{
			{
				Type: SectionData,
				Items: []Item{
					&Label{Name: "message"},
					&DataDeclaration{Type: "asciz", Value: "Hello"},
				},
			},
			{
				Type: SectionText,
				Items: []Item{
					&Label{Name: "_start"},
					&Instruction{Mnemonic: "mov", Operands: []*Operand{
						{Type: OperandRegister, Value: "x0"},
						{Type: OperandImmediate, Value: "1"},
					}},
				},
			},
		},
	}

	layout := NewLayout(program)
	err := layout.Calculate()

	if err != nil {
		t.Fatalf("layout calculation failed: %v", err)
	}

	st := layout.GetSymbolTable()

	// Check that we have symbols in both sections
	messageSym, exists := st.Lookup("message")
	if !exists {
		t.Fatal("expected message symbol to exist")
	}
	if messageSym.Section != SectionData {
		t.Errorf("expected message in data section, got %v", messageSym.Section)
	}

	startSym, exists := st.Lookup("_start")
	if !exists {
		t.Fatal("expected _start symbol to exist")
	}
	if startSym.Section != SectionText {
		t.Errorf("expected _start in text section, got %v", startSym.Section)
	}
}

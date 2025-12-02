package slasm

import (
	"testing"
)

func TestParser_SimpleInstruction(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []Token
		expectError bool
	}{
		{
			name: "mov with immediate",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "mov"},
				{Type: TokenRegister, Value: "x0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenHash, Value: "#"},
				{Type: TokenInteger, Value: "42"},
				{Type: TokenEOF},
			},
			expectError: false,
		},
		{
			name: "add three registers",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "add"},
				{Type: TokenRegister, Value: "x2"},
				{Type: TokenComma, Value: ","},
				{Type: TokenRegister, Value: "x0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenRegister, Value: "x1"},
				{Type: TokenEOF},
			},
			expectError: false,
		},
		{
			name: "branch to label",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "b"},
				{Type: TokenIdentifier, Value: "main"},
				{Type: TokenEOF},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.tokens)
			program, err := parser.Parse()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if program == nil {
					t.Error("expected program but got nil")
				}
			}
		})
	}
}

func TestParser_LabelDefinition(t *testing.T) {
	tokens := []Token{
		{Type: TokenIdentifier, Value: "main"},
		{Type: TokenColon, Value: ":"},
		{Type: TokenNewline},
		{Type: TokenIdentifier, Value: "mov"},
		{Type: TokenRegister, Value: "x0"},
		{Type: TokenComma, Value: ","},
		{Type: TokenHash, Value: "#"},
		{Type: TokenInteger, Value: "1"},
		{Type: TokenEOF},
	}

	parser := NewParser(tokens)
	program, err := parser.Parse()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if program == nil {
		t.Fatal("expected program but got nil")
	}

	// Program should have at least one section
	if len(program.Sections) == 0 {
		t.Fatal("expected at least one section")
	}
}

func TestParser_Directives(t *testing.T) {
	tests := []struct {
		name      string
		tokens    []Token
		directive string
	}{
		{
			name: ".global directive",
			tokens: []Token{
				{Type: TokenDirective, Value: ".global"},
				{Type: TokenIdentifier, Value: "_start"},
				{Type: TokenEOF},
			},
			directive: ".global",
		},
		{
			name: ".align directive",
			tokens: []Token{
				{Type: TokenDirective, Value: ".align"},
				{Type: TokenInteger, Value: "4"},
				{Type: TokenEOF},
			},
			directive: ".align",
		},
		{
			name: ".data section",
			tokens: []Token{
				{Type: TokenDirective, Value: ".data"},
				{Type: TokenEOF},
			},
			directive: ".data",
		},
		{
			name: ".text section",
			tokens: []Token{
				{Type: TokenDirective, Value: ".text"},
				{Type: TokenEOF},
			},
			directive: ".text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.tokens)
			program, err := parser.Parse()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if program == nil {
				t.Error("expected program but got nil")
			}
		})
	}
}

func TestParser_DataDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		tokens []Token
	}{
		{
			name: ".space declaration",
			tokens: []Token{
				{Type: TokenDirective, Value: ".data"},
				{Type: TokenNewline},
				{Type: TokenIdentifier, Value: "buffer"},
				{Type: TokenColon, Value: ":"},
				{Type: TokenDirective, Value: ".space"},
				{Type: TokenInteger, Value: "32"},
				{Type: TokenEOF},
			},
		},
		{
			name: ".byte declaration",
			tokens: []Token{
				{Type: TokenDirective, Value: ".data"},
				{Type: TokenNewline},
				{Type: TokenIdentifier, Value: "newline"},
				{Type: TokenColon, Value: ":"},
				{Type: TokenDirective, Value: ".byte"},
				{Type: TokenInteger, Value: "10"},
				{Type: TokenEOF},
			},
		},
		{
			name: ".asciz declaration",
			tokens: []Token{
				{Type: TokenDirective, Value: ".data"},
				{Type: TokenNewline},
				{Type: TokenIdentifier, Value: "message"},
				{Type: TokenColon, Value: ":"},
				{Type: TokenDirective, Value: ".asciz"},
				{Type: TokenString, Value: "Hello"},
				{Type: TokenEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.tokens)
			program, err := parser.Parse()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if program == nil {
				t.Error("expected program but got nil")
			}
		})
	}
}

func TestParser_MemoryOperands(t *testing.T) {
	tests := []struct {
		name   string
		tokens []Token
	}{
		{
			name: "load with base and offset",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "ldr"},
				{Type: TokenRegister, Value: "x0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenRegister, Value: "sp"},
				{Type: TokenComma, Value: ","},
				{Type: TokenHash, Value: "#"},
				{Type: TokenInteger, Value: "16"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenEOF},
			},
		},
		{
			name: "store pair",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "stp"},
				{Type: TokenRegister, Value: "x29"},
				{Type: TokenComma, Value: ","},
				{Type: TokenRegister, Value: "x30"},
				{Type: TokenComma, Value: ","},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenRegister, Value: "sp"},
				{Type: TokenComma, Value: ","},
				{Type: TokenHash, Value: "#"},
				{Type: TokenInteger, Value: "-16"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.tokens)
			program, err := parser.Parse()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if program == nil {
				t.Error("expected program but got nil")
			}
		})
	}
}

func TestParser_PageOffsetRelocations(t *testing.T) {
	tests := []struct {
		name   string
		tokens []Token
	}{
		{
			name: "adrp with @PAGE",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "adrp"},
				{Type: TokenRegister, Value: "x0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenIdentifier, Value: "buffer"},
				{Type: TokenAt, Value: "@"},
				{Type: TokenIdentifier, Value: "PAGE"},
				{Type: TokenEOF},
			},
		},
		{
			name: "add with @PAGEOFF",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "add"},
				{Type: TokenRegister, Value: "x0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenRegister, Value: "x0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenIdentifier, Value: "buffer"},
				{Type: TokenAt, Value: "@"},
				{Type: TokenIdentifier, Value: "PAGEOFF"},
				{Type: TokenEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.tokens)
			program, err := parser.Parse()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if program == nil {
				t.Error("expected program but got nil")
			}
		})
	}
}

func TestParser_CompleteProgram(t *testing.T) {
	// A simple but complete assembly program
	tokens := []Token{
		// .global _start
		{Type: TokenDirective, Value: ".global"},
		{Type: TokenIdentifier, Value: "_start"},
		{Type: TokenNewline},

		// .align 4
		{Type: TokenDirective, Value: ".align"},
		{Type: TokenInteger, Value: "4"},
		{Type: TokenNewline},

		// _start:
		{Type: TokenIdentifier, Value: "_start"},
		{Type: TokenColon, Value: ":"},
		{Type: TokenNewline},

		// mov x0, #1
		{Type: TokenIdentifier, Value: "mov"},
		{Type: TokenRegister, Value: "x0"},
		{Type: TokenComma, Value: ","},
		{Type: TokenHash, Value: "#"},
		{Type: TokenInteger, Value: "1"},
		{Type: TokenNewline},

		// mov x16, #1
		{Type: TokenIdentifier, Value: "mov"},
		{Type: TokenRegister, Value: "x16"},
		{Type: TokenComma, Value: ","},
		{Type: TokenHash, Value: "#"},
		{Type: TokenInteger, Value: "1"},
		{Type: TokenNewline},

		// svc #0
		{Type: TokenIdentifier, Value: "svc"},
		{Type: TokenHash, Value: "#"},
		{Type: TokenInteger, Value: "0"},
		{Type: TokenEOF},
	}

	parser := NewParser(tokens)
	program, err := parser.Parse()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if program == nil {
		t.Fatal("expected program but got nil")
	}

	// Should have at least one section
	if len(program.Sections) == 0 {
		t.Error("expected at least one section in program")
	}
}

func TestParser_ConstantDefinition(t *testing.T) {
	tests := []struct {
		name          string
		tokens        []Token
		expectedName  string
		expectedValue int64
	}{
		{
			name: "simple constant",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "MY_CONST"},
				{Type: TokenEquals, Value: "="},
				{Type: TokenInteger, Value: "42"},
				{Type: TokenEOF},
			},
			expectedName:  "MY_CONST",
			expectedValue: 42,
		},
		{
			name: "hex constant",
			tokens: []Token{
				{Type: TokenIdentifier, Value: "BUF_SIZE"},
				{Type: TokenEquals, Value: "="},
				{Type: TokenInteger, Value: "0x100"},
				{Type: TokenEOF},
			},
			expectedName:  "BUF_SIZE",
			expectedValue: 256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.tokens)
			program, err := parser.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if program == nil {
				t.Fatal("expected program but got nil")
			}

			// Should have one section with one constant definition
			if len(program.Sections) == 0 {
				t.Fatal("expected at least one section")
			}

			if len(program.Sections[0].Items) == 0 {
				t.Fatal("expected at least one item in section")
			}

			constDef, ok := program.Sections[0].Items[0].(*ConstantDef)
			if !ok {
				t.Fatalf("expected ConstantDef, got %T", program.Sections[0].Items[0])
			}

			if constDef.Name != tt.expectedName {
				t.Errorf("expected name %s, got %s", tt.expectedName, constDef.Name)
			}

			if constDef.Value != tt.expectedValue {
				t.Errorf("expected value %d, got %d", tt.expectedValue, constDef.Value)
			}
		})
	}
}

func TestParser_ConstantWithInstructions(t *testing.T) {
	// Test constant definition followed by instruction that uses it
	tokens := []Token{
		// message_len = 14
		{Type: TokenIdentifier, Value: "message_len"},
		{Type: TokenEquals, Value: "="},
		{Type: TokenInteger, Value: "14"},
		{Type: TokenNewline},

		// mov x2, #message_len
		{Type: TokenIdentifier, Value: "mov"},
		{Type: TokenRegister, Value: "x2"},
		{Type: TokenComma, Value: ","},
		{Type: TokenHash, Value: "#"},
		{Type: TokenIdentifier, Value: "message_len"},
		{Type: TokenEOF},
	}

	parser := NewParser(tokens)
	program, err := parser.Parse()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if program == nil {
		t.Fatal("expected program but got nil")
	}

	// Should have one section with two items
	if len(program.Sections) == 0 {
		t.Fatal("expected at least one section")
	}

	if len(program.Sections[0].Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(program.Sections[0].Items))
	}

	// First item should be constant
	constDef, ok := program.Sections[0].Items[0].(*ConstantDef)
	if !ok {
		t.Fatalf("expected ConstantDef, got %T", program.Sections[0].Items[0])
	}
	if constDef.Name != "message_len" || constDef.Value != 14 {
		t.Errorf("unexpected constant: %+v", constDef)
	}

	// Second item should be instruction
	inst, ok := program.Sections[0].Items[1].(*Instruction)
	if !ok {
		t.Fatalf("expected Instruction, got %T", program.Sections[0].Items[1])
	}
	if inst.Mnemonic != "mov" {
		t.Errorf("expected mov instruction, got %s", inst.Mnemonic)
	}
}

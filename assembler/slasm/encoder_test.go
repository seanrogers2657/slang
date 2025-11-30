package slasm

import (
    "encoding/hex"
    "testing"
)

func TestEncodeAdd(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// add x2, x0, x1
	inst := &Instruction{
		Mnemonic: "add",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0x8b010002 (little-endian)
	expected := "0200018b"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeSub(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// sub x2, x0, x1
	inst := &Instruction{
		Mnemonic: "sub",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0xcb010002 (little-endian)
	expected := "020001cb"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeMul(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// mul x2, x0, x1
	inst := &Instruction{
		Mnemonic: "mul",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0x9b017c02 (little-endian)
	expected := "027c019b"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeSdiv(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// sdiv x2, x0, x1
	inst := &Instruction{
		Mnemonic: "sdiv",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0x9ac10c02 (little-endian)
	expected := "020cc19a"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeMsub(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// msub x3, x2, x0, x1
	inst := &Instruction{
		Mnemonic: "msub",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x3"},
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0x9b008443 (little-endian)
	expected := "4384009b"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeCmp(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// cmp x0, x1
	inst := &Instruction{
		Mnemonic: "cmp",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0xeb01001f (little-endian)
	expected := "1f0001eb"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeCset(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable())

	// cset x0, eq
	inst := &Instruction{
		Mnemonic: "cset",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandLabel, Value: "eq"},
		},
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Expected: 0x9a9f17e0 (little-endian)
	expected := "e0179f9a"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeBranch(t *testing.T) {
	tests := []struct {
		name           string
		targetAddr     uint64
		currentAddr    uint64
		expectedHex    string
		expectedUint32 uint32
	}{
		{
			name:           "branch forward 1 instruction",
			targetAddr:     0x04,
			currentAddr:    0x00,
			expectedHex:    "01000014", // b +1 (0x14000001)
			expectedUint32: 0x14000001,
		},
		{
			name:           "branch forward 4 instructions",
			targetAddr:     0x10,
			currentAddr:    0x00,
			expectedHex:    "04000014", // b +4 (0x14000004)
			expectedUint32: 0x14000004,
		},
		{
			name:           "branch backward 1 instruction",
			targetAddr:     0x00,
			currentAddr:    0x04,
			expectedHex:    "ffffff17", // b -1 (0x17ffffff)
			expectedUint32: 0x17ffffff,
		},
		{
			name:           "branch backward 4 instructions",
			targetAddr:     0x00,
			currentAddr:    0x10,
			expectedHex:    "fcffff17", // b -4 (0x17fffffc)
			expectedUint32: 0x17fffffc,
		},
		{
			name:           "branch to self (infinite loop)",
			targetAddr:     0x08,
			currentAddr:    0x08,
			expectedHex:    "00000014", // b +0 (0x14000000)
			expectedUint32: 0x14000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create symbol table with target label
			symbolTable := NewSymbolTable()
			symbolTable.Define("target", tt.targetAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable)

			inst := &Instruction{
				Mnemonic: "b",
				Operands: []*Operand{
					{Type: OperandLabel, Value: "target"},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, tt.currentAddr)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(bytes)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeBranch_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inst        *Instruction
		expectError string
	}{
		{
			name: "missing operand",
			inst: &Instruction{
				Mnemonic: "b",
				Operands: []*Operand{},
				Line:     1,
				Column:   1,
			},
			expectError: "b requires 1 operand",
		},
		{
			name: "wrong operand type",
			inst: &Instruction{
				Mnemonic: "b",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "b requires a label operand",
		},
		{
			name: "undefined label",
			inst: &Instruction{
				Mnemonic: "b",
				Operands: []*Operand{
					{Type: OperandLabel, Value: "undefined_label"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "undefined label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			encoder := NewEncoder(symbolTable)

			_, err := encoder.Encode(tt.inst, 0)
			if err == nil {
				t.Fatalf("Expected error containing '%s', got nil", tt.expectError)
			}

			if !contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectError, err.Error())
			}
		})
	}
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

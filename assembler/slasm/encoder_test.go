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

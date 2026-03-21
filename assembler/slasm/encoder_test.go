package slasm

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestEncodeAdd(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

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
	encoder := NewEncoder(NewSymbolTable(), nil)

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

func TestEncodeSubImmediate(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		rn          string
		imm         string
		expectedHex string
	}{
		{
			name:        "sub x1, x1, #1",
			rd:          "x1",
			rn:          "x1",
			imm:         "1",
			expectedHex: "210400d1", // 0xD1000421 in little-endian
		},
		{
			name:        "sub x0, x0, #10",
			rd:          "x0",
			rn:          "x0",
			imm:         "10",
			expectedHex: "002800d1", // 0xD1002800 in little-endian
		},
		{
			name:        "sub x2, x3, #100",
			rd:          "x2",
			rn:          "x3",
			imm:         "100",
			expectedHex: "629001d1", // 0xD1019062 in little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "sub",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandImmediate, Value: tt.imm},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeMul(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

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
	encoder := NewEncoder(NewSymbolTable(), nil)

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
	encoder := NewEncoder(NewSymbolTable(), nil)

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

func TestEncodeNeg(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		inst     *Instruction
		expected string // hex little-endian
	}{
		{
			name: "neg x1, x0",
			inst: &Instruction{
				Mnemonic: "neg",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x1"},
					{Type: OperandRegister, Value: "x0"},
				},
			},
			// NEG x1, x0 = SUB x1, XZR, x0 = 0xcb0003e1
			expected: "e10300cb",
		},
		{
			name: "neg x2, x5",
			inst: &Instruction{
				Mnemonic: "neg",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x2"},
					{Type: OperandRegister, Value: "x5"},
				},
			},
			// NEG x2, x5 = SUB x2, XZR, x5 = 0xcb0503e2
			expected: "e20305cb",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytes, err := encoder.Encode(tc.inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(bytes)
			if got != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, got)
			}
		})
	}
}

func TestEncodeUdiv(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		rn          string
		rm          string
		expectedHex string
	}{
		{
			name:        "udiv x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "0208c19a", // 0x9AC10802
		},
		{
			name:        "udiv x11, x20, x10",
			rd:          "x11",
			rn:          "x20",
			rm:          "x10",
			expectedHex: "8b0aca9a", // 0x9ACA0A8B
		},
		{
			name:        "udiv x0, x1, x2",
			rd:          "x0",
			rn:          "x1",
			rm:          "x2",
			expectedHex: "2008c29a", // 0x9AC20820
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "udiv",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeCmp(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

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
	encoder := NewEncoder(NewSymbolTable(), nil)

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

			encoder := NewEncoder(symbolTable, nil)

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
			encoder := NewEncoder(symbolTable, nil)

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

func TestEncodeBranchLink(t *testing.T) {
	tests := []struct {
		name        string
		targetAddr  uint64
		currentAddr uint64
		expectedHex string
	}{
		{
			name:        "bl forward 1 instruction",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "01000094", // bl +1 (0x94000001)
		},
		{
			name:        "bl forward 4 instructions",
			targetAddr:  0x10,
			currentAddr: 0x00,
			expectedHex: "04000094", // bl +4 (0x94000004)
		},
		{
			name:        "bl backward 1 instruction",
			targetAddr:  0x00,
			currentAddr: 0x04,
			expectedHex: "ffffff97", // bl -1 (0x97ffffff)
		},
		{
			name:        "bl backward 4 instructions",
			targetAddr:  0x00,
			currentAddr: 0x10,
			expectedHex: "fcffff97", // bl -4 (0x97fffffc)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			symbolTable.Define("target", tt.targetAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable, nil)

			inst := &Instruction{
				Mnemonic: "bl",
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

func TestEncodeBranchLink_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inst        *Instruction
		expectError string
	}{
		{
			name: "missing operand",
			inst: &Instruction{
				Mnemonic: "bl",
				Operands: []*Operand{},
				Line:     1,
				Column:   1,
			},
			expectError: "bl requires 1 operand",
		},
		{
			name: "wrong operand type",
			inst: &Instruction{
				Mnemonic: "bl",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "bl requires a label operand",
		},
		{
			name: "undefined label",
			inst: &Instruction{
				Mnemonic: "bl",
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
			encoder := NewEncoder(symbolTable, nil)

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

func TestEncodeBranchRegister(t *testing.T) {
	tests := []struct {
		name        string
		register    string
		expectedHex string
	}{
		{
			name:        "br x0",
			register:    "x0",
			expectedHex: "00001fd6", // br x0 (0xD61F0000)
		},
		{
			name:        "br x1",
			register:    "x1",
			expectedHex: "20001fd6", // br x1 (0xD61F0020)
		},
		{
			name:        "br x30 (link register)",
			register:    "x30",
			expectedHex: "c0031fd6", // br x30 (0xD61F03C0)
		},
		{
			name:        "br lr (alias for x30)",
			register:    "lr",
			expectedHex: "c0031fd6", // br lr (0xD61F03C0)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "br",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.register},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeBranchRegister_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inst        *Instruction
		expectError string
	}{
		{
			name: "missing operand",
			inst: &Instruction{
				Mnemonic: "br",
				Operands: []*Operand{},
				Line:     1,
				Column:   1,
			},
			expectError: "br requires 1 operand",
		},
		{
			name: "wrong operand type (label)",
			inst: &Instruction{
				Mnemonic: "br",
				Operands: []*Operand{
					{Type: OperandLabel, Value: "target"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "br requires a register operand",
		},
		{
			name: "wrong operand type (immediate)",
			inst: &Instruction{
				Mnemonic: "br",
				Operands: []*Operand{
					{Type: OperandImmediate, Value: "42"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "br requires a register operand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

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

func TestEncodeBranchConditional(t *testing.T) {
	tests := []struct {
		name        string
		mnemonic    string
		targetAddr  uint64
		currentAddr uint64
		expectedHex string
	}{
		// Test all condition codes with forward branch
		{
			name:        "b.eq forward",
			mnemonic:    "b.eq",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "20000054", // b.eq +1 (0x54000020)
		},
		{
			name:        "b.ne forward",
			mnemonic:    "b.ne",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "21000054", // b.ne +1 (0x54000021)
		},
		{
			name:        "b.cs forward",
			mnemonic:    "b.cs",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "22000054", // b.cs +1 (0x54000022)
		},
		{
			name:        "b.cc forward",
			mnemonic:    "b.cc",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "23000054", // b.cc +1 (0x54000023)
		},
		{
			name:        "b.mi forward",
			mnemonic:    "b.mi",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "24000054", // b.mi +1 (0x54000024)
		},
		{
			name:        "b.pl forward",
			mnemonic:    "b.pl",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "25000054", // b.pl +1 (0x54000025)
		},
		{
			name:        "b.vs forward",
			mnemonic:    "b.vs",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "26000054", // b.vs +1 (0x54000026)
		},
		{
			name:        "b.vc forward",
			mnemonic:    "b.vc",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "27000054", // b.vc +1 (0x54000027)
		},
		{
			name:        "b.hi forward",
			mnemonic:    "b.hi",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "28000054", // b.hi +1 (0x54000028)
		},
		{
			name:        "b.ls forward",
			mnemonic:    "b.ls",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "29000054", // b.ls +1 (0x54000029)
		},
		{
			name:        "b.ge forward",
			mnemonic:    "b.ge",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "2a000054", // b.ge +1 (0x5400002a)
		},
		{
			name:        "b.lt forward",
			mnemonic:    "b.lt",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "2b000054", // b.lt +1 (0x5400002b)
		},
		{
			name:        "b.gt forward",
			mnemonic:    "b.gt",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "2c000054", // b.gt +1 (0x5400002c)
		},
		{
			name:        "b.le forward",
			mnemonic:    "b.le",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "2d000054", // b.le +1 (0x5400002d)
		},
		// Test backward branch
		{
			name:        "b.eq backward",
			mnemonic:    "b.eq",
			targetAddr:  0x00,
			currentAddr: 0x04,
			expectedHex: "e0ffff54", // b.eq -1 (0x54ffffe0)
		},
		// Test larger offset
		{
			name:        "b.ne forward 4 instructions",
			mnemonic:    "b.ne",
			targetAddr:  0x10,
			currentAddr: 0x00,
			expectedHex: "81000054", // b.ne +4 (0x54000081)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			symbolTable.Define("target", tt.targetAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable, nil)

			inst := &Instruction{
				Mnemonic: tt.mnemonic,
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

func TestEncodeBranchConditional_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inst        *Instruction
		expectError string
	}{
		{
			name: "missing operand",
			inst: &Instruction{
				Mnemonic: "b.eq",
				Operands: []*Operand{},
				Line:     1,
				Column:   1,
			},
			expectError: "b.eq requires 1 operand",
		},
		{
			name: "wrong operand type",
			inst: &Instruction{
				Mnemonic: "b.eq",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "b.eq requires a label operand",
		},
		{
			name: "undefined label",
			inst: &Instruction{
				Mnemonic: "b.ne",
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
			encoder := NewEncoder(symbolTable, nil)

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

func TestEncodeRet(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	inst := &Instruction{
		Mnemonic: "ret",
		Operands: []*Operand{},
		Line:     1,
		Column:   1,
	}

	bytes, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// RET is BR X30 = 0xD65F03C0
	expected := "c0035fd6"
	got := hex.EncodeToString(bytes)

	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeLdr(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "ldr x0, [sp]",
			rt:          "x0",
			base:        "sp",
			offset:      "",
			expectedHex: "e00340f9", // 0xF94003E0
		},
		{
			name:        "ldr x0, [sp, #0]",
			rt:          "x0",
			base:        "sp",
			offset:      "0",
			expectedHex: "e00340f9", // 0xF94003E0
		},
		{
			name:        "ldr x0, [sp, #8]",
			rt:          "x0",
			base:        "sp",
			offset:      "8",
			expectedHex: "e00740f9", // 0xF94007E0
		},
		{
			name:        "ldr x1, [x2, #16]",
			rt:          "x1",
			base:        "x2",
			offset:      "16",
			expectedHex: "410840f9", // 0xF9400841
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "ldr",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeStr(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "str x0, [sp]",
			rt:          "x0",
			base:        "sp",
			offset:      "",
			expectedHex: "e00300f9", // 0xF90003E0
		},
		{
			name:        "str x0, [sp, #8]",
			rt:          "x0",
			base:        "sp",
			offset:      "8",
			expectedHex: "e00700f9", // 0xF90007E0
		},
		{
			name:        "str x1, [x2, #16]",
			rt:          "x1",
			base:        "x2",
			offset:      "16",
			expectedHex: "410800f9", // 0xF9000841
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "str",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeLdrPreIndexed(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "ldr x0, [sp, #-16]!",
			rt:          "x0",
			base:        "sp",
			offset:      "-16",
			expectedHex: "e00f5ff8", // LDR pre-indexed: 0xF85F0FE0
		},
		{
			name:        "ldr x1, [x2, #-8]!",
			rt:          "x1",
			base:        "x2",
			offset:      "-8",
			expectedHex: "418c5ff8", // LDR pre-indexed: 0xF85F8C41
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "ldr",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset, Writeback: true},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeLdrPostIndexed(t *testing.T) {
	tests := []struct {
		name            string
		rt              string
		base            string
		postIndexOffset string
		expectedHex     string
	}{
		{
			name:            "ldr x0, [sp], #16",
			rt:              "x0",
			base:            "sp",
			postIndexOffset: "16",
			expectedHex:     "e00741f8", // LDR post-indexed: 0xF84107E0
		},
		{
			name:            "ldr x1, [x2], #8",
			rt:              "x1",
			base:            "x2",
			postIndexOffset: "8",
			expectedHex:     "418440f8", // LDR post-indexed: 0xF8408441
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "ldr",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, PostIndexOffset: tt.postIndexOffset},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeStrPreIndexed(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "str x0, [sp, #-16]!",
			rt:          "x0",
			base:        "sp",
			offset:      "-16",
			expectedHex: "e00f1ff8", // STR pre-indexed: 0xF81F0FE0
		},
		{
			name:        "str x2, [sp, #-16]!",
			rt:          "x2",
			base:        "sp",
			offset:      "-16",
			expectedHex: "e20f1ff8", // STR pre-indexed: 0xF81F0FE2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "str",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset, Writeback: true},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeStrPostIndexed(t *testing.T) {
	tests := []struct {
		name            string
		rt              string
		base            string
		postIndexOffset string
		expectedHex     string
	}{
		{
			name:            "str x0, [sp], #16",
			rt:              "x0",
			base:            "sp",
			postIndexOffset: "16",
			expectedHex:     "e00701f8", // STR post-indexed: 0xF80107E0
		},
		{
			name:            "str x1, [x2], #-8",
			rt:              "x1",
			base:            "x2",
			postIndexOffset: "-8",
			expectedHex:     "41841ff8", // STR post-indexed: 0xF81F8441
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "str",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, PostIndexOffset: tt.postIndexOffset},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeLdp(t *testing.T) {
	tests := []struct {
		name        string
		rt1         string
		rt2         string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "ldp x29, x30, [sp]",
			rt1:         "x29",
			rt2:         "x30",
			base:        "sp",
			offset:      "",
			expectedHex: "fd7b40a9", // 0xA9407BFD
		},
		{
			name:        "ldp x0, x1, [sp, #16]",
			rt1:         "x0",
			rt2:         "x1",
			base:        "sp",
			offset:      "16",
			expectedHex: "e00741a9", // 0xA94107E0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "ldp",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt1},
					{Type: OperandRegister, Value: tt.rt2},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

func TestEncodeStp(t *testing.T) {
	tests := []struct {
		name        string
		rt1         string
		rt2         string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "stp x29, x30, [sp]",
			rt1:         "x29",
			rt2:         "x30",
			base:        "sp",
			offset:      "",
			expectedHex: "fd7b00a9", // 0xA9007BFD
		},
		{
			name:        "stp x0, x1, [sp, #16]",
			rt1:         "x0",
			rt2:         "x1",
			base:        "sp",
			offset:      "16",
			expectedHex: "e00701a9", // 0xA90107E0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			inst := &Instruction{
				Mnemonic: "stp",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt1},
					{Type: OperandRegister, Value: tt.rt2},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset},
				},
				Line:   1,
				Column: 1,
			}

			bytes, err := encoder.Encode(inst, 0)
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

// Data encoding tests

func TestEncodeDataByte(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []byte
	}{
		{
			name:     "single byte",
			value:    "42",
			expected: []byte{42},
		},
		{
			name:     "multiple bytes",
			value:    "1,2,3,4",
			expected: []byte{1, 2, 3, 4},
		},
		{
			name:     "hex value",
			value:    "0x0a",
			expected: []byte{10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			data := &DataDeclaration{Type: "byte", Value: tt.value}

			result, err := encoder.EncodeData(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEncodeDataQuad(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []byte
	}{
		{
			name:     "single quad",
			value:    "0x100000000",
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			name:     "small value",
			value:    "42",
			expected: []byte{42, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			data := &DataDeclaration{Type: "quad", Value: tt.value}

			result, err := encoder.EncodeData(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEncodeDataWord(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []byte
	}{
		{
			name:     "single word",
			value:    "0x12345678",
			expected: []byte{0x78, 0x56, 0x34, 0x12}, // little-endian
		},
		{
			name:     "small value",
			value:    "256",
			expected: []byte{0x00, 0x01, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			data := &DataDeclaration{Type: "word", Value: tt.value}

			result, err := encoder.EncodeData(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEncodeDataAsciz(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []byte
	}{
		{
			name:     "simple string",
			value:    "Hello",
			expected: []byte{'H', 'e', 'l', 'l', 'o', 0},
		},
		{
			name:     "string with newline",
			value:    "Hello\\n",
			expected: []byte{'H', 'e', 'l', 'l', 'o', '\n', 0},
		},
		{
			name:     "empty string",
			value:    "",
			expected: []byte{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			data := &DataDeclaration{Type: "asciz", Value: tt.value}

			result, err := encoder.EncodeData(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEncodeDataAscii(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []byte
	}{
		{
			name:     "simple string no null",
			value:    "Hello",
			expected: []byte{'H', 'e', 'l', 'l', 'o'}, // no null terminator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			data := &DataDeclaration{Type: "ascii", Value: tt.value}

			result, err := encoder.EncodeData(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEncodeDataSpace(t *testing.T) {
	tests := []struct {
		name  string
		value string
		size  int
	}{
		{
			name:  "8 bytes",
			value: "8",
			size:  8,
		},
		{
			name:  "32 bytes",
			value: "32",
			size:  32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			data := &DataDeclaration{Type: "space", Value: tt.value}

			result, err := encoder.EncodeData(data)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			if len(result) != tt.size {
				t.Errorf("Expected %d bytes, got %d", tt.size, len(result))
			}

			// All bytes should be zero
			for i, b := range result {
				if b != 0 {
					t.Errorf("Expected byte %d to be 0, got %d", i, b)
				}
			}
		})
	}
}

func TestEncodeLsl(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		rn          string
		shift       string
		isImmediate bool
		expectedHex string
	}{
		{
			name:        "lsl x0, x0, #1",
			rd:          "x0",
			rn:          "x0",
			shift:       "1",
			isImmediate: true,
			// LSL #1 = UBFM x0, x0, #63, #62
			// immr = (64-1) & 0x3F = 63 = 0x3F
			// imms = 63-1 = 62 = 0x3E
			// Encoding: 0xD3400000 | (63 << 16) | (62 << 10) | (0 << 5) | 0
			// = 0xD3400000 | 0x3F0000 | 0xF800 | 0 | 0 = 0xD37FF800
			expectedHex: "00f87fd3",
		},
		{
			name:        "lsl x1, x2, #4",
			rd:          "x1",
			rn:          "x2",
			shift:       "4",
			isImmediate: true,
			// LSL #4 = UBFM x1, x2, #60, #59
			// immr = (64-4) & 0x3F = 60 = 0x3C
			// imms = 63-4 = 59 = 0x3B
			expectedHex: "41ec7cd3",
		},
		{
			name:        "lsl x0, x1, x2 (register)",
			rd:          "x0",
			rn:          "x1",
			shift:       "x2",
			isImmediate: false,
			// LSLV: 0x9AC02000 | (2 << 16) | (1 << 5) | 0
			expectedHex: "2020c29a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			operands := []*Operand{
				{Type: OperandRegister, Value: tt.rd},
				{Type: OperandRegister, Value: tt.rn},
			}
			if tt.isImmediate {
				operands = append(operands, &Operand{Type: OperandImmediate, Value: tt.shift})
			} else {
				operands = append(operands, &Operand{Type: OperandRegister, Value: tt.shift})
			}

			inst := &Instruction{
				Mnemonic: "lsl",
				Operands: operands,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeLsr(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		rn          string
		shift       string
		isImmediate bool
		expectedHex string
	}{
		{
			name:        "lsr x0, x0, #1",
			rd:          "x0",
			rn:          "x0",
			shift:       "1",
			isImmediate: true,
			// LSR #1 = UBFM x0, x0, #1, #63
			// immr = 1, imms = 63
			// Encoding: 0xD3400000 | (1 << 16) | (63 << 10) | (0 << 5) | 0
			expectedHex: "00fc41d3",
		},
		{
			name:        "lsr x0, x1, x2 (register)",
			rd:          "x0",
			rn:          "x1",
			shift:       "x2",
			isImmediate: false,
			// LSRV: 0x9AC02400 | (2 << 16) | (1 << 5) | 0
			expectedHex: "2024c29a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			operands := []*Operand{
				{Type: OperandRegister, Value: tt.rd},
				{Type: OperandRegister, Value: tt.rn},
			}
			if tt.isImmediate {
				operands = append(operands, &Operand{Type: OperandImmediate, Value: tt.shift})
			} else {
				operands = append(operands, &Operand{Type: OperandRegister, Value: tt.shift})
			}

			inst := &Instruction{
				Mnemonic: "lsr",
				Operands: operands,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeAsr(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		rn          string
		shift       string
		isImmediate bool
		expectedHex string
	}{
		{
			name:        "asr x0, x0, #1",
			rd:          "x0",
			rn:          "x0",
			shift:       "1",
			isImmediate: true,
			// ASR #1 = SBFM x0, x0, #1, #63
			// immr = 1, imms = 63
			// Encoding: 0x93400000 | (1 << 16) | (63 << 10) | (0 << 5) | 0
			expectedHex: "00fc4193",
		},
		{
			name:        "asr x0, x1, x2 (register)",
			rd:          "x0",
			rn:          "x1",
			shift:       "x2",
			isImmediate: false,
			// ASRV: 0x9AC02800 | (2 << 16) | (1 << 5) | 0
			expectedHex: "2028c29a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)

			operands := []*Operand{
				{Type: OperandRegister, Value: tt.rd},
				{Type: OperandRegister, Value: tt.rn},
			}
			if tt.isImmediate {
				operands = append(operands, &Operand{Type: OperandImmediate, Value: tt.shift})
			} else {
				operands = append(operands, &Operand{Type: OperandRegister, Value: tt.shift})
			}

			inst := &Instruction{
				Mnemonic: "asr",
				Operands: operands,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeAdr(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		label       string
		labelAddr   uint64
		instAddr    uint64
		expectedHex string
	}{
		{
			name:      "adr x0, label (forward, +8)",
			rd:        "x0",
			label:     "test",
			labelAddr: 8,
			instAddr:  0,
			// offset = 8, immlo = 0, immhi = 2
			// Encoding: (0 << 29) | (0b10000 << 24) | (2 << 5) | 0
			expectedHex: "40000010",
		},
		{
			name:      "adr x1, label (backward, -4)",
			rd:        "x1",
			label:     "test",
			labelAddr: 0,
			instAddr:  4,
			// offset = -4 = 0xFFFFFFFC (21-bit: 0x1FFFFC)
			// immlo = 0, immhi = 0x7FFFF (all 1s except sign in 21-bit)
			expectedHex: "e1ffff10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			symbolTable.Define(tt.label, tt.labelAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable, nil)

			inst := &Instruction{
				Mnemonic: "adr",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandLabel, Value: tt.label},
				},
			}

			result, err := encoder.Encode(inst, tt.instAddr)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestResolveImmediate(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		constants map[string]int64
		expected  int64
		wantErr   bool
	}{
		{
			name:      "integer value",
			value:     "42",
			constants: nil,
			expected:  42,
			wantErr:   false,
		},
		{
			name:      "hex value",
			value:     "0xFF",
			constants: nil,
			expected:  255,
			wantErr:   false,
		},
		{
			name:      "constant lookup",
			value:     "MY_CONST",
			constants: map[string]int64{"MY_CONST": 100},
			expected:  100,
			wantErr:   false,
		},
		{
			name:      "undefined constant",
			value:     "UNKNOWN",
			constants: map[string]int64{},
			expected:  0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), tt.constants)

			result, err := encoder.ResolveImmediate(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Logical Operations Tests
// ============================================================================

func TestEncodeAnd(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "and x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "0200018a", // 0x8A010002 little-endian
		},
		{
			name:        "and x5, x3, x4",
			rd:          "x5",
			rn:          "x3",
			rm:          "x4",
			expectedHex: "6500048a", // 0x8A040065 little-endian
		},
		{
			name:        "and x0, x0, x0",
			rd:          "x0",
			rn:          "x0",
			rm:          "x0",
			expectedHex: "0000008a", // 0x8A000000 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "and",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeOrr(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "orr x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "020001aa", // 0xAA010002 little-endian
		},
		{
			name:        "orr x10, x8, x9",
			rd:          "x10",
			rn:          "x8",
			rm:          "x9",
			expectedHex: "0a0109aa", // 0xAA09010A little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "orr",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeEor(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "eor x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "020001ca", // 0xCA010002 little-endian
		},
		{
			name:        "eor x7, x5, x6",
			rd:          "x7",
			rn:          "x5",
			rm:          "x6",
			expectedHex: "a70006ca", // 0xCA0600A7 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "eor",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeMvn(t *testing.T) {
	tests := []struct {
		name        string
		rd, rm      string
		expectedHex string
	}{
		{
			name:        "mvn x2, x1",
			rd:          "x2",
			rm:          "x1",
			expectedHex: "e20321aa", // 0xAA2103E2 little-endian (ORN with Rn=XZR)
		},
		{
			name:        "mvn x0, x5",
			rd:          "x0",
			rm:          "x5",
			expectedHex: "e00325aa", // 0xAA2503E0 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "mvn",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeAnds(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "ands x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "020001ea", // 0xEA010002 little-endian
		},
		{
			name:        "ands x3, x4, x5",
			rd:          "x3",
			rn:          "x4",
			rm:          "x5",
			expectedHex: "830005ea", // 0xEA050083 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "ands",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeTst(t *testing.T) {
	tests := []struct {
		name        string
		rn, rm      string
		expectedHex string
	}{
		{
			name:        "tst x0, x1",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "1f0001ea", // 0xEA01001F little-endian (ANDS with Rd=XZR)
		},
		{
			name:        "tst x5, x6",
			rn:          "x5",
			rm:          "x6",
			expectedHex: "bf0006ea", // 0xEA0600BF little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "tst",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeBic(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "bic x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "0200218a", // 0x8A210002 little-endian
		},
		{
			name:        "bic x5, x3, x4",
			rd:          "x5",
			rn:          "x3",
			rm:          "x4",
			expectedHex: "6500248a", // 0x8A240065 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "bic",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeOrn(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "orn x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "020021aa", // 0xAA210002 little-endian
		},
		{
			name:        "orn x7, x8, x9",
			rd:          "x7",
			rn:          "x8",
			rm:          "x9",
			expectedHex: "070129aa", // 0xAA290107 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "orn",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeEon(t *testing.T) {
	tests := []struct {
		name        string
		rd, rn, rm  string
		expectedHex string
	}{
		{
			name:        "eon x2, x0, x1",
			rd:          "x2",
			rn:          "x0",
			rm:          "x1",
			expectedHex: "020021ca", // 0xCA210002 little-endian
		},
		{
			name:        "eon x6, x4, x5",
			rd:          "x6",
			rn:          "x4",
			rm:          "x5",
			expectedHex: "860025ca", // 0xCA250086 little-endian
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "eon",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandRegister, Value: tt.rm},
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

// ============================================================================
// Halfword Memory Operations Tests
// ============================================================================

func TestEncodeLdrh(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "ldrh w0, [x1]",
			rt:          "w0",
			base:        "x1",
			offset:      "",
			expectedHex: "20004079", // 0x79400020 little-endian
		},
		{
			name:        "ldrh w2, [x3, #4]",
			rt:          "w2",
			base:        "x3",
			offset:      "4",
			expectedHex: "62084079", // 0x79400862 - offset/2 = 2 shifted left 10 bits
		},
		{
			name:        "ldrh w5, [x10, #10]",
			rt:          "w5",
			base:        "x10",
			offset:      "10",
			expectedHex: "45154079", // offset/2 = 5 -> 5<<10 = 0x1400
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			memOp := &Operand{
				Type: OperandMemory,
				Base: tt.base,
			}
			if tt.offset != "" {
				memOp.Offset = tt.offset
			}
			inst := &Instruction{
				Mnemonic: "ldrh",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					memOp,
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeCbz(t *testing.T) {
	tests := []struct {
		name        string
		register    string
		targetAddr  uint64
		currentAddr uint64
		expectedHex string
	}{
		{
			name:        "cbz x0, forward 1 instruction",
			register:    "x0",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "200000b4", // cbz x0, +1 (0xB4000020)
		},
		{
			name:        "cbz x1, forward 4 instructions",
			register:    "x1",
			targetAddr:  0x10,
			currentAddr: 0x00,
			expectedHex: "810000b4", // cbz x1, +4 (0xB4000081)
		},
		{
			name:        "cbz x2, backward 1 instruction",
			register:    "x2",
			targetAddr:  0x00,
			currentAddr: 0x04,
			expectedHex: "e2ffffb4", // cbz x2, -1 (0xB4FFFFE2)
		},
		{
			name:        "cbz x3, backward 4 instructions",
			register:    "x3",
			targetAddr:  0x00,
			currentAddr: 0x10,
			expectedHex: "83ffffb4", // cbz x3, -4 (0xB4FFFF83)
		},
		{
			name:        "cbz x0, to self (infinite loop)",
			register:    "x0",
			targetAddr:  0x08,
			currentAddr: 0x08,
			expectedHex: "000000b4", // cbz x0, +0 (0xB4000000)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			symbolTable.Define("target", tt.targetAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable, nil)

			inst := &Instruction{
				Mnemonic: "cbz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.register},
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

func TestEncodeCbz_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inst        *Instruction
		expectError string
	}{
		{
			name: "missing operand",
			inst: &Instruction{
				Mnemonic: "cbz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "cbz requires 2 operands",
		},
		{
			name: "wrong first operand type",
			inst: &Instruction{
				Mnemonic: "cbz",
				Operands: []*Operand{
					{Type: OperandLabel, Value: "target"},
					{Type: OperandLabel, Value: "target"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "cbz first operand must be a register",
		},
		{
			name: "wrong second operand type",
			inst: &Instruction{
				Mnemonic: "cbz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
					{Type: OperandRegister, Value: "x1"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "cbz second operand must be a label",
		},
		{
			name: "undefined label",
			inst: &Instruction{
				Mnemonic: "cbz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
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
			encoder := NewEncoder(symbolTable, nil)

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

func TestEncodeCbnz(t *testing.T) {
	tests := []struct {
		name        string
		register    string
		targetAddr  uint64
		currentAddr uint64
		expectedHex string
	}{
		{
			name:        "cbnz x0, forward 1 instruction",
			register:    "x0",
			targetAddr:  0x04,
			currentAddr: 0x00,
			expectedHex: "200000b5", // cbnz x0, +1 (0xB5000020)
		},
		{
			name:        "cbnz x1, forward 4 instructions",
			register:    "x1",
			targetAddr:  0x10,
			currentAddr: 0x00,
			expectedHex: "810000b5", // cbnz x1, +4 (0xB5000081)
		},
		{
			name:        "cbnz x2, backward 1 instruction",
			register:    "x2",
			targetAddr:  0x00,
			currentAddr: 0x04,
			expectedHex: "e2ffffb5", // cbnz x2, -1 (0xB5FFFFE2)
		},
		{
			name:        "cbnz x3, backward 4 instructions",
			register:    "x3",
			targetAddr:  0x00,
			currentAddr: 0x10,
			expectedHex: "83ffffb5", // cbnz x3, -4 (0xB5FFFF83)
		},
		{
			name:        "cbnz x0, to self (infinite loop)",
			register:    "x0",
			targetAddr:  0x08,
			currentAddr: 0x08,
			expectedHex: "000000b5", // cbnz x0, +0 (0xB5000000)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			symbolTable.Define("target", tt.targetAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable, nil)

			inst := &Instruction{
				Mnemonic: "cbnz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.register},
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

func TestEncodeCbnz_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inst        *Instruction
		expectError string
	}{
		{
			name: "missing operand",
			inst: &Instruction{
				Mnemonic: "cbnz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "cbnz requires 2 operands",
		},
		{
			name: "wrong first operand type",
			inst: &Instruction{
				Mnemonic: "cbnz",
				Operands: []*Operand{
					{Type: OperandLabel, Value: "target"},
					{Type: OperandLabel, Value: "target"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "cbnz first operand must be a register",
		},
		{
			name: "wrong second operand type",
			inst: &Instruction{
				Mnemonic: "cbnz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
					{Type: OperandRegister, Value: "x1"},
				},
				Line:   1,
				Column: 1,
			},
			expectError: "cbnz second operand must be a label",
		},
		{
			name: "undefined label",
			inst: &Instruction{
				Mnemonic: "cbnz",
				Operands: []*Operand{
					{Type: OperandRegister, Value: "x0"},
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
			encoder := NewEncoder(symbolTable, nil)

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

func TestEncodeStrh(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "strh w0, [x1]",
			rt:          "w0",
			base:        "x1",
			offset:      "",
			expectedHex: "20000079", // 0x79000020 little-endian
		},
		{
			name:        "strh w2, [x3, #4]",
			rt:          "w2",
			base:        "x3",
			offset:      "4",
			expectedHex: "62080079", // 0x79000862 - offset/2 = 2 shifted left 10 bits
		},
		{
			name:        "strh w7, [x8, #20]",
			rt:          "w7",
			base:        "x8",
			offset:      "20",
			expectedHex: "07290079", // offset/2 = 10 -> 10<<10 = 0x2800
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			memOp := &Operand{
				Type: OperandMemory,
				Base: tt.base,
			}
			if tt.offset != "" {
				memOp.Offset = tt.offset
			}
			inst := &Instruction{
				Mnemonic: "strh",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					memOp,
				},
			}
			got, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if hex.EncodeToString(got) != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, hex.EncodeToString(got))
			}
		})
	}
}

func TestEncodeWithRelocations_ExternSymbol(t *testing.T) {
	// Create a symbol table with an extern symbol
	st := NewSymbolTable()
	st.MarkExtern("_external_func")

	encoder := NewEncoder(st, nil)

	// Test bl to extern symbol generates relocation
	t.Run("bl extern generates relocation", func(t *testing.T) {
		inst := &Instruction{
			Mnemonic: "bl",
			Operands: []*Operand{
				{Type: OperandLabel, Value: "_external_func"},
			},
		}

		result, err := encoder.EncodeWithRelocations(inst, 0x10)
		if err != nil {
			t.Fatalf("EncodeWithRelocations failed: %v", err)
		}

		// Should have bytes (BL with imm26=0)
		expectedHex := "00000094" // 0x94000000 little-endian
		if hex.EncodeToString(result.Bytes) != expectedHex {
			t.Errorf("Expected %s, got %s", expectedHex, hex.EncodeToString(result.Bytes))
		}

		// Should have relocation
		if result.Relocation == nil {
			t.Fatal("Expected relocation, got nil")
		}
		if result.Relocation.Offset != 0x10 {
			t.Errorf("Expected relocation offset 0x10, got 0x%x", result.Relocation.Offset)
		}
		if result.Relocation.SymbolName != "_external_func" {
			t.Errorf("Expected symbol name '_external_func', got '%s'", result.Relocation.SymbolName)
		}
		if result.Relocation.Type != ARM64_RELOC_BRANCH26 {
			t.Errorf("Expected relocation type BRANCH26, got %d", result.Relocation.Type)
		}
		if !result.Relocation.PCRel {
			t.Error("Expected PCRel to be true")
		}
		if !result.Relocation.Extern {
			t.Error("Expected Extern to be true")
		}
	})

	// Test b to extern symbol generates relocation
	t.Run("b extern generates relocation", func(t *testing.T) {
		inst := &Instruction{
			Mnemonic: "b",
			Operands: []*Operand{
				{Type: OperandLabel, Value: "_external_func"},
			},
		}

		result, err := encoder.EncodeWithRelocations(inst, 0x20)
		if err != nil {
			t.Fatalf("EncodeWithRelocations failed: %v", err)
		}

		// Should have bytes (B with imm26=0)
		expectedHex := "00000014" // 0x14000000 little-endian
		if hex.EncodeToString(result.Bytes) != expectedHex {
			t.Errorf("Expected %s, got %s", expectedHex, hex.EncodeToString(result.Bytes))
		}

		// Should have relocation
		if result.Relocation == nil {
			t.Fatal("Expected relocation, got nil")
		}
		if result.Relocation.Offset != 0x20 {
			t.Errorf("Expected relocation offset 0x20, got 0x%x", result.Relocation.Offset)
		}
	})

	// Test bl to defined symbol does NOT generate relocation
	t.Run("bl defined symbol no relocation", func(t *testing.T) {
		st2 := NewSymbolTable()
		st2.Define("_local_func", 0x100, SectionText, 0, 0)
		encoder2 := NewEncoder(st2, nil)

		inst := &Instruction{
			Mnemonic: "bl",
			Operands: []*Operand{
				{Type: OperandLabel, Value: "_local_func"},
			},
		}

		result, err := encoder2.EncodeWithRelocations(inst, 0x0)
		if err != nil {
			t.Fatalf("EncodeWithRelocations failed: %v", err)
		}

		// Should have no relocation
		if result.Relocation != nil {
			t.Errorf("Expected no relocation for defined symbol, got %+v", result.Relocation)
		}

		// Should have valid encoding (branch offset = 0x100/4 = 64 = 0x40)
		// BL encoding: 0x94000040 -> little-endian: 40 00 00 94
		expectedHex := "40000094"
		if hex.EncodeToString(result.Bytes) != expectedHex {
			t.Errorf("Expected %s, got %s", expectedHex, hex.EncodeToString(result.Bytes))
		}
	})
}

func TestEncodeMov(t *testing.T) {
	tests := []struct {
		name        string
		operands    []*Operand
		expectedHex string
	}{
		{
			name: "mov x0, x1 (register to register)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
			},
			// ORR X0, XZR, X1: (1<<31)|(0b0101010<<24)|(1<<16)|(31<<5)|0 = 0xAA0103E0
			expectedHex: "e00301aa",
		},
		{
			name: "mov x5, x10 (higher registers)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x5"},
				{Type: OperandRegister, Value: "x10"},
			},
			// ORR X5, XZR, X10 = 0xAA0A03E5
			expectedHex: "e5030aaa",
		},
		{
			name: "mov x0, #42 (immediate)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "42"},
			},
			// MOVZ X0, #42: (1<<31)|(0b10100101<<23)|(42<<5) = 0xD2800540
			expectedHex: "400580d2",
		},
		{
			name: "mov sp, x0 (move to SP uses ADD)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "sp"},
				{Type: OperandRegister, Value: "x0"},
			},
			// ADD SP, X0, #0: (1<<31)|(0b0010001<<24)|(0<<5)|31 = 0x9100001F
			expectedHex: "1f000091",
		},
		{
			name: "mov x0, sp (move from SP uses ADD)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "sp"},
			},
			// ADD X0, SP, #0: (1<<31)|(0b0010001<<24)|(31<<5)|0 = 0x910003E0
			expectedHex: "e0030091",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "mov",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeMov_Errors(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		operands []*Operand
	}{
		{
			name: "too few operands",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
			},
		},
		{
			name: "too many operands",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandRegister, Value: "x2"},
			},
		},
		{
			name: "immediate too large",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "70000"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instruction{
				Mnemonic: "mov",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}
			_, err := encoder.Encode(inst, 0)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestEncodeMovz(t *testing.T) {
	tests := []struct {
		name        string
		operands    []*Operand
		expectedHex string
	}{
		{
			name: "movz x1, #0x1234",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandImmediate, Value: "0x1234"},
			},
			// MOVZ: (1<<31)|(0b10100101<<23)|(0<<21)|(0x1234<<5)|1 = 0xD2824681
			expectedHex: "814682d2",
		},
		{
			name: "movz x2, #0x5678, lsl #16",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x2"},
				{Type: OperandImmediate, Value: "0x5678"},
				{Type: OperandShift, Value: "16", ShiftType: "lsl"},
			},
			// MOVZ: (1<<31)|(0b10100101<<23)|(1<<21)|(0x5678<<5)|2 = 0xD2AACF02
			expectedHex: "02cfaad2",
		},
		{
			name: "movz x0, #0, lsl #48",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "0"},
				{Type: OperandShift, Value: "48", ShiftType: "lsl"},
			},
			// hw=3: (1<<31)|(0b10100101<<23)|(3<<21)|(0<<5)|0 = 0xD2E00000
			expectedHex: "0000e0d2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "movz",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeMovk(t *testing.T) {
	tests := []struct {
		name        string
		operands    []*Operand
		expectedHex string
	}{
		{
			name: "movk x1, #0x5678",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandImmediate, Value: "0x5678"},
			},
			// MOVK: (1<<31)|(0b11100101<<23)|(0<<21)|(0x5678<<5)|1 = 0xF28ACF01
			expectedHex: "01cf8af2",
		},
		{
			name: "movk x2, #0xABCD, lsl #32",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x2"},
				{Type: OperandImmediate, Value: "0xABCD"},
				{Type: OperandShift, Value: "32", ShiftType: "lsl"},
			},
			// MOVK: (1<<31)|(0b11100101<<23)|(2<<21)|(0xABCD<<5)|2 = 0xF2D579A2
			expectedHex: "a279d5f2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "movk",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeAdds(t *testing.T) {
	tests := []struct {
		name        string
		operands    []*Operand
		expectedHex string
	}{
		{
			name: "adds x2, x0, #5 (immediate)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x2"},
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "5"},
			},
			// ARM64_ADDS_IMM|(5<<10)|(0<<5)|2 = 0xB1001402
			expectedHex: "021400b1",
		},
		{
			name: "adds x2, x0, x1 (register)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x2"},
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
			},
			// ARM64_ADDS_REG|(1<<16)|(0<<5)|2 = 0xAB010002
			expectedHex: "020001ab",
		},
		{
			name: "adds x0, x0, #4095 (max imm12)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "4095"},
			},
			// ARM64_ADDS_IMM|(4095<<10)|(0<<5)|0 = 0xB13FFC00
			expectedHex: "00fc3fb1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "adds",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeSubs(t *testing.T) {
	tests := []struct {
		name        string
		operands    []*Operand
		expectedHex string
	}{
		{
			name: "subs x2, x0, #5 (immediate)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x2"},
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "5"},
			},
			// ARM64_SUBS_IMM|(5<<10)|(0<<5)|2 = 0xF1001402
			expectedHex: "021400f1",
		},
		{
			name: "subs x2, x0, x1 (register)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x2"},
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
			},
			// ARM64_SUBS_REG|(1<<16)|(0<<5)|2 = 0xEB010002
			expectedHex: "020001eb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "subs",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeSmulh(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	inst := &Instruction{
		Mnemonic: "smulh",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	result, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// ARM64_SMULH|(1<<16)|(0<<5)|2 = 0x9B417C02
	expected := "027c419b"
	got := hex.EncodeToString(result)
	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeUmulh(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	inst := &Instruction{
		Mnemonic: "umulh",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x2"},
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandRegister, Value: "x1"},
		},
	}

	result, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// ARM64_UMULH|(1<<16)|(0<<5)|2 = 0x9BC17C02
	expected := "027cc19b"
	got := hex.EncodeToString(result)
	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeLdrb(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "ldrb w0, [x1]",
			rt:          "w0",
			base:        "x1",
			offset:      "",
			expectedHex: "20004039", // ARM64_LDRB_UOFF|(0)|(1<<5)|0 = 0x39400020
		},
		{
			name:        "ldrb w0, [x1, #10]",
			rt:          "w0",
			base:        "x1",
			offset:      "10",
			expectedHex: "20284039", // ARM64_LDRB_UOFF|(10<<10)|(1<<5)|0 = 0x39402820
		},
		{
			name:        "ldrb w2, [sp, #0]",
			rt:          "w2",
			base:        "sp",
			offset:      "0",
			expectedHex: "e2034039", // ARM64_LDRB_UOFF|(0)|(31<<5)|2 = 0x394003E2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "ldrb",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset},
				},
				Line:   1,
				Column: 1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeLdrb_Errors(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		operands []*Operand
	}{
		{
			name: "offset out of range (negative)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "w0"},
				{Type: OperandMemory, Base: "x1", Offset: "-1"},
			},
		},
		{
			name: "offset out of range (too large)",
			operands: []*Operand{
				{Type: OperandRegister, Value: "w0"},
				{Type: OperandMemory, Base: "x1", Offset: "4096"},
			},
		},
		{
			name: "wrong operand count",
			operands: []*Operand{
				{Type: OperandRegister, Value: "w0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instruction{
				Mnemonic: "ldrb",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}
			_, err := encoder.Encode(inst, 0)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestEncodeStrb(t *testing.T) {
	tests := []struct {
		name        string
		rt          string
		base        string
		offset      string
		expectedHex string
	}{
		{
			name:        "strb w0, [x1]",
			rt:          "w0",
			base:        "x1",
			offset:      "",
			expectedHex: "20000039", // ARM64_STRB_UOFF|(0)|(1<<5)|0 = 0x39000020
		},
		{
			name:        "strb w0, [x1, #5]",
			rt:          "w0",
			base:        "x1",
			offset:      "5",
			expectedHex: "20140039", // ARM64_STRB_UOFF|(5<<10)|(1<<5)|0 = 0x39001420
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "strb",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rt},
					{Type: OperandMemory, Base: tt.base, Offset: tt.offset},
				},
				Line:   1,
				Column: 1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeAdrp(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		label       string
		labelAddr   uint64
		instAddr    uint64
		expectedHex string
	}{
		{
			name:      "adrp x0, label (same page)",
			rd:        "x0",
			label:     "test",
			labelAddr: 0x100,
			instAddr:  0x0,
			// targetPage=0, currentPage=0, pageOffset=0
			// encoding: (1<<31)|(0<<29)|(0b10000<<24)|(0<<5)|0 = 0x90000000
			expectedHex: "00000090",
		},
		{
			name:      "adrp x0, label (next page)",
			rd:        "x0",
			label:     "test",
			labelAddr: 0x2000,
			instAddr:  0x0,
			// targetPage=0x2000, currentPage=0, pageOffset=2
			// immlo=0, immhi=0 (pageOffset 2: immlo=2&0x3=2, immhi=(2>>2)=0)
			// encoding: (1<<31)|(2<<29)|(0b10000<<24)|(0<<5)|0 = 0xD0000000
			expectedHex: "000000d0",
		},
		{
			name:      "adrp x1, label (8 pages forward)",
			rd:        "x1",
			label:     "test",
			labelAddr: 0x8000,
			instAddr:  0x0,
			// targetPage=0x8000, currentPage=0, pageOffset=8
			// immlo=8&0x3=0, immhi=(8>>2)&0x7FFFF=2
			// encoding: (1<<31)|(0<<29)|(0b10000<<24)|(2<<5)|1 = 0x90000041
			expectedHex: "41000090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolTable := NewSymbolTable()
			symbolTable.Define(tt.label, tt.labelAddr, SectionText, 1, 1)

			encoder := NewEncoder(symbolTable, nil)
			inst := &Instruction{
				Mnemonic: "adrp",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandLabel, Value: tt.label + "@PAGE"},
				},
				Line:   1,
				Column: 1,
			}

			result, err := encoder.Encode(inst, tt.instAddr)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeSvc(t *testing.T) {
	tests := []struct {
		name        string
		operands    []*Operand
		expectedHex string
	}{
		{
			name:        "svc #0",
			operands:    []*Operand{{Type: OperandImmediate, Value: "0"}},
			expectedHex: "010000d4", // ARM64_SVC = 0xD4000001
		},
		{
			name:        "svc #0x80",
			operands:    []*Operand{{Type: OperandImmediate, Value: "0x80"}},
			expectedHex: "011000d4", // ARM64_SVC|(0x80<<5) = 0xD4001001
		},
		{
			name:        "svc with no operand (defaults to 0)",
			operands:    []*Operand{},
			expectedHex: "010000d4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "svc",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeSvc_Errors(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		operands []*Operand
	}{
		{
			name:     "immediate too large",
			operands: []*Operand{{Type: OperandImmediate, Value: "70000"}},
		},
		{
			name:     "non-immediate operand",
			operands: []*Operand{{Type: OperandRegister, Value: "x0"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instruction{
				Mnemonic: "svc",
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}
			_, err := encoder.Encode(inst, 0)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestEncodeErrors_General(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		mnemonic string
		operands []*Operand
	}{
		{
			name:     "unsupported instruction",
			mnemonic: "xyzzy",
			operands: []*Operand{},
		},
		{
			name:     "add with too few operands",
			mnemonic: "add",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
			},
		},
		{
			name:     "mul with too few operands",
			mnemonic: "mul",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
			},
		},
		{
			name:     "sdiv with too few operands",
			mnemonic: "sdiv",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
			},
		},
		{
			name:     "adds immediate out of range",
			mnemonic: "adds",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandImmediate, Value: "5000"},
			},
		},
		{
			name:     "sub immediate out of range",
			mnemonic: "sub",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandImmediate, Value: "5000"},
			},
		},
		{
			name:     "subs immediate out of range",
			mnemonic: "subs",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandImmediate, Value: "-1"},
			},
		},
		{
			name:     "smulh with wrong operand count",
			mnemonic: "smulh",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
			},
		},
		{
			name:     "umulh with wrong operand count",
			mnemonic: "umulh",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
			},
		},
		{
			name:     "movz immediate out of range",
			mnemonic: "movz",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "70000"},
			},
		},
		{
			name:     "movz invalid shift amount",
			mnemonic: "movz",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "1"},
				{Type: OperandShift, Value: "8", ShiftType: "lsl"},
			},
		},
		{
			name:     "movk immediate out of range",
			mnemonic: "movk",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandImmediate, Value: "70000"},
			},
		},
		{
			name:     "neg with wrong operand count",
			mnemonic: "neg",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
			},
		},
		{
			name:     "msub with wrong operand count",
			mnemonic: "msub",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandRegister, Value: "x2"},
			},
		},
		{
			name:     "invalid register name",
			mnemonic: "add",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandRegister, Value: "x1"},
				{Type: OperandRegister, Value: "x99"},
			},
		},
		{
			name:     "strb offset out of range",
			mnemonic: "strb",
			operands: []*Operand{
				{Type: OperandRegister, Value: "w0"},
				{Type: OperandMemory, Base: "x1", Offset: "5000"},
			},
		},
		{
			name:     "ldrh non-memory operand",
			mnemonic: "ldrh",
			operands: []*Operand{
				{Type: OperandRegister, Value: "w0"},
				{Type: OperandRegister, Value: "x1"},
			},
		},
		{
			name:     "adr with undefined label",
			mnemonic: "adr",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandLabel, Value: "nonexistent"},
			},
		},
		{
			name:     "adrp with undefined label",
			mnemonic: "adrp",
			operands: []*Operand{
				{Type: OperandRegister, Value: "x0"},
				{Type: OperandLabel, Value: "nonexistent@PAGE"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instruction{
				Mnemonic: tt.mnemonic,
				Operands: tt.operands,
				Line:     1,
				Column:   1,
			}
			_, err := encoder.Encode(inst, 0)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestEncodeDataErrors(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		dataType string
		value    string
	}{
		{
			name:     "space with negative size",
			dataType: "space",
			value:    "-1",
		},
		{
			name:     "space exceeds max size",
			dataType: "space",
			value:    "2000000",
		},
		{
			name:     "zero with negative size",
			dataType: "zero",
			value:    "-5",
		},
		{
			name:     "byte with invalid value",
			dataType: "byte",
			value:    "abc",
		},
		{
			name:     "quad with invalid value",
			dataType: "quad",
			value:    "not_a_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &DataDeclaration{
				Type:  tt.dataType,
				Value: tt.value,
			}
			_, _, err := encoder.EncodeDataWithRelocations(data, 0)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestEncodeAddImmediate(t *testing.T) {
	tests := []struct {
		name        string
		rd          string
		rn          string
		imm         string
		expectedHex string
	}{
		{
			name:        "add x0, x0, #1",
			rd:          "x0",
			rn:          "x0",
			imm:         "1",
			expectedHex: "00040091", // (1<<31)|(0b0010001<<24)|(1<<10)|(0<<5)|0 = 0x91000400
		},
		{
			name:        "add x1, x2, #4095",
			rd:          "x1",
			rn:          "x2",
			imm:         "4095",
			expectedHex: "41fc3f91", // (1<<31)|(0b0010001<<24)|(4095<<10)|(2<<5)|1 = 0x913FFC41
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder(NewSymbolTable(), nil)
			inst := &Instruction{
				Mnemonic: "add",
				Operands: []*Operand{
					{Type: OperandRegister, Value: tt.rd},
					{Type: OperandRegister, Value: tt.rn},
					{Type: OperandImmediate, Value: tt.imm},
				},
				Line:   1,
				Column: 1,
			}

			result, err := encoder.Encode(inst, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}

			got := hex.EncodeToString(result)
			if got != tt.expectedHex {
				t.Errorf("Expected %s, got %s", tt.expectedHex, got)
			}
		})
	}
}

func TestEncodeResolveImmediateWithConstants(t *testing.T) {
	constants := map[string]int64{
		"STACK_SIZE": 256,
		"MAX_COUNT":  100,
	}
	encoder := NewEncoder(NewSymbolTable(), constants)

	// Test that constant names are resolved via ResolveImmediate (used by movz/movk)
	inst := &Instruction{
		Mnemonic: "movz",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandImmediate, Value: "MAX_COUNT"},
		},
		Line:   1,
		Column: 1,
	}

	result, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// movz x0, #100: (1<<31)|(0b10100101<<23)|(0<<21)|(100<<5)|0 = 0xD2800C80
	expected := "800c80d2"
	got := hex.EncodeToString(result)
	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEncodeLdrRegisterOffset(t *testing.T) {
	// Test LDR with register offset: ldr x0, [x1, x2]
	encoder := NewEncoder(NewSymbolTable(), nil)

	inst := &Instruction{
		Mnemonic: "ldr",
		Operands: []*Operand{
			{Type: OperandRegister, Value: "x0"},
			{Type: OperandMemory, Base: "x1", IndexReg: "x2"},
		},
		Line:   1,
		Column: 1,
	}

	result, err := encoder.Encode(inst, 0)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Should produce valid bytes (4 bytes for ARM64)
	if len(result) != 4 {
		t.Errorf("Expected 4 bytes, got %d", len(result))
	}
}

func TestEncodeWithRelocations_NoRelocation(t *testing.T) {
	// Test that non-branch instructions produce no relocation
	st := NewSymbolTable()
	encoder := NewEncoder(st, nil)

	inst := &Instruction{
		Mnemonic: "ret",
		Operands: []*Operand{},
		Line:     1,
		Column:   1,
	}

	result, err := encoder.EncodeWithRelocations(inst, 0)
	if err != nil {
		t.Fatalf("EncodeWithRelocations failed: %v", err)
	}

	if result.Relocation != nil {
		t.Errorf("Expected no relocation for ret, got %+v", result.Relocation)
	}

	expectedHex := "c0035fd6" // RET
	if hex.EncodeToString(result.Bytes) != expectedHex {
		t.Errorf("Expected %s, got %s", expectedHex, hex.EncodeToString(result.Bytes))
	}
}

func TestEncodeMultipleDataValues(t *testing.T) {
	encoder := NewEncoder(NewSymbolTable(), nil)

	tests := []struct {
		name     string
		dataType string
		value    string
		expected int // expected byte count
	}{
		{
			name:     "multiple bytes",
			dataType: "byte",
			value:    "1,2,3,4",
			expected: 4,
		},
		{
			name:     "multiple words",
			dataType: "word",
			value:    "100,200",
			expected: 8, // 2 * 4 bytes
		},
		{
			name:     "multiple quads",
			dataType: "quad",
			value:    "1000,2000,3000",
			expected: 24, // 3 * 8 bytes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &DataDeclaration{
				Type:  tt.dataType,
				Value: tt.value,
			}
			result, _, err := encoder.EncodeDataWithRelocations(data, 0)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("Expected %d bytes, got %d", tt.expected, len(result))
			}
		})
	}
}

// Verify the ARM64 instruction constants match the expected base values
func TestARM64Constants(t *testing.T) {
	tests := []struct {
		name     string
		constant uint32
		expected uint32
	}{
		{"ARM64_NOP", ARM64_NOP, 0xD503201F},
		{"ARM64_RET", ARM64_RET, 0xD65F03C0},
		{"ARM64_SVC", ARM64_SVC, 0xD4000001},
		{"ARM64_B", ARM64_B, 0x14000000},
		{"ARM64_BL", ARM64_BL, 0x94000000},
		{"ARM64_BR", ARM64_BR, 0xD61F0000},
		{"ARM64_CBZ_X", ARM64_CBZ_X, 0xB4000000},
		{"ARM64_CBNZ_X", ARM64_CBNZ_X, 0xB5000000},
		{"ARM64_ADDS_IMM", ARM64_ADDS_IMM, 0xB1000000},
		{"ARM64_SUB_IMM", ARM64_SUB_IMM, 0xD1000000},
		{"ARM64_SUBS_IMM", ARM64_SUBS_IMM, 0xF1000000},
		{"ARM64_ADDS_REG", ARM64_ADDS_REG, 0xAB000000},
		{"ARM64_SUBS_REG", ARM64_SUBS_REG, 0xEB000000},
		{"ARM64_LDR_UOFF", ARM64_LDR_UOFF, 0xF9400000},
		{"ARM64_STR_UOFF", ARM64_STR_UOFF, 0xF9000000},
		{"ARM64_LDRB_UOFF", ARM64_LDRB_UOFF, 0x39400000},
		{"ARM64_STRB_UOFF", ARM64_STRB_UOFF, 0x39000000},
		{"ARM64_LDRH_UOFF", ARM64_LDRH_UOFF, 0x79400000},
		{"ARM64_STRH_UOFF", ARM64_STRH_UOFF, 0x79000000},
		{"ARM64_IMM26_MASK", ARM64_IMM26_MASK, 0x03FFFFFF},
		{"ARM64_BRANCH_OP_MASK", ARM64_BRANCH_OP_MASK, 0xFC000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s: expected 0x%08X, got 0x%08X", tt.name, tt.expected, tt.constant)
			}
		})
	}

	// Verify masks are complements
	if ARM64_IMM26_MASK|ARM64_BRANCH_OP_MASK != 0xFFFFFFFF {
		t.Error("IMM26_MASK and BRANCH_OP_MASK should be complements")
	}

	// Verify NOP encodes to expected bytes
	nopBytes := EncodeLittleEndian(ARM64_NOP)
	expectedNop := []byte{0x1f, 0x20, 0x03, 0xd5}
	if !bytes.Equal(nopBytes, expectedNop) {
		t.Errorf("NOP bytes: expected %x, got %x", expectedNop, nopBytes)
	}

	// Verify RET encodes to expected bytes
	retBytes := EncodeLittleEndian(ARM64_RET)
	expectedRet := []byte{0xc0, 0x03, 0x5f, 0xd6}
	if !bytes.Equal(retBytes, expectedRet) {
		t.Errorf("RET bytes: expected %x, got %x", expectedRet, retBytes)
	}
}

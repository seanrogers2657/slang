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

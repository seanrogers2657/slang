package slasm

import (
	"testing"
)

func TestSymbolTable_DefineAndLookup(t *testing.T) {
	st := NewSymbolTable()

	// Define a symbol
	err := st.Define("main", 0x1000, SectionText, 0, 0)
	if err != nil {
		t.Fatalf("failed to define symbol: %v", err)
	}

	// Lookup the symbol
	symbol, exists := st.Lookup("main")
	if !exists {
		t.Fatal("symbol 'main' should exist")
	}

	if symbol.Name != "main" {
		t.Errorf("expected name 'main', got %q", symbol.Name)
	}

	if symbol.Address != 0x1000 {
		t.Errorf("expected address 0x1000, got 0x%x", symbol.Address)
	}

	if symbol.Section != SectionText {
		t.Errorf("expected section Text, got %v", symbol.Section)
	}
}

func TestSymbolTable_LookupNonexistent(t *testing.T) {
	st := NewSymbolTable()

	_, exists := st.Lookup("nonexistent")
	if exists {
		t.Error("nonexistent symbol should not exist")
	}
}

func TestSymbolTable_MultipleSymbols(t *testing.T) {
	st := NewSymbolTable()

	symbols := []struct {
		name    string
		address uint64
		section SectionType
	}{
		{"_start", 0x0, SectionText},
		{"main", 0x100, SectionText},
		{"buffer", 0x0, SectionData},
		{"newline", 0x20, SectionData},
	}

	// Define all symbols
	for _, sym := range symbols {
		err := st.Define(sym.name, sym.address, sym.section, 0, 0)
		if err != nil {
			t.Fatalf("failed to define symbol %q: %v", sym.name, err)
		}
	}

	// Lookup all symbols
	for _, expected := range symbols {
		symbol, exists := st.Lookup(expected.name)
		if !exists {
			t.Errorf("symbol %q should exist", expected.name)
			continue
		}

		if symbol.Address != expected.address {
			t.Errorf("symbol %q: expected address 0x%x, got 0x%x",
				expected.name, expected.address, symbol.Address)
		}

		if symbol.Section != expected.section {
			t.Errorf("symbol %q: expected section %v, got %v",
				expected.name, expected.section, symbol.Section)
		}
	}
}

func TestSymbolTable_MarkGlobal(t *testing.T) {
	st := NewSymbolTable()

	// Define a symbol
	err := st.Define("_start", 0x0, SectionText, 0, 0)
	if err != nil {
		t.Fatalf("failed to define symbol: %v", err)
	}

	// Initially should not be global
	symbol, _ := st.Lookup("_start")
	if symbol.Global {
		t.Error("symbol should not be global initially")
	}

	// Mark as global
	st.MarkGlobal("_start")

	// Now should be global
	symbol, _ = st.Lookup("_start")
	if !symbol.Global {
		t.Error("symbol should be global after MarkGlobal")
	}
}

func TestSymbolTable_MarkGlobalNonexistent(t *testing.T) {
	st := NewSymbolTable()

	// Marking a nonexistent symbol as global creates a forward reference
	st.MarkGlobal("nonexistent")

	// Symbol should exist as a forward reference
	sym, exists := st.Lookup("nonexistent")
	if !exists {
		t.Error("MarkGlobal should create forward reference for nonexistent symbol")
	}

	if !sym.Global {
		t.Error("symbol should be marked as global")
	}

	if sym.Defined {
		t.Error("symbol should not be defined yet (forward reference)")
	}
}

func TestSymbolTable_All(t *testing.T) {
	st := NewSymbolTable()

	expectedNames := []string{"_start", "main", "buffer"}
	for i, name := range expectedNames {
		err := st.Define(name, uint64(i*0x100), SectionText, 0, 0)
		if err != nil {
			t.Fatalf("failed to define symbol %q: %v", name, err)
		}
	}

	allSymbols := st.All()

	if len(allSymbols) != len(expectedNames) {
		t.Fatalf("expected %d symbols, got %d", len(expectedNames), len(allSymbols))
	}

	// Check that all expected symbols are present
	foundNames := make(map[string]bool)
	for _, symbol := range allSymbols {
		foundNames[symbol.Name] = true
	}

	for _, expected := range expectedNames {
		if !foundNames[expected] {
			t.Errorf("expected symbol %q not found in All()", expected)
		}
	}
}

func TestSymbolTable_DuplicateDefinition(t *testing.T) {
	st := NewSymbolTable()

	// Define a symbol
	err := st.Define("main", 0x1000, SectionText, 0, 0)
	if err != nil {
		t.Fatalf("failed to define symbol: %v", err)
	}

	// Try to define it again - should return an error
	err = st.Define("main", 0x2000, SectionText, 0, 0)
	if err == nil {
		t.Error("expected error when defining duplicate symbol")
	}
}

func TestSymbolTable_AddressZero(t *testing.T) {
	st := NewSymbolTable()

	// Address 0 should be valid
	err := st.Define("start", 0, SectionText, 0, 0)
	if err != nil {
		t.Fatalf("failed to define symbol at address 0: %v", err)
	}

	symbol, exists := st.Lookup("start")
	if !exists {
		t.Fatal("symbol should exist")
	}

	if symbol.Address != 0 {
		t.Errorf("expected address 0, got %d", symbol.Address)
	}
}

func TestSymbolTable_SectionTypes(t *testing.T) {
	st := NewSymbolTable()

	tests := []struct {
		name    string
		section SectionType
	}{
		{"text_symbol", SectionText},
		{"data_symbol", SectionData},
	}

	for _, tt := range tests {
		err := st.Define(tt.name, 0, tt.section, 0, 0)
		if err != nil {
			t.Fatalf("failed to define symbol %q: %v", tt.name, err)
		}

		symbol, exists := st.Lookup(tt.name)
		if !exists {
			t.Fatalf("symbol %q should exist", tt.name)
		}

		if symbol.Section != tt.section {
			t.Errorf("symbol %q: expected section %v, got %v",
				tt.name, tt.section, symbol.Section)
		}
	}
}

func TestSymbolTable_GlobalFlag(t *testing.T) {
	st := NewSymbolTable()

	// Define two symbols
	st.Define("local", 0x100, SectionText, 0, 0)
	st.Define("global", 0x200, SectionText, 0, 0)

	// Mark one as global
	st.MarkGlobal("global")

	// Check flags
	localSym, _ := st.Lookup("local")
	if localSym.Global {
		t.Error("local symbol should not be global")
	}

	globalSym, _ := st.Lookup("global")
	if !globalSym.Global {
		t.Error("global symbol should be global")
	}

	// Get all global symbols
	allSymbols := st.All()
	globalCount := 0
	for _, sym := range allSymbols {
		if sym.Global {
			globalCount++
		}
	}

	if globalCount != 1 {
		t.Errorf("expected 1 global symbol, got %d", globalCount)
	}
}

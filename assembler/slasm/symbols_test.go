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

func TestSymbolTable_AdjustAddresses_AllSections(t *testing.T) {
	st := NewSymbolTable()

	// Define symbols in different sections
	st.Define("text_sym", 0x10, SectionText, 0, 0)
	st.Define("data_sym", 0x20, SectionData, 0, 0)
	st.Define("bss_sym", 0x30, SectionBSS, 0, 0)

	textBase := uint64(0x1000)
	dataBase := uint64(0x2000)

	// Adjust addresses
	st.AdjustAddresses(textBase, dataBase)

	// Check text symbol
	textSym, _ := st.Lookup("text_sym")
	expectedTextAddr := uint64(0x10 + 0x1000)
	if textSym.Address != expectedTextAddr {
		t.Errorf("text_sym: expected address 0x%x, got 0x%x", expectedTextAddr, textSym.Address)
	}

	// Check data symbol
	dataSym, _ := st.Lookup("data_sym")
	expectedDataAddr := uint64(0x20 + 0x2000)
	if dataSym.Address != expectedDataAddr {
		t.Errorf("data_sym: expected address 0x%x, got 0x%x", expectedDataAddr, dataSym.Address)
	}

	// Check BSS symbol (should use data base)
	bssSym, _ := st.Lookup("bss_sym")
	expectedBssAddr := uint64(0x30 + 0x2000)
	if bssSym.Address != expectedBssAddr {
		t.Errorf("bss_sym: expected address 0x%x, got 0x%x", expectedBssAddr, bssSym.Address)
	}
}

func TestSymbolTable_AdjustAddresses_UnknownSection(t *testing.T) {
	st := NewSymbolTable()

	// Define a symbol with an unknown section type (simulating external/undefined)
	st.symbols["extern_sym"] = &Symbol{
		Name:    "extern_sym",
		Address: 0x100,
		Section: SectionType(99), // Unknown section type
		Defined: false,
	}

	textBase := uint64(0x1000)
	dataBase := uint64(0x2000)

	// Adjust addresses - should not panic and should not adjust unknown section
	st.AdjustAddresses(textBase, dataBase)

	// Check that the address was NOT adjusted for unknown section
	sym, _ := st.Lookup("extern_sym")
	if sym.Address != 0x100 {
		t.Errorf("extern_sym: expected unchanged address 0x100, got 0x%x", sym.Address)
	}
}

func TestSymbolTable_SectionBSS(t *testing.T) {
	st := NewSymbolTable()

	// Define a BSS symbol
	err := st.Define("bss_buffer", 0x0, SectionBSS, 1, 1)
	if err != nil {
		t.Fatalf("failed to define BSS symbol: %v", err)
	}

	sym, exists := st.Lookup("bss_buffer")
	if !exists {
		t.Fatal("BSS symbol should exist")
	}

	if sym.Section != SectionBSS {
		t.Errorf("expected section BSS, got %v", sym.Section)
	}
}

func TestSymbolState_String(t *testing.T) {
	tests := []struct {
		state    SymbolState
		expected string
	}{
		{SymbolUndefined, "undefined"},
		{SymbolDefined, "defined"},
		{SymbolForwardRef, "forward-ref"},
		{SymbolExtern, "extern"},
		{SymbolState(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("SymbolState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSymbolTable_SymbolState_Define(t *testing.T) {
	st := NewSymbolTable()

	// Define a symbol - should have SymbolDefined state
	err := st.Define("main", 0x1000, SectionText, 1, 1)
	if err != nil {
		t.Fatalf("failed to define symbol: %v", err)
	}

	sym, _ := st.Lookup("main")
	if sym.State != SymbolDefined {
		t.Errorf("expected state SymbolDefined, got %v", sym.State)
	}
}

func TestSymbolTable_SymbolState_ForwardRef(t *testing.T) {
	st := NewSymbolTable()

	// Reference a symbol before defining it - should create forward reference
	sym := st.Reference("future_label", 0x100)

	if sym.State != SymbolForwardRef {
		t.Errorf("expected state SymbolForwardRef, got %v", sym.State)
	}

	if sym.Defined {
		t.Error("forward reference should not be defined")
	}

	// Now define the symbol - state should change to SymbolDefined
	err := st.Define("future_label", 0x200, SectionText, 5, 1)
	if err != nil {
		t.Fatalf("failed to define forward-referenced symbol: %v", err)
	}

	sym, _ = st.Lookup("future_label")
	if sym.State != SymbolDefined {
		t.Errorf("after definition, expected state SymbolDefined, got %v", sym.State)
	}

	if !sym.Defined {
		t.Error("symbol should be defined after Define()")
	}
}

func TestSymbolTable_SymbolState_Extern(t *testing.T) {
	st := NewSymbolTable()

	// Mark a symbol as extern
	st.MarkExtern("printf")

	sym, exists := st.Lookup("printf")
	if !exists {
		t.Fatal("extern symbol should exist")
	}

	if sym.State != SymbolExtern {
		t.Errorf("expected state SymbolExtern, got %v", sym.State)
	}

	if sym.Defined {
		t.Error("extern symbol should not be defined locally")
	}
}

func TestSymbolTable_MarkExtern_ExistingSymbol(t *testing.T) {
	st := NewSymbolTable()

	// First create a forward reference
	st.Reference("lib_func", 0x100)

	// Then mark it as extern
	st.MarkExtern("lib_func")

	sym, _ := st.Lookup("lib_func")
	if sym.State != SymbolExtern {
		t.Errorf("expected state SymbolExtern after MarkExtern, got %v", sym.State)
	}

	// References should be preserved
	if len(sym.References) != 1 || sym.References[0] != 0x100 {
		t.Error("MarkExtern should preserve existing references")
	}
}

func TestSymbolTable_AllSorted(t *testing.T) {
	st := NewSymbolTable()

	// Add symbols in non-alphabetical order
	names := []string{"zebra", "alpha", "middle", "beta"}
	for i, name := range names {
		err := st.Define(name, uint64(i*0x100), SectionText, i, 1)
		if err != nil {
			t.Fatalf("failed to define symbol %q: %v", name, err)
		}
	}

	// AllSorted should return symbols in alphabetical order
	sorted := st.AllSorted()

	if len(sorted) != len(names) {
		t.Fatalf("expected %d symbols, got %d", len(names), len(sorted))
	}

	expectedOrder := []string{"alpha", "beta", "middle", "zebra"}
	for i, expected := range expectedOrder {
		if sorted[i].Name != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, sorted[i].Name)
		}
	}
}

func TestSymbolTable_AllSorted_Deterministic(t *testing.T) {
	st := NewSymbolTable()

	// Add multiple symbols
	for i := 0; i < 10; i++ {
		name := string(rune('j' - i)) // j, i, h, g, f, e, d, c, b, a
		st.Define(name, uint64(i), SectionText, i, 1)
	}

	// Call AllSorted multiple times - should always return same order
	first := st.AllSorted()
	for run := 0; run < 10; run++ {
		current := st.AllSorted()
		for i := range first {
			if first[i].Name != current[i].Name {
				t.Errorf("run %d: AllSorted() not deterministic at position %d: %q vs %q",
					run, i, first[i].Name, current[i].Name)
			}
		}
	}
}

func TestSymbolTable_AllSorted_Empty(t *testing.T) {
	st := NewSymbolTable()

	sorted := st.AllSorted()

	if len(sorted) != 0 {
		t.Errorf("expected empty slice, got %d symbols", len(sorted))
	}
}

func TestSymbolTable_MarkGlobal_SetsForwardRefState(t *testing.T) {
	st := NewSymbolTable()

	// MarkGlobal on non-existent symbol should create forward ref
	st.MarkGlobal("_start")

	sym, _ := st.Lookup("_start")
	if sym.State != SymbolForwardRef {
		t.Errorf("MarkGlobal on new symbol: expected state SymbolForwardRef, got %v", sym.State)
	}
}

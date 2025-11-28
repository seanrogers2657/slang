package asm

// SymbolTable tracks label definitions and their addresses
type SymbolTable struct {
	symbols map[string]*Symbol
}

// Symbol represents a symbol (label) in the assembly
type Symbol struct {
	Name    string
	Address uint64
	Section SectionType
	Global  bool // whether this symbol is marked as .global
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		symbols: make(map[string]*Symbol),
	}
}

// Define adds a symbol definition to the table
func (st *SymbolTable) Define(name string, address uint64, section SectionType) error {
	// TODO: Implement symbol definition
	// Check for duplicate definitions
	// Add symbol to table
	return nil
}

// Lookup finds a symbol by name
func (st *SymbolTable) Lookup(name string) (*Symbol, bool) {
	symbol, exists := st.symbols[name]
	return symbol, exists
}

// MarkGlobal marks a symbol as global
func (st *SymbolTable) MarkGlobal(name string) {
	if symbol, exists := st.symbols[name]; exists {
		symbol.Global = true
	}
}

// All returns all symbols in the table
func (st *SymbolTable) All() []*Symbol {
	symbols := make([]*Symbol, 0, len(st.symbols))
	for _, symbol := range st.symbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

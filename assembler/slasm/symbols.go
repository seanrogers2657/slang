package slasm

import "fmt"

// SymbolTable tracks label definitions and their addresses
type SymbolTable struct {
	symbols map[string]*Symbol
}

// Symbol represents a symbol (label) in the assembly
type Symbol struct {
	Name       string
	Address    uint64
	Section    SectionType
	Global     bool     // whether this symbol is marked as .global
	Defined    bool     // whether the symbol has been defined
	References []uint64 // addresses where this symbol is referenced
	Line       int      // source line where symbol was defined
	Column     int      // source column where symbol was defined
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		symbols: make(map[string]*Symbol),
	}
}

// Define adds a symbol definition to the table
func (st *SymbolTable) Define(name string, address uint64, section SectionType, line, column int) error {
	// Check if symbol already exists
	if existing, exists := st.symbols[name]; exists {
		if existing.Defined {
			// Duplicate definition error
			return fmt.Errorf("duplicate symbol '%s' at line %d:%d (previously defined at line %d:%d)",
				name, line, column, existing.Line, existing.Column)
		}
		// Update existing forward reference
		existing.Defined = true
		existing.Address = address
		existing.Section = section
		existing.Line = line
		existing.Column = column
		return nil
	}

	// Add new symbol to table
	st.symbols[name] = &Symbol{
		Name:       name,
		Address:    address,
		Section:    section,
		Global:     false,
		Defined:    true,
		References: []uint64{},
		Line:       line,
		Column:     column,
	}

	return nil
}

// Lookup finds a symbol by name
func (st *SymbolTable) Lookup(name string) (*Symbol, bool) {
	symbol, exists := st.symbols[name]
	return symbol, exists
}

// Reference records a reference to a symbol at the given address
// If the symbol doesn't exist yet, it creates a forward reference
func (st *SymbolTable) Reference(name string, address uint64) *Symbol {
	if symbol, exists := st.symbols[name]; exists {
		symbol.References = append(symbol.References, address)
		return symbol
	}

	// Create forward reference
	symbol := &Symbol{
		Name:       name,
		Address:    0,
		Section:    SectionText,
		Global:     false,
		Defined:    false,
		References: []uint64{address},
		Line:       0,
		Column:     0,
	}
	st.symbols[name] = symbol
	return symbol
}

// MarkGlobal marks a symbol as global
func (st *SymbolTable) MarkGlobal(name string) {
	if symbol, exists := st.symbols[name]; exists {
		symbol.Global = true
	} else {
		// Create forward reference for global symbol
		st.symbols[name] = &Symbol{
			Name:       name,
			Global:     true,
			Defined:    false,
			References: []uint64{},
		}
	}
}

// UndefinedSymbols returns all symbols that have been referenced but not defined
func (st *SymbolTable) UndefinedSymbols() []*Symbol {
	var undefined []*Symbol
	for _, symbol := range st.symbols {
		if !symbol.Defined && len(symbol.References) > 0 {
			undefined = append(undefined, symbol)
		}
	}
	return undefined
}

// All returns all symbols in the table
func (st *SymbolTable) All() []*Symbol {
	symbols := make([]*Symbol, 0, len(st.symbols))
	for _, symbol := range st.symbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// AdjustAddresses adjusts symbol addresses based on section base addresses
// This is called after Mach-O layout is calculated to convert relative
// addresses to absolute VM addresses
func (st *SymbolTable) AdjustAddresses(textBase, dataBase uint64) {
	for _, symbol := range st.symbols {
		if symbol.Section == SectionText {
			symbol.Address += textBase
		} else if symbol.Section == SectionData {
			symbol.Address += dataBase
		}
	}
}

package slasm

import (
	"fmt"
	"sort"
)

// SymbolState represents the explicit state of a symbol
type SymbolState int

const (
	// SymbolUndefined indicates the symbol has not been defined yet
	SymbolUndefined SymbolState = iota
	// SymbolDefined indicates the symbol has been defined with an address
	SymbolDefined
	// SymbolForwardRef indicates the symbol was referenced before being defined
	SymbolForwardRef
	// SymbolExtern indicates the symbol is an external reference (imported)
	SymbolExtern
)

// String returns a human-readable representation of the symbol state
func (s SymbolState) String() string {
	switch s {
	case SymbolUndefined:
		return "undefined"
	case SymbolDefined:
		return "defined"
	case SymbolForwardRef:
		return "forward-ref"
	case SymbolExtern:
		return "extern"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// SymbolTable tracks label definitions and their addresses
type SymbolTable struct {
	symbols map[string]*Symbol
}

// Symbol represents a symbol (label) in the assembly
type Symbol struct {
	Name       string
	Address    uint64
	Section    SectionType
	Global     bool        // whether this symbol is marked as .global
	State      SymbolState // explicit state of the symbol
	Defined    bool        // whether the symbol has been defined (kept for backward compatibility)
	References []uint64    // addresses where this symbol is referenced
	Line       int         // source line where symbol was defined
	Column     int         // source column where symbol was defined
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
		existing.State = SymbolDefined
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
		State:      SymbolDefined,
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
		State:      SymbolForwardRef,
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
			State:      SymbolForwardRef,
			Defined:    false,
			References: []uint64{},
		}
	}
}

// MarkExtern marks a symbol as external (imported from another object)
func (st *SymbolTable) MarkExtern(name string) {
	if symbol, exists := st.symbols[name]; exists {
		symbol.State = SymbolExtern
	} else {
		// Create extern symbol entry
		st.symbols[name] = &Symbol{
			Name:       name,
			Global:     false,
			State:      SymbolExtern,
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

// AllSorted returns all symbols sorted by name for deterministic iteration
// Use this instead of All() or ForEach() when consistent ordering is required
// (e.g., when generating output that should be reproducible)
func (st *SymbolTable) AllSorted() []*Symbol {
	symbols := st.All()
	sort.Slice(symbols, func(i, j int) bool {
		return symbols[i].Name < symbols[j].Name
	})
	return symbols
}

// Count returns the number of symbols in the table
func (st *SymbolTable) Count() int {
	return len(st.symbols)
}

// ForEach iterates over all symbols, calling the provided function for each
func (st *SymbolTable) ForEach(fn func(name string, sym *Symbol)) {
	for name, sym := range st.symbols {
		fn(name, sym)
	}
}

// AdjustAddresses adjusts symbol addresses based on section base addresses
// This is called after Mach-O layout is calculated to convert relative
// addresses to absolute VM addresses
func (st *SymbolTable) AdjustAddresses(textBase, dataBase uint64) {
	for _, symbol := range st.symbols {
		switch symbol.Section {
		case SectionText:
			symbol.Address += textBase
		case SectionData:
			symbol.Address += dataBase
		case SectionBSS:
			// BSS symbols also use data base (they follow data section)
			symbol.Address += dataBase
		default:
			// Unknown section type - symbol address is not adjusted
			// This could happen for external/undefined symbols
		}
	}
}

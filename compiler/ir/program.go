package ir

import "fmt"

// Program represents a complete IR program.
type Program struct {
	// Functions are all functions in the program.
	Functions []*Function

	// Structs are all struct type definitions.
	Structs []*StructType

	// Globals are global variables.
	Globals []*Global

	// Strings is the string constant pool.
	Strings []string

	// stringIndex maps string values to their index in Strings.
	stringIndex map[string]int
}

// Global represents a global variable.
type Global struct {
	Name string
	Type Type
	Init *Value // initial value (constant), or nil for zero
}

// NewProgram creates a new empty program.
func NewProgram() *Program {
	return &Program{
		stringIndex: make(map[string]int),
	}
}

// NewFunction creates a new function and adds it to the program.
func (p *Program) NewFunction(name string, returnType Type) *Function {
	f := &Function{
		Name:       name,
		ReturnType: returnType,
		Program:    p,
	}
	p.Functions = append(p.Functions, f)
	return f
}

// AddStruct adds a struct type to the program.
func (p *Program) AddStruct(s *StructType) {
	p.Structs = append(p.Structs, s)
}

// AddGlobal adds a global variable to the program.
func (p *Program) AddGlobal(g *Global) {
	p.Globals = append(p.Globals, g)
}

// AddString adds a string to the constant pool and returns its index.
// If the string already exists, returns the existing index.
func (p *Program) AddString(s string) int {
	if idx, ok := p.stringIndex[s]; ok {
		return idx
	}
	idx := len(p.Strings)
	p.Strings = append(p.Strings, s)
	p.stringIndex[s] = idx
	return idx
}

// GetString returns the string at the given index.
func (p *Program) GetString(idx int) string {
	if idx < 0 || idx >= len(p.Strings) {
		return ""
	}
	return p.Strings[idx]
}

// FunctionByName finds a function by name.
func (p *Program) FunctionByName(name string) *Function {
	for _, f := range p.Functions {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// StructByName finds a struct type by name.
func (p *Program) StructByName(name string) *StructType {
	for _, s := range p.Structs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// GlobalByName finds a global variable by name.
func (p *Program) GlobalByName(name string) *Global {
	for _, g := range p.Globals {
		if g.Name == name {
			return g
		}
	}
	return nil
}

// Main returns the main function, or nil if not found.
func (p *Program) Main() *Function {
	return p.FunctionByName("main")
}

// NumFunctions returns the number of functions.
func (p *Program) NumFunctions() int {
	return len(p.Functions)
}

// NumStructs returns the number of struct types.
func (p *Program) NumStructs() int {
	return len(p.Structs)
}

// NumGlobals returns the number of global variables.
func (p *Program) NumGlobals() int {
	return len(p.Globals)
}

// NumStrings returns the number of strings in the constant pool.
func (p *Program) NumStrings() int {
	return len(p.Strings)
}

// UsesPrint returns true if any function calls print().
// Used to determine if print-related data sections are needed.
func (p *Program) UsesPrint() bool {
	for _, f := range p.Functions {
		for _, b := range f.Blocks {
			for _, v := range b.Values {
				if v.Op == OpCall && v.AuxString == "print" {
					return true
				}
			}
		}
	}
	return false
}

// Validate performs basic validation on the program structure.
func (p *Program) Validate() []error {
	var errs []error

	// Check for main function
	if p.Main() == nil {
		errs = append(errs, fmt.Errorf("no main function"))
	}

	// Validate each function
	for _, f := range p.Functions {
		fErrs := f.Validate()
		errs = append(errs, fErrs...)
	}

	// Check for duplicate function names
	seen := make(map[string]bool)
	for _, f := range p.Functions {
		if seen[f.Name] {
			errs = append(errs, fmt.Errorf("duplicate function name: %s", f.Name))
		}
		seen[f.Name] = true
	}

	// Check for duplicate struct names
	seenStructs := make(map[string]bool)
	for _, s := range p.Structs {
		if seenStructs[s.Name] {
			errs = append(errs, fmt.Errorf("duplicate struct name: %s", s.Name))
		}
		seenStructs[s.Name] = true
	}

	return errs
}

package ir

import "fmt"

// Function represents a function in the IR.
type Function struct {
	// Name is the function name.
	Name string

	// Params are the function parameters (OpArg values).
	Params []*Value

	// ReturnType is the function's return type, or nil for void.
	ReturnType Type

	// Blocks are the basic blocks in this function.
	// Blocks[0] is always the entry block.
	Blocks []*Block

	// Program is the program containing this function.
	Program *Program

	// nextValue is the counter for generating value IDs.
	nextValue int

	// nextBlock is the counter for generating block IDs.
	nextBlock int
}

// String returns a human-readable representation of the function.
func (f *Function) String() string {
	if f == nil {
		return "<nil>"
	}
	return f.Name
}

// Signature returns a string representation of the function signature.
func (f *Function) Signature() string {
	result := f.Name + "("
	for i, param := range f.Params {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("v%d: %s", param.ID, param.Type.String())
	}
	result += ")"
	if f.ReturnType != nil {
		result += " -> " + f.ReturnType.String()
	}
	return result
}

// NewBlock creates a new basic block in this function.
func (f *Function) NewBlock(kind BlockKind) *Block {
	b := &Block{
		ID:   f.nextBlockID(),
		Kind: kind,
		Func: f,
	}
	f.Blocks = append(f.Blocks, b)
	return b
}

// Entry returns the entry block (first block) of the function.
func (f *Function) Entry() *Block {
	if len(f.Blocks) == 0 {
		return nil
	}
	return f.Blocks[0]
}

// nextValueID generates the next value ID.
func (f *Function) nextValueID() int {
	id := f.nextValue
	f.nextValue++
	return id
}

// nextBlockID generates the next block ID.
func (f *Function) nextBlockID() int {
	id := f.nextBlock
	f.nextBlock++
	return id
}

// NewParam creates a new parameter value for this function.
func (f *Function) NewParam(typ Type) *Value {
	entry := f.Entry()
	if entry == nil {
		// Create entry block if needed
		entry = f.NewBlock(BlockPlain)
	}

	v := &Value{
		ID:     f.nextValueID(),
		Op:     OpArg,
		Type:   typ,
		AuxInt: int64(len(f.Params)),
		Block:  entry,
	}
	f.Params = append(f.Params, v)
	return v
}

// NumBlocks returns the number of blocks in the function.
func (f *Function) NumBlocks() int {
	return len(f.Blocks)
}

// NumParams returns the number of parameters.
func (f *Function) NumParams() int {
	return len(f.Params)
}

// NumValues returns the total number of values in all blocks.
func (f *Function) NumValues() int {
	count := 0
	for _, b := range f.Blocks {
		count += len(b.Values)
	}
	return count
}

// AllValues returns all values in the function in block order.
func (f *Function) AllValues() []*Value {
	var values []*Value
	for _, b := range f.Blocks {
		values = append(values, b.Values...)
	}
	return values
}

// BlockByID finds a block by its ID.
func (f *Function) BlockByID(id int) *Block {
	for _, b := range f.Blocks {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// ValueByID finds a value by its ID, searching all blocks.
func (f *Function) ValueByID(id int) *Value {
	// Check params first
	for _, p := range f.Params {
		if p.ID == id {
			return p
		}
	}

	// Check all blocks
	for _, b := range f.Blocks {
		for _, v := range b.Values {
			if v.ID == id {
				return v
			}
		}
	}
	return nil
}

// IsVoid returns true if this function returns void.
func (f *Function) IsVoid() bool {
	return f.ReturnType == nil || f.ReturnType.Equal(TypeVoid)
}

// RemoveBlock removes a block from the function.
// This does NOT update predecessor/successor edges.
func (f *Function) RemoveBlock(b *Block) {
	for i, block := range f.Blocks {
		if block == b {
			f.Blocks = append(f.Blocks[:i], f.Blocks[i+1:]...)
			b.Func = nil
			return
		}
	}
}

// Validate performs basic validation on the function structure.
func (f *Function) Validate() []error {
	var errs []error

	if f.Name == "" {
		errs = append(errs, fmt.Errorf("function has no name"))
	}

	if len(f.Blocks) == 0 {
		errs = append(errs, fmt.Errorf("function %s has no blocks", f.Name))
		return errs
	}

	// Check that entry block has no predecessors
	if len(f.Blocks[0].Preds) > 0 {
		errs = append(errs, fmt.Errorf("entry block has predecessors"))
	}

	// Check each block
	for _, b := range f.Blocks {
		if b.Func != f {
			errs = append(errs, fmt.Errorf("block b%d has wrong function pointer", b.ID))
		}

		// Check terminator
		switch b.Kind {
		case BlockPlain:
			if len(b.Succs) != 1 {
				errs = append(errs, fmt.Errorf("block b%d is Plain but has %d successors", b.ID, len(b.Succs)))
			}
		case BlockIf:
			if len(b.Succs) != 2 {
				errs = append(errs, fmt.Errorf("block b%d is If but has %d successors", b.ID, len(b.Succs)))
			}
			if b.Control == nil {
				errs = append(errs, fmt.Errorf("block b%d is If but has no control value", b.ID))
			}
		case BlockReturn, BlockExit:
			if len(b.Succs) != 0 {
				errs = append(errs, fmt.Errorf("block b%d is %s but has successors", b.ID, b.Kind))
			}
		}

		// Check values
		for _, v := range b.Values {
			if v.Block != b {
				errs = append(errs, fmt.Errorf("value v%d has wrong block pointer", v.ID))
			}
		}
	}

	return errs
}

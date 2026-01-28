package ir

import "fmt"

// BlockKind represents the type of block terminator.
type BlockKind int

const (
	// BlockInvalid is the zero value for uninitialized blocks.
	BlockInvalid BlockKind = iota

	// BlockPlain has exactly one successor (unconditional jump).
	BlockPlain

	// BlockIf has exactly two successors (conditional branch).
	// Succs[0] is the "then" branch (condition true).
	// Succs[1] is the "else" branch (condition false).
	BlockIf

	// BlockReturn ends the function with a return.
	// No successors. The return value (if any) is in Values.
	BlockReturn

	// BlockExit ends the program.
	// No successors. The exit code is in Values.
	BlockExit
)

// String returns the name of the block kind.
func (k BlockKind) String() string {
	switch k {
	case BlockInvalid:
		return "Invalid"
	case BlockPlain:
		return "Plain"
	case BlockIf:
		return "If"
	case BlockReturn:
		return "Return"
	case BlockExit:
		return "Exit"
	default:
		return "Unknown"
	}
}

// Block represents a basic block in the control flow graph.
// A basic block is a sequence of instructions with:
//   - One entry point (no jumps into the middle)
//   - One exit point (the terminator at the end)
type Block struct {
	// ID is a unique identifier within the function (b0, b1, b2, ...).
	ID int

	// Kind specifies the type of terminator for this block.
	Kind BlockKind

	// Func is the function containing this block.
	Func *Function

	// Values are the SSA values in this block, in order.
	// Phi nodes should come first, followed by other instructions.
	Values []*Value

	// Control is the condition value for BlockIf.
	// Must be a boolean-typed value.
	Control *Value

	// Preds are the predecessor blocks in the control flow graph.
	Preds []*Block

	// Succs are the successor blocks in the control flow graph.
	// For BlockPlain: exactly 1 successor
	// For BlockIf: exactly 2 successors (then, else)
	// For BlockReturn/BlockExit: no successors
	Succs []*Block

	// Sealed indicates that all predecessors are known.
	// Used during SSA construction for phi node placement.
	Sealed bool
}

// String returns a human-readable representation of the block.
func (b *Block) String() string {
	if b == nil {
		return "<nil>"
	}
	return fmt.Sprintf("b%d", b.ID)
}

// LongString returns a detailed representation of the block.
func (b *Block) LongString() string {
	if b == nil {
		return "<nil>"
	}

	result := fmt.Sprintf("b%d (%s)", b.ID, b.Kind)

	if len(b.Preds) > 0 {
		result += " preds:["
		for i, pred := range b.Preds {
			if i > 0 {
				result += ", "
			}
			result += pred.String()
		}
		result += "]"
	}

	if len(b.Succs) > 0 {
		result += " succs:["
		for i, succ := range b.Succs {
			if i > 0 {
				result += ", "
			}
			result += succ.String()
		}
		result += "]"
	}

	return result
}

// NewValue creates a new value in this block.
func (b *Block) NewValue(op Op, typ Type, args ...*Value) *Value {
	v := &Value{
		ID:    b.Func.nextValueID(),
		Op:    op,
		Type:  typ,
		Block: b,
	}
	v.SetArgs(args...)
	b.Values = append(b.Values, v)
	return v
}

// NewPhiValue creates a new phi node and inserts it at the correct position
// (after existing phi nodes, before non-phi values).
func (b *Block) NewPhiValue(typ Type) *Value {
	v := &Value{
		ID:    b.Func.nextValueID(),
		Op:    OpPhi,
		Type:  typ,
		Block: b,
	}

	// Find the position after all existing phi nodes
	insertPos := 0
	for i, existing := range b.Values {
		if existing.Op == OpPhi {
			insertPos = i + 1
		} else {
			break
		}
	}

	// Insert at the correct position
	if insertPos == len(b.Values) {
		b.Values = append(b.Values, v)
	} else {
		b.Values = append(b.Values[:insertPos], append([]*Value{v}, b.Values[insertPos:]...)...)
	}

	return v
}

// NewValueBefore creates a new value and inserts it before the given value.
func (b *Block) NewValueBefore(before *Value, op Op, typ Type, args ...*Value) *Value {
	v := &Value{
		ID:    b.Func.nextValueID(),
		Op:    op,
		Type:  typ,
		Block: b,
	}
	v.SetArgs(args...)

	// Find position of 'before' and insert
	for i, existing := range b.Values {
		if existing == before {
			// Insert at position i
			b.Values = append(b.Values[:i], append([]*Value{v}, b.Values[i:]...)...)
			return v
		}
	}

	// 'before' not found, append at end
	b.Values = append(b.Values, v)
	return v
}

// AddSucc adds a successor block and updates both blocks' edges.
func (b *Block) AddSucc(succ *Block) {
	b.Succs = append(b.Succs, succ)
	succ.Preds = append(succ.Preds, b)
}

// SetSucc sets a specific successor (by index).
func (b *Block) SetSucc(index int, succ *Block) {
	// Grow Succs slice if needed
	for len(b.Succs) <= index {
		b.Succs = append(b.Succs, nil)
	}

	// Remove old pred edge if replacing
	if old := b.Succs[index]; old != nil {
		for i, pred := range old.Preds {
			if pred == b {
				old.Preds = append(old.Preds[:i], old.Preds[i+1:]...)
				break
			}
		}
	}

	b.Succs[index] = succ
	if succ != nil {
		succ.Preds = append(succ.Preds, b)
	}
}

// NumPreds returns the number of predecessor blocks.
func (b *Block) NumPreds() int {
	return len(b.Preds)
}

// NumSuccs returns the number of successor blocks.
func (b *Block) NumSuccs() int {
	return len(b.Succs)
}

// NumValues returns the number of values in the block.
func (b *Block) NumValues() int {
	return len(b.Values)
}

// FirstValue returns the first value in the block, or nil if empty.
func (b *Block) FirstValue() *Value {
	if len(b.Values) == 0 {
		return nil
	}
	return b.Values[0]
}

// LastValue returns the last value in the block, or nil if empty.
func (b *Block) LastValue() *Value {
	if len(b.Values) == 0 {
		return nil
	}
	return b.Values[len(b.Values)-1]
}

// IsEntry returns true if this is the function's entry block.
func (b *Block) IsEntry() bool {
	return b.Func != nil && len(b.Func.Blocks) > 0 && b.Func.Blocks[0] == b
}

// Phis returns all phi nodes in this block.
func (b *Block) Phis() []*Value {
	var phis []*Value
	for _, v := range b.Values {
		if v.Op == OpPhi {
			phis = append(phis, v)
		} else {
			// Phis must be at the beginning
			break
		}
	}
	return phis
}

// NonPhis returns all non-phi values in this block.
func (b *Block) NonPhis() []*Value {
	for i, v := range b.Values {
		if v.Op != OpPhi {
			return b.Values[i:]
		}
	}
	return nil
}

// RemoveValue removes a value from this block.
func (b *Block) RemoveValue(v *Value) {
	for i, val := range b.Values {
		if val == v {
			b.Values = append(b.Values[:i], b.Values[i+1:]...)
			v.Block = nil
			return
		}
	}
}

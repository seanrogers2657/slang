package ir

import "fmt"

// Value represents an SSA value - the result of exactly one operation.
// In SSA form, each Value is defined exactly once and can be used multiple times.
type Value struct {
	// ID is a unique identifier within the function (v0, v1, v2, ...).
	ID int

	// Op is the operation that produces this value.
	Op Op

	// Type is the result type of this value.
	Type Type

	// Args are the input values (SSA use-def edges).
	Args []*Value

	// Block is the basic block containing this value.
	Block *Block

	// PhiArgs are the incoming edges for phi nodes.
	// Each PhiArg specifies a value and the predecessor block it comes from.
	PhiArgs []*PhiArg

	// Auxiliary data for constants and other operations
	AuxInt    int64   // immediate integer value, field offset, size, etc.
	AuxFloat  float64 // immediate float value
	AuxString string  // string constant, function name, label, etc.

	// Uses tracks all values that use this value (back-edges).
	// This is populated during IR construction for optimization passes.
	Uses []*Value

	// Pos is the source position for error messages.
	Pos Position
}

// PhiArg represents one incoming edge to a phi node.
type PhiArg struct {
	// Value is the SSA value from this predecessor.
	Value *Value

	// From is the predecessor block this value comes from.
	From *Block
}

// Position represents a source code location.
type Position struct {
	File   string
	Line   int
	Column int
}

// String returns a human-readable representation of the value.
func (v *Value) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("v%d", v.ID)
}

// LongString returns a detailed representation including the operation.
func (v *Value) LongString() string {
	if v == nil {
		return "<nil>"
	}

	result := fmt.Sprintf("v%d = %s", v.ID, v.Op)

	// Add auxiliary data
	switch v.Op {
	case OpConst:
		if v.Type != nil {
			if _, ok := v.Type.(*StringType); ok {
				result += fmt.Sprintf(" %q", v.AuxString)
			} else if _, ok := v.Type.(*BoolType); ok {
				if v.AuxInt != 0 {
					result += " true"
				} else {
					result += " false"
				}
			} else {
				result += fmt.Sprintf(" %d", v.AuxInt)
			}
		} else {
			result += fmt.Sprintf(" %d", v.AuxInt)
		}
	case OpArg:
		result += fmt.Sprintf(" [%d]", v.AuxInt)
	case OpCall:
		result += fmt.Sprintf(" %s", v.AuxString)
	case OpFieldPtr:
		result += fmt.Sprintf(" +%d", v.AuxInt)
	case OpAlloc, OpFree:
		result += fmt.Sprintf(" size=%d", v.AuxInt)
	case OpPhi:
		result += " ["
		for i, arg := range v.PhiArgs {
			if i > 0 {
				result += ", "
			}
			result += fmt.Sprintf("v%d<-b%d", arg.Value.ID, arg.From.ID)
		}
		result += "]"
	}

	// Add arguments
	if len(v.Args) > 0 && v.Op != OpPhi {
		result += "("
		for i, arg := range v.Args {
			if i > 0 {
				result += ", "
			}
			result += arg.String()
		}
		result += ")"
	}

	// Add type
	if v.Type != nil {
		result += " [" + v.Type.String() + "]"
	}

	return result
}

// AddArg adds an argument to this value and updates the argument's Uses list.
func (v *Value) AddArg(arg *Value) {
	v.Args = append(v.Args, arg)
	arg.Uses = append(arg.Uses, v)
}

// SetArgs sets all arguments at once, updating Uses lists.
func (v *Value) SetArgs(args ...*Value) {
	v.Args = args
	for _, arg := range args {
		arg.Uses = append(arg.Uses, v)
	}
}

// RemoveArg removes an argument at the given index.
func (v *Value) RemoveArg(index int) {
	if index < 0 || index >= len(v.Args) {
		return
	}

	// Remove from the argument's Uses list
	arg := v.Args[index]
	for i, use := range arg.Uses {
		if use == v {
			arg.Uses = append(arg.Uses[:i], arg.Uses[i+1:]...)
			break
		}
	}

	// Remove from Args
	v.Args = append(v.Args[:index], v.Args[index+1:]...)
}

// ReplaceArg replaces the argument at the given index.
func (v *Value) ReplaceArg(index int, newArg *Value) {
	if index < 0 || index >= len(v.Args) {
		return
	}

	// Remove from old argument's Uses list
	oldArg := v.Args[index]
	for i, use := range oldArg.Uses {
		if use == v {
			oldArg.Uses = append(oldArg.Uses[:i], oldArg.Uses[i+1:]...)
			break
		}
	}

	// Set new argument and update its Uses
	v.Args[index] = newArg
	newArg.Uses = append(newArg.Uses, v)
}

// AddPhiArg adds an incoming edge to a phi node.
func (v *Value) AddPhiArg(val *Value, from *Block) {
	v.PhiArgs = append(v.PhiArgs, &PhiArg{
		Value: val,
		From:  from,
	})
	val.Uses = append(val.Uses, v)
}

// IsPhi returns true if this is a phi node.
func (v *Value) IsPhi() bool {
	return v.Op == OpPhi
}

// IsConst returns true if this is a constant value.
func (v *Value) IsConst() bool {
	return v.Op == OpConst
}

// IsConstInt returns true if this is an integer constant.
func (v *Value) IsConstInt() bool {
	if v.Op != OpConst {
		return false
	}
	_, ok := v.Type.(*IntType)
	return ok
}

// ConstValue returns the constant integer value, or 0 if not a constant.
func (v *Value) ConstValue() int64 {
	if v.Op == OpConst {
		return v.AuxInt
	}
	return 0
}

// HasSideEffects returns true if this value has side effects.
func (v *Value) HasSideEffects() bool {
	return v.Op.HasSideEffects()
}

// NumUses returns the number of values that use this value.
func (v *Value) NumUses() int {
	return len(v.Uses)
}

// IsDead returns true if this value has no uses and no side effects.
func (v *Value) IsDead() bool {
	return len(v.Uses) == 0 && !v.HasSideEffects()
}

// Func returns the function containing this value.
func (v *Value) Func() *Function {
	if v.Block == nil {
		return nil
	}
	return v.Block.Func
}

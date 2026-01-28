package ir

import (
	"fmt"
)

// ValidationError represents an error found during IR validation.
type ValidationError struct {
	Func    *Function
	Block   *Block
	Value   *Value
	Message string
}

func (e *ValidationError) Error() string {
	loc := ""
	if e.Func != nil {
		loc = e.Func.Name
	}
	if e.Block != nil {
		loc += fmt.Sprintf(" b%d", e.Block.ID)
	}
	if e.Value != nil {
		loc += fmt.Sprintf(" v%d", e.Value.ID)
	}
	if loc != "" {
		return fmt.Sprintf("%s: %s", loc, e.Message)
	}
	return e.Message
}

// Validator checks IR for well-formedness.
type Validator struct {
	errs    []*ValidationError
	prog    *Program
	fn      *Function
	defined map[*Value]bool // values that have been defined
}

// Validate checks a program for well-formedness and returns any errors.
func Validate(prog *Program) []*ValidationError {
	v := &Validator{
		prog:    prog,
		defined: make(map[*Value]bool),
	}
	v.validateProgram()
	return v.errs
}

// ValidateFunction checks a single function for well-formedness.
func ValidateFunction(fn *Function) []*ValidationError {
	v := &Validator{
		fn:      fn,
		defined: make(map[*Value]bool),
	}
	v.validateFunction(fn)
	return v.errs
}

func (v *Validator) error(fn *Function, b *Block, val *Value, msg string, args ...interface{}) {
	v.errs = append(v.errs, &ValidationError{
		Func:    fn,
		Block:   b,
		Value:   val,
		Message: fmt.Sprintf(msg, args...),
	})
}

func (v *Validator) validateProgram() {
	// Check for main function
	if v.prog.Main() == nil {
		v.error(nil, nil, nil, "no main function")
	}

	// Check for duplicate function names
	names := make(map[string]bool)
	for _, fn := range v.prog.Functions {
		if names[fn.Name] {
			v.error(fn, nil, nil, "duplicate function name")
		}
		names[fn.Name] = true
	}

	// Check for duplicate struct names
	structNames := make(map[string]bool)
	for _, s := range v.prog.Structs {
		if structNames[s.Name] {
			v.error(nil, nil, nil, "duplicate struct name: %s", s.Name)
		}
		structNames[s.Name] = true
	}

	// Validate each function
	for _, fn := range v.prog.Functions {
		v.validateFunction(fn)
	}
}

func (v *Validator) validateFunction(fn *Function) {
	v.fn = fn
	v.defined = make(map[*Value]bool)

	// Check function has blocks
	if len(fn.Blocks) == 0 {
		v.error(fn, nil, nil, "function has no blocks")
		return
	}

	// Check entry block has no predecessors
	entry := fn.Entry()
	if len(entry.Preds) > 0 {
		v.error(fn, entry, nil, "entry block has predecessors")
	}

	// Mark parameters as defined
	for _, param := range fn.Params {
		v.defined[param] = true
	}

	// Validate each block
	for _, b := range fn.Blocks {
		v.validateBlock(b)
	}

	// Check CFG consistency
	v.validateCFG(fn)
}

func (v *Validator) validateBlock(b *Block) {
	// Check block belongs to correct function
	if b.Func != v.fn {
		v.error(v.fn, b, nil, "block has wrong function pointer")
	}

	// Check phi nodes come first
	seenNonPhi := false
	for _, val := range b.Values {
		if val.Op == OpPhi {
			if seenNonPhi {
				v.error(v.fn, b, val, "phi node after non-phi value")
			}
		} else {
			seenNonPhi = true
		}

		v.validateValue(b, val)
	}

	// Check terminator based on block kind
	v.validateTerminator(b)
}

func (v *Validator) validateValue(b *Block, val *Value) {
	// Check value belongs to correct block
	if val.Block != b {
		v.error(v.fn, b, val, "value has wrong block pointer")
	}

	// Check value has a type (most values need one)
	if val.Type == nil && !v.canBeTypeless(val) {
		v.error(v.fn, b, val, "value has no type")
	}

	// Mark this value as defined
	v.defined[val] = true

	// Check operands are defined before use (SSA property)
	for i, arg := range val.Args {
		if !v.defined[arg] {
			// For phi nodes, operands come from predecessors which may be defined later
			if val.Op != OpPhi {
				v.error(v.fn, b, val, "argument %d (v%d) used before definition", i, arg.ID)
			}
		}
	}

	// Operation-specific validation
	switch val.Op {
	case OpPhi:
		v.validatePhi(b, val)
	case OpCall:
		v.validateCall(b, val)
	case OpFieldPtr:
		v.validateFieldPtr(b, val)
	case OpAlloc:
		v.validateAlloc(b, val)
	case OpLoad:
		v.validateLoad(b, val)
	case OpStore:
		v.validateStore(b, val)
	}

	// Check binary operations have correct number of operands
	if val.Op.IsBinary() && len(val.Args) != 2 {
		v.error(v.fn, b, val, "binary operation has %d args, expected 2", len(val.Args))
	}
}

func (v *Validator) validatePhi(b *Block, val *Value) {
	// Phi nodes must have one argument per predecessor
	if len(val.PhiArgs) != len(b.Preds) {
		v.error(v.fn, b, val, "phi has %d args but block has %d predecessors",
			len(val.PhiArgs), len(b.Preds))
	}

	// Check each phi argument comes from a predecessor
	predSet := make(map[*Block]bool)
	for _, pred := range b.Preds {
		predSet[pred] = true
	}

	for _, phiArg := range val.PhiArgs {
		if !predSet[phiArg.From] {
			v.error(v.fn, b, val, "phi arg from b%d which is not a predecessor", phiArg.From.ID)
		}
	}

	// Check all phi args have compatible types
	if val.Type != nil {
		for _, phiArg := range val.PhiArgs {
			if phiArg.Value == nil {
				v.error(v.fn, b, val, "phi arg from b%d has nil value", phiArg.From.ID)
				continue
			}
			if phiArg.Value.Type != nil && !phiArg.Value.Type.Equal(val.Type) {
				v.error(v.fn, b, val, "phi arg v%d has type %s, expected %s",
					phiArg.Value.ID, phiArg.Value.Type.String(), val.Type.String())
			}
		}
	}
}

func (v *Validator) validateCall(b *Block, val *Value) {
	if val.AuxString == "" {
		v.error(v.fn, b, val, "call has no function name")
	}
}

func (v *Validator) validateFieldPtr(b *Block, val *Value) {
	if len(val.Args) != 1 {
		v.error(v.fn, b, val, "FieldPtr requires exactly 1 argument")
		return
	}

	// Check argument is a pointer type
	arg := val.Args[0]
	if arg.Type != nil {
		if _, ok := arg.Type.(*PtrType); !ok {
			v.error(v.fn, b, val, "FieldPtr argument must be a pointer, got %s", arg.Type.String())
		}
	}
}

func (v *Validator) validateAlloc(b *Block, val *Value) {
	if val.AuxInt <= 0 {
		v.error(v.fn, b, val, "Alloc with invalid size: %d", val.AuxInt)
	}

	// Result should be a pointer type
	if val.Type != nil {
		if _, ok := val.Type.(*PtrType); !ok {
			v.error(v.fn, b, val, "Alloc result must be a pointer type, got %s", val.Type.String())
		}
	}
}

func (v *Validator) validateLoad(b *Block, val *Value) {
	if len(val.Args) != 1 {
		v.error(v.fn, b, val, "Load requires exactly 1 argument")
		return
	}

	// Check argument is a pointer type
	arg := val.Args[0]
	if arg.Type != nil {
		if _, ok := arg.Type.(*PtrType); !ok {
			v.error(v.fn, b, val, "Load argument must be a pointer, got %s", arg.Type.String())
		}
	}
}

func (v *Validator) validateStore(b *Block, val *Value) {
	if len(val.Args) != 2 {
		v.error(v.fn, b, val, "Store requires exactly 2 arguments")
		return
	}

	// First argument should be a pointer
	ptr := val.Args[0]
	if ptr.Type != nil {
		if _, ok := ptr.Type.(*PtrType); !ok {
			v.error(v.fn, b, val, "Store first argument must be a pointer, got %s", ptr.Type.String())
		}
	}
}

func (v *Validator) validateTerminator(b *Block) {
	switch b.Kind {
	case BlockPlain:
		if len(b.Succs) != 1 {
			v.error(v.fn, b, nil, "Plain block has %d successors, expected 1", len(b.Succs))
		}

	case BlockIf:
		if len(b.Succs) != 2 {
			v.error(v.fn, b, nil, "If block has %d successors, expected 2", len(b.Succs))
		}
		if b.Control == nil {
			v.error(v.fn, b, nil, "If block has no control value")
		} else if b.Control.Type != nil {
			if _, ok := b.Control.Type.(*BoolType); !ok {
				v.error(v.fn, b, nil, "If block control must be bool, got %s", b.Control.Type.String())
			}
		}

	case BlockReturn:
		if len(b.Succs) != 0 {
			v.error(v.fn, b, nil, "Return block has successors")
		}
		// Check for OpReturn value
		hasReturn := false
		for _, val := range b.Values {
			if val.Op == OpReturn {
				hasReturn = true
				break
			}
		}
		if !hasReturn {
			v.error(v.fn, b, nil, "Return block has no OpReturn value")
		}

	case BlockExit:
		if len(b.Succs) != 0 {
			v.error(v.fn, b, nil, "Exit block has successors")
		}

	case BlockInvalid:
		v.error(v.fn, b, nil, "block has invalid kind")
	}
}

func (v *Validator) validateCFG(fn *Function) {
	// Check predecessor/successor consistency
	for _, b := range fn.Blocks {
		for _, succ := range b.Succs {
			// Check succ has b as predecessor
			found := false
			for _, pred := range succ.Preds {
				if pred == b {
					found = true
					break
				}
			}
			if !found {
				v.error(fn, b, nil, "successor b%d does not have b%d as predecessor", succ.ID, b.ID)
			}
		}

		for _, pred := range b.Preds {
			// Check pred has b as successor
			found := false
			for _, succ := range pred.Succs {
				if succ == b {
					found = true
					break
				}
			}
			if !found {
				v.error(fn, b, nil, "predecessor b%d does not have b%d as successor", pred.ID, b.ID)
			}
		}
	}
}

func (v *Validator) canBeTypeless(val *Value) bool {
	// Some operations don't produce values
	switch val.Op {
	case OpStore, OpFree, OpReturn, OpExit, OpMemCopy:
		return true
	default:
		return false
	}
}

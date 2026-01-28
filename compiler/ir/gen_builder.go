package ir

// ValueBuilder provides a fluent API for creating IR values.
// It reduces boilerplate when creating common value patterns.
type ValueBuilder struct {
	block *Block
}

// builder returns a ValueBuilder for the current block.
func (g *Generator) builder() *ValueBuilder {
	return &ValueBuilder{block: g.block}
}

// =============================================================================
// Constants
// =============================================================================

// ConstInt creates an integer constant.
func (vb *ValueBuilder) ConstInt(typ Type, val int64) *Value {
	v := vb.block.NewValue(OpConst, typ)
	v.AuxInt = val
	return v
}

// ConstBool creates a boolean constant.
func (vb *ValueBuilder) ConstBool(val bool) *Value {
	v := vb.block.NewValue(OpConst, TypeBool)
	if val {
		v.AuxInt = 1
	} else {
		v.AuxInt = 0
	}
	return v
}

// ConstString creates a string constant (requires program for string table).
func (vb *ValueBuilder) ConstString(prog *Program, s string) *Value {
	idx := prog.AddString(s)
	v := vb.block.NewValue(OpConst, TypeString)
	v.AuxInt = int64(idx)
	v.AuxString = s
	return v
}

// =============================================================================
// Memory Operations
// =============================================================================

// Alloc allocates memory of the given size and returns a pointer.
func (vb *ValueBuilder) Alloc(elemType Type, size int64) *Value {
	v := vb.block.NewValue(OpAlloc, &PtrType{Elem: elemType})
	v.AuxInt = size
	return v
}

// Load loads a value from a pointer.
func (vb *ValueBuilder) Load(ptr *Value, resultType Type) *Value {
	v := vb.block.NewValue(OpLoad, resultType)
	v.AddArg(ptr)
	return v
}

// Store stores a value to a pointer. Returns the store value (for chaining).
func (vb *ValueBuilder) Store(ptr, val *Value) *Value {
	v := vb.block.NewValue(OpStore, nil)
	v.AddArg(ptr)
	v.AddArg(val)
	return v
}

// FieldPtr creates a pointer to a struct field.
func (vb *ValueBuilder) FieldPtr(obj *Value, elemType Type, offset int64) *Value {
	v := vb.block.NewValue(OpFieldPtr, &PtrType{Elem: elemType})
	v.AddArg(obj)
	v.AuxInt = offset
	return v
}

// IndexPtr creates a pointer to an array element.
func (vb *ValueBuilder) IndexPtr(arr, idx *Value, elemType Type) *Value {
	v := vb.block.NewValue(OpIndexPtr, &PtrType{Elem: elemType})
	v.AddArg(arr)
	v.AddArg(idx)
	return v
}

// MemCopy copies memory from src to dest.
func (vb *ValueBuilder) MemCopy(dest, src *Value, size int64) *Value {
	v := vb.block.NewValue(OpMemCopy, nil)
	v.AddArg(dest)
	v.AddArg(src)
	v.AuxInt = size
	return v
}

// =============================================================================
// Arithmetic Operations
// =============================================================================

// Binary creates a binary operation.
func (vb *ValueBuilder) Binary(op Op, typ Type, left, right *Value) *Value {
	v := vb.block.NewValue(op, typ)
	v.AddArg(left)
	v.AddArg(right)
	return v
}

// Add creates an addition.
func (vb *ValueBuilder) Add(typ Type, left, right *Value) *Value {
	return vb.Binary(OpAdd, typ, left, right)
}

// Sub creates a subtraction.
func (vb *ValueBuilder) Sub(typ Type, left, right *Value) *Value {
	return vb.Binary(OpSub, typ, left, right)
}

// Mul creates a multiplication.
func (vb *ValueBuilder) Mul(typ Type, left, right *Value) *Value {
	return vb.Binary(OpMul, typ, left, right)
}

// Div creates a division.
func (vb *ValueBuilder) Div(typ Type, left, right *Value) *Value {
	return vb.Binary(OpDiv, typ, left, right)
}

// Mod creates a modulo.
func (vb *ValueBuilder) Mod(typ Type, left, right *Value) *Value {
	return vb.Binary(OpMod, typ, left, right)
}

// =============================================================================
// Comparison Operations
// =============================================================================

// Eq creates an equality comparison.
func (vb *ValueBuilder) Eq(left, right *Value) *Value {
	return vb.Binary(OpEq, TypeBool, left, right)
}

// Ne creates a not-equal comparison.
func (vb *ValueBuilder) Ne(left, right *Value) *Value {
	return vb.Binary(OpNe, TypeBool, left, right)
}

// Lt creates a less-than comparison.
func (vb *ValueBuilder) Lt(left, right *Value) *Value {
	return vb.Binary(OpLt, TypeBool, left, right)
}

// Le creates a less-than-or-equal comparison.
func (vb *ValueBuilder) Le(left, right *Value) *Value {
	return vb.Binary(OpLe, TypeBool, left, right)
}

// Gt creates a greater-than comparison.
func (vb *ValueBuilder) Gt(left, right *Value) *Value {
	return vb.Binary(OpGt, TypeBool, left, right)
}

// Ge creates a greater-than-or-equal comparison.
func (vb *ValueBuilder) Ge(left, right *Value) *Value {
	return vb.Binary(OpGe, TypeBool, left, right)
}

// =============================================================================
// Unary Operations
// =============================================================================

// Not creates a logical not.
func (vb *ValueBuilder) Not(operand *Value) *Value {
	v := vb.block.NewValue(OpNot, TypeBool)
	v.AddArg(operand)
	return v
}

// Neg creates a numeric negation.
func (vb *ValueBuilder) Neg(typ Type, operand *Value) *Value {
	v := vb.block.NewValue(OpNeg, typ)
	v.AddArg(operand)
	return v
}

// =============================================================================
// Nullable Operations
// =============================================================================

// IsNull checks if a nullable value is null.
func (vb *ValueBuilder) IsNull(val *Value) *Value {
	v := vb.block.NewValue(OpIsNull, TypeBool)
	v.AddArg(val)
	return v
}

// Unwrap extracts the value from a nullable.
func (vb *ValueBuilder) Unwrap(val *Value, resultType Type) *Value {
	v := vb.block.NewValue(OpUnwrap, resultType)
	v.AddArg(val)
	return v
}

// Wrap wraps a value as nullable.
func (vb *ValueBuilder) Wrap(val *Value, nullableType Type) *Value {
	v := vb.block.NewValue(OpWrap, nullableType)
	v.AddArg(val)
	return v
}

// WrapNull creates a null value of the given nullable type.
func (vb *ValueBuilder) WrapNull(nullableType Type) *Value {
	return vb.block.NewValue(OpWrapNull, nullableType)
}

// =============================================================================
// Control Flow
// =============================================================================

// Call creates a function call.
func (vb *ValueBuilder) Call(name string, resultType Type, args ...*Value) *Value {
	v := vb.block.NewValue(OpCall, resultType)
	v.AuxString = name
	for _, arg := range args {
		v.AddArg(arg)
	}
	return v
}

// Return creates a return value.
func (vb *ValueBuilder) Return(val *Value) *Value {
	v := vb.block.NewValue(OpReturn, nil)
	if val != nil {
		v.AddArg(val)
	}
	return v
}

// Phi creates a phi node (args added separately via PhiArgs).
func (vb *ValueBuilder) Phi(typ Type) *Value {
	return vb.block.NewPhiValue(typ)
}

// =============================================================================
// Copy Operations
// =============================================================================

// Copy creates a deep copy of a pointer value.
func (vb *ValueBuilder) Copy(ptr *Value) *Value {
	v := vb.block.NewValue(OpCopy, ptr.Type)
	v.AddArg(ptr)
	return v
}

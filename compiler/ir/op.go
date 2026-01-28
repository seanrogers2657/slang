package ir

// Op represents an IR operation code.
type Op int

const (
	// OpInvalid is the zero value and represents an invalid operation.
	OpInvalid Op = iota

	// Constants
	OpConst // constant value (uses AuxInt, AuxFloat, or AuxString)
	OpArg   // function argument (uses AuxInt for arg index)

	// Arithmetic operations (binary: Args[0], Args[1] -> result)
	OpAdd // addition
	OpSub // subtraction
	OpMul // multiplication
	OpDiv // division
	OpMod // modulo

	// Unary arithmetic
	OpNeg // negation (-x)

	// Comparison operations (produce bool)
	OpEq // equal
	OpNe // not equal
	OpLt // less than
	OpLe // less than or equal
	OpGt // greater than
	OpGe // greater than or equal

	// Logical operations (bool operands, bool result)
	OpAnd // logical and
	OpOr  // logical or
	OpNot // logical not (unary)

	// Memory operations
	OpAlloc    // allocate memory (AuxInt = size), returns pointer
	OpFree     // free memory (Args[0] = ptr, AuxInt = size)
	OpLoad     // load from pointer (Args[0] = ptr)
	OpStore    // store to pointer (Args[0] = ptr, Args[1] = value), no result
	OpCopy     // deep copy (Args[0] = ptr), returns new pointer
	OpMemCopy  // copy memory (Args[0] = dest ptr, Args[1] = src ptr, AuxInt = size), no result
	OpFieldPtr // get field pointer (Args[0] = struct ptr, AuxInt = field offset)
	OpIndexPtr // get array element pointer (Args[0] = array ptr, Args[1] = index)
	OpArrayLen // get array length (Args[0] = array)

	// Nullable operations
	OpIsNull   // check if nullable is null (Args[0] = nullable) -> bool
	OpUnwrap   // extract value from nullable (Args[0] = nullable), assumes not null
	OpWrap     // create nullable with value (Args[0] = value)
	OpWrapNull // create null value (no args, Type = nullable type)

	// Control flow (these are block terminators)
	OpPhi    // SSA phi node (uses PhiArgs)
	OpCall   // function call (Args = arguments, AuxString = func name)
	OpReturn // return from function (Args[0] = return value, or empty for void)
	OpExit   // exit program (Args[0] = exit code)

	// Type conversions
	OpZeroExt   // zero extend (Args[0] = value, Type = target type)
	OpSignExt   // sign extend
	OpTrunc     // truncate to smaller type
	OpIntToPtr  // integer to pointer
	OpPtrToInt  // pointer to integer
	OpBitcast   // reinterpret bits as different type

	// Note: Jump and Branch are represented by Block.Kind, not as Values
)

// String returns the name of the operation.
func (op Op) String() string {
	switch op {
	case OpInvalid:
		return "Invalid"
	case OpConst:
		return "Const"
	case OpArg:
		return "Arg"
	case OpAdd:
		return "Add"
	case OpSub:
		return "Sub"
	case OpMul:
		return "Mul"
	case OpDiv:
		return "Div"
	case OpMod:
		return "Mod"
	case OpNeg:
		return "Neg"
	case OpEq:
		return "Eq"
	case OpNe:
		return "Ne"
	case OpLt:
		return "Lt"
	case OpLe:
		return "Le"
	case OpGt:
		return "Gt"
	case OpGe:
		return "Ge"
	case OpAnd:
		return "And"
	case OpOr:
		return "Or"
	case OpNot:
		return "Not"
	case OpAlloc:
		return "Alloc"
	case OpFree:
		return "Free"
	case OpLoad:
		return "Load"
	case OpStore:
		return "Store"
	case OpCopy:
		return "Copy"
	case OpMemCopy:
		return "MemCopy"
	case OpFieldPtr:
		return "FieldPtr"
	case OpIndexPtr:
		return "IndexPtr"
	case OpArrayLen:
		return "ArrayLen"
	case OpIsNull:
		return "IsNull"
	case OpUnwrap:
		return "Unwrap"
	case OpWrap:
		return "Wrap"
	case OpWrapNull:
		return "WrapNull"
	case OpPhi:
		return "Phi"
	case OpCall:
		return "Call"
	case OpReturn:
		return "Return"
	case OpExit:
		return "Exit"
	case OpZeroExt:
		return "ZeroExt"
	case OpSignExt:
		return "SignExt"
	case OpTrunc:
		return "Trunc"
	case OpIntToPtr:
		return "IntToPtr"
	case OpPtrToInt:
		return "PtrToInt"
	case OpBitcast:
		return "Bitcast"
	default:
		return "Unknown"
	}
}

// IsTerminator returns true if this operation is a block terminator.
func (op Op) IsTerminator() bool {
	switch op {
	case OpReturn, OpExit:
		return true
	default:
		return false
	}
}

// HasSideEffects returns true if this operation has side effects
// and cannot be eliminated even if its result is unused.
func (op Op) HasSideEffects() bool {
	switch op {
	case OpStore, OpFree, OpCall, OpReturn, OpExit:
		return true
	default:
		return false
	}
}

// IsBinary returns true if this is a binary operation (two operands).
func (op Op) IsBinary() bool {
	switch op {
	case OpAdd, OpSub, OpMul, OpDiv, OpMod,
		OpEq, OpNe, OpLt, OpLe, OpGt, OpGe,
		OpAnd, OpOr:
		return true
	default:
		return false
	}
}

// IsComparison returns true if this operation produces a boolean result.
func (op Op) IsComparison() bool {
	switch op {
	case OpEq, OpNe, OpLt, OpLe, OpGt, OpGe:
		return true
	default:
		return false
	}
}

// IsCommutative returns true if operand order doesn't matter.
func (op Op) IsCommutative() bool {
	switch op {
	case OpAdd, OpMul, OpEq, OpNe, OpAnd, OpOr:
		return true
	default:
		return false
	}
}

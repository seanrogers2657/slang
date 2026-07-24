package semantic

import (
	"github.com/seanrogers2657/slang/compiler/ast"
)

// IsCopyable returns true if a type can be implicitly copied. Primitives and
// references are copyable; owned pointers (*T) and aggregates containing them
// are not (they are single-owner under the scope-frees-it model).
func IsCopyable(t Type) bool {
	switch tt := t.(type) {
	// Primitive types are always copyable
	case S8Type, S16Type, S32Type, S64Type, S128Type,
		U8Type, U16Type, U32Type, U64Type, U128Type,
		F32Type, F64Type,
		BooleanType, StringType, VecType:
		return true

	// Void and error types are trivially copyable
	case VoidType, ErrorType, NothingType:
		return true

	// References are copyable (they're just borrows)
	case RefPointerType:
		return true

	// Owned pointers are NOT copyable (single-owner, scope-freed)
	case OwnedPointerType:
		return false

	// Nullable types: copyable if inner type is copyable
	case NullableType:
		return IsCopyable(tt.InnerType)

	// Structs: copyable if all fields are copyable
	case StructType:
		for _, field := range tt.Fields {
			if !IsCopyable(field.Type) {
				return false
			}
		}
		return true

	// Arrays: copyable if element type is copyable
	case ArrayType:
		return IsCopyable(tt.ElementType)

	// Functions are not copyable (they're not values in Slang)
	case FunctionType:
		return false

	default:
		// Unknown types default to not copyable for safety
		return false
	}
}

// IsNonCopyable returns true if a type cannot be implicitly copied. Under the
// scope-frees-it model these values — owned pointers (*T), aggregates that
// contain one, and classes — are single-owner: they may not be aliased to a
// second binding (no implicit copy, and no moves either).
func IsNonCopyable(t Type) bool {
	return !IsCopyable(t)
}

// ownsHeap reports whether a type is an owned heap pointer (*T or *T?). These are
// the only heap-owning, single-owner category permitted by the scope-frees-it
// ownership model, and they are restricted to local bindings (never returns,
// fields, or params).
func ownsHeap(t Type) bool {
	return IsOwnedPointer(t) || IsNullableOwnedPointer(t)
}

// isValueType reports whether a type has pure value semantics: it is not a borrow
// (&T/&&T) and owns no heap that a second binding could alias. Value types are exactly what
// may be returned from a function, stored in a field, and freely copied.
//
// This is the allow-list at the heart of the scope-frees-it ownership model: we
// define what a value type *is* and restrict the non-value categories to specific
// positions (borrows: parameters only; owned pointers: local bindings only).
// Anything not enumerated here is, by default, not a value type.
func isValueType(t Type) bool {
	switch tt := t.(type) {
	// Primitives and strings are values. (string is heap-backed but copyable with
	// value semantics — see the string memory model.)
	case S8Type, S16Type, S32Type, S64Type, S128Type,
		U8Type, U16Type, U32Type, U64Type, U128Type,
		F32Type, F64Type,
		BooleanType, StringType, VecType:
		return true

	// Void/error/nothing are trivially value-like (and harmless in these positions).
	case VoidType, ErrorType, NothingType:
		return true

	// Nullable is a value type iff its payload is.
	case NullableType:
		return isValueType(tt.InnerType)

	// Aggregates are values iff every component is a value (no owned-pointer fields,
	// no nested borrows).
	case StructType:
		for _, field := range tt.Fields {
			if !isValueType(field.Type) {
				return false
			}
		}
		return true
	case ClassType:
		for _, field := range tt.Fields {
			if !isValueType(field.Type) {
				return false
			}
		}
		return true
	case ArrayType:
		return isValueType(tt.ElementType)

	default:
		// Borrows (&T/&&T), owned pointers (*T/*T?), functions, namespaces, etc.
		// are not value types.
		return false
	}
}

// ContainsOwnedPointer returns true if a type contains any Own<T> fields (recursively)
func ContainsOwnedPointer(t Type) bool {
	switch tt := t.(type) {
	case OwnedPointerType:
		return true
	case NullableType:
		return ContainsOwnedPointer(tt.InnerType)
	case StructType:
		for _, field := range tt.Fields {
			if ContainsOwnedPointer(field.Type) {
				return true
			}
		}
		return false
	case ArrayType:
		return ContainsOwnedPointer(tt.ElementType)
	default:
		return false
	}
}

// BorrowInfo tracks a borrow of a variable during a function call
type BorrowInfo struct {
	VarName  string       // root variable being borrowed
	Mutable  bool         // whether this is a mutable borrow
	Position ast.Position // position of the borrow (for error messages)
	ArgIndex int          // which argument position
}

// GetRootVarName extracts the root variable name from an expression.
// For example: p -> "p", p.x -> "p", p.x.y -> "p"
// Returns empty string if the expression doesn't have a clear root variable.
func GetRootVarName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		return e.Name
	case *ast.SelfExpr:
		// 'self' is bound in the method scope under the name "self".
		return "self"
	case *ast.FieldAccessExpr:
		return GetRootVarName(e.Object)
	case *ast.IndexExpr:
		return GetRootVarName(e.Array)
	default:
		return ""
	}
}

// CheckBorrowConflicts checks for conflicting borrows and returns an error message if found.
// Rules:
// - Multiple immutable borrows of the same variable: OK
// - Multiple mutable borrows of the same variable: ERROR
// - Mixed mutable + immutable borrows of the same variable: ERROR
func CheckBorrowConflicts(borrows []BorrowInfo) (hasConflict bool, conflictMsg string, conflictPos1, conflictPos2 ast.Position) {
	// Group borrows by variable name
	byVar := make(map[string][]BorrowInfo)
	for _, b := range borrows {
		if b.VarName != "" {
			byVar[b.VarName] = append(byVar[b.VarName], b)
		}
	}

	// Check each variable for conflicts
	for varName, varBorrows := range byVar {
		if len(varBorrows) < 2 {
			continue
		}

		// Check for any mutable borrow
		var hasMutable bool
		var mutableBorrow BorrowInfo
		for _, b := range varBorrows {
			if b.Mutable {
				hasMutable = true
				mutableBorrow = b
				break
			}
		}

		if hasMutable {
			// Find another borrow that conflicts
			for _, b := range varBorrows {
				if b.ArgIndex != mutableBorrow.ArgIndex {
					if b.Mutable {
						// Two mutable borrows
						return true,
							"cannot borrow '" + varName + "' as mutable more than once",
							mutableBorrow.Position, b.Position
					}
					// Mutable + immutable
					return true,
						"cannot borrow '" + varName + "' as both mutable and immutable",
						mutableBorrow.Position, b.Position
				}
			}
		}
	}

	return false, "", ast.Position{}, ast.Position{}
}

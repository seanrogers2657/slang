package semantic

import (
	"github.com/seanrogers2657/slang/compiler/ast"
)

// OwnershipState represents the current ownership state of a variable
type OwnershipState int

const (
	// StateOwned indicates the variable currently owns its value
	StateOwned OwnershipState = iota
	// StateMoved indicates the variable's value has been moved elsewhere
	StateMoved
	// StateBorrowed indicates the variable is currently borrowed (not yet used)
	StateBorrowed
)

func (s OwnershipState) String() string {
	switch s {
	case StateOwned:
		return "owned"
	case StateMoved:
		return "moved"
	case StateBorrowed:
		return "borrowed"
	default:
		return "unknown"
	}
}

// MoveInfo records where and how a variable was moved
type MoveInfo struct {
	MovedTo  string       // name of variable it was moved to (or "<param>" for function param)
	Location ast.Position // position where the move occurred
}

// OwnershipInfo tracks the ownership state of a single variable
type OwnershipInfo struct {
	State    OwnershipState
	Type     Type     // the variable's type
	MoveInfo MoveInfo // info about where it was moved (if State == StateMoved)
}

// OwnershipScope tracks ownership within a lexical scope
type OwnershipScope struct {
	parent    *OwnershipScope
	ownership map[string]OwnershipInfo
	inLoop    bool // true if this scope is inside a loop body
}

// newOwnershipScope creates a new ownership scope
func newOwnershipScope(parent *OwnershipScope) *OwnershipScope {
	return &OwnershipScope{
		parent:    parent,
		ownership: make(map[string]OwnershipInfo),
	}
}

// declare adds a new variable to ownership tracking
func (s *OwnershipScope) declare(name string, typ Type) {
	s.ownership[name] = OwnershipInfo{
		State: StateOwned,
		Type:  typ,
	}
}

// lookup finds ownership info for a variable in this scope or parent scopes
func (s *OwnershipScope) lookup(name string) (OwnershipInfo, bool) {
	if info, exists := s.ownership[name]; exists {
		return info, true
	}
	if s.parent != nil {
		return s.parent.lookup(name)
	}
	return OwnershipInfo{}, false
}

// markMoved marks a variable as moved
func (s *OwnershipScope) markMoved(name string, movedTo string, location ast.Position) bool {
	// First check this scope
	if info, exists := s.ownership[name]; exists {
		info.State = StateMoved
		info.MoveInfo = MoveInfo{MovedTo: movedTo, Location: location}
		s.ownership[name] = info
		return true
	}
	// Check parent scopes
	if s.parent != nil {
		return s.parent.markMoved(name, movedTo, location)
	}
	return false
}

// restoreOwned marks a previously moved variable as owned again
// This is used when a moved variable is reassigned a new value
func (s *OwnershipScope) restoreOwned(name string) bool {
	// First check this scope
	if info, exists := s.ownership[name]; exists {
		info.State = StateOwned
		info.MoveInfo = MoveInfo{} // clear move info
		s.ownership[name] = info
		return true
	}
	// Check parent scopes
	if s.parent != nil {
		return s.parent.restoreOwned(name)
	}
	return false
}

// isInLoop returns true if this scope or any parent is inside a loop
func (s *OwnershipScope) isInLoop() bool {
	if s.inLoop {
		return true
	}
	if s.parent != nil {
		return s.parent.isInLoop()
	}
	return false
}

// snapshotParentState captures the current state of variables from parent scopes
// Returns a map of variable name -> ownership info for all variables in parent scopes
func (s *OwnershipScope) snapshotParentState() map[string]OwnershipInfo {
	snapshot := make(map[string]OwnershipInfo)
	for scope := s.parent; scope != nil; scope = scope.parent {
		for name, info := range scope.ownership {
			// Don't overwrite if we already have this variable (inner scope wins)
			if _, exists := snapshot[name]; !exists {
				snapshot[name] = info
			}
		}
	}
	return snapshot
}

// getMovedVars returns a list of variable names that were moved in this scope
// (either in this scope's map or promoted to parent)
func (s *OwnershipScope) getMovedVars(beforeSnapshot map[string]OwnershipInfo) []string {
	var moved []string

	// Check parent scopes for variables that changed state
	for scope := s; scope != nil; scope = scope.parent {
		for name, info := range scope.ownership {
			if info.State == StateMoved {
				// Check if it was owned before
				if before, existed := beforeSnapshot[name]; existed && before.State == StateOwned {
					moved = append(moved, name)
				}
			}
		}
	}

	return moved
}

// getMoveInfo gets the move info for a moved variable
func (s *OwnershipScope) getMoveInfo(name string) (MoveInfo, bool) {
	info, found := s.lookup(name)
	if found && info.State == StateMoved {
		return info.MoveInfo, true
	}
	return MoveInfo{}, false
}

// IsCopyable returns true if a type can be implicitly copied (not moved).
// Primitives and references are copyable; Own<T> and structs containing Own<T> are move-only.
func IsCopyable(t Type) bool {
	switch tt := t.(type) {
	// Primitive types are always copyable
	case S8Type, S16Type, S32Type, S64Type, S128Type,
		U8Type, U16Type, U32Type, U64Type, U128Type,
		F32Type, F64Type,
		BooleanType, StringType:
		return true

	// Void and error types are trivially copyable
	case VoidType, ErrorType, NothingType:
		return true

	// References are copyable (they're just borrows)
	case RefPointerType:
		return true

	// Owned pointers are NOT copyable (they're move-only)
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

// IsMoveOnly returns true if a type must be moved (cannot be implicitly copied)
func IsMoveOnly(t Type) bool {
	return !IsCopyable(t)
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

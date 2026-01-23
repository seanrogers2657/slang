package semantic

import (
	"fmt"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/errors"
)

// ============================================================================
// Type-related errors
// ============================================================================

// errTypeMismatch reports a type mismatch error
func (a *Analyzer) errTypeMismatch(expected, got Type, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("type mismatch: expected '%s', got '%s'", expected.String(), got.String()),
		pos, pos,
	)
}

// errCannotAssign reports an assignment type mismatch error
func (a *Analyzer) errCannotAssign(target, source Type, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot assign '%s' to '%s'", source.String(), target.String()),
		pos, pos,
	)
}

// errUnknownType reports an undefined type error
func (a *Analyzer) errUnknownType(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("undefined type '%s'", name),
		pos, pos,
	)
}

// errRefAsFieldType reports that references cannot be used as field types
func (a *Analyzer) errRefAsFieldType(pos ast.Position) *errors.CompilerError {
	return a.addError("&T cannot be used as a field type; use *T instead", pos, pos)
}

// errRefAsReturnType reports that references cannot be used as return types
func (a *Analyzer) errRefAsReturnType(pos ast.Position) *errors.CompilerError {
	return a.addError("references cannot be used as return types; use *T instead", pos, pos)
}

// errRefAsLocalVar reports that references cannot be stored in local variables
func (a *Analyzer) errRefAsLocalVar(pos ast.Position) *errors.CompilerError {
	return a.addError(
		"&T cannot be stored in local variables; references can only be function parameters",
		pos, pos,
	)
}

// ============================================================================
// Variable errors
// ============================================================================

// errUndefinedVar reports an undefined variable error
func (a *Analyzer) errUndefinedVar(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("undefined variable '%s'", name),
		pos, pos,
	)
}

// errDuplicateVar reports a duplicate variable declaration error
func (a *Analyzer) errDuplicateVar(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("variable '%s' is already declared in this scope", name),
		pos, pos,
	)
}

// errImmutableAssign reports an attempt to assign to an immutable variable
func (a *Analyzer) errImmutableAssign(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot assign to immutable variable '%s'", name),
		pos, pos,
	)
}

// errDuplicateParam reports a duplicate parameter declaration error
func (a *Analyzer) errDuplicateParam(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("parameter '%s' is already declared", name),
		pos, pos,
	)
}

// ============================================================================
// Declaration errors
// ============================================================================

// errDuplicateFunction reports a duplicate function declaration error
func (a *Analyzer) errDuplicateFunction(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("function '%s' is already declared", name),
		pos, pos,
	)
}

// errDuplicateStruct reports a duplicate struct declaration error
func (a *Analyzer) errDuplicateStruct(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("struct '%s' is already declared", name),
		pos, pos,
	)
}

// errDuplicateClass reports a duplicate class declaration error
func (a *Analyzer) errDuplicateClass(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("class '%s' is already declared", name),
		pos, pos,
	)
}

// errDuplicateObject reports a duplicate object declaration error
func (a *Analyzer) errDuplicateObject(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("object '%s' is already declared", name),
		pos, pos,
	)
}

// errTypeNameConflict reports that a type name conflicts with an existing type
func (a *Analyzer) errTypeNameConflict(name, existingKind string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("type '%s' is already declared as %s", name, existingKind),
		pos, pos,
	)
}

// ============================================================================
// Ownership errors
// ============================================================================

// errUsedAfterMove reports use of a value after it has been moved
func (a *Analyzer) errUsedAfterMove(name, movedTo string, usePos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("use of moved value '%s'", name),
		usePos, usePos,
	).WithHint(fmt.Sprintf("value was moved to '%s'", movedTo))
}

// errCannotMoveInLoop reports that a move cannot occur inside a loop
func (a *Analyzer) errCannotMoveInLoop(name string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot move '%s' inside a loop", name),
		pos, pos,
	).WithHint("use .copy() to create a copy, or move the value before the loop")
}

// ============================================================================
// Method errors
// ============================================================================

// errUndefinedMethod reports an undefined method error
func (a *Analyzer) errUndefinedMethod(method, typeName string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("type '%s' has no method '%s'", typeName, method),
		pos, pos,
	)
}

// errAmbiguousOverload reports an ambiguous method overload error
func (a *Analyzer) errAmbiguousOverload(method, typeName string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("ambiguous call to overloaded method '%s' on type '%s'", method, typeName),
		pos, pos,
	)
}

// errNoMatchingOverload reports no matching overload for a method call
func (a *Analyzer) errNoMatchingOverload(method, typeName string, argTypes []Type, pos ast.Position) *errors.CompilerError {
	args := ""
	for i, t := range argTypes {
		if i > 0 {
			args += ", "
		}
		args += t.String()
	}
	return a.addError(
		fmt.Sprintf("no matching overload for '%s.%s(%s)'", typeName, method, args),
		pos, pos,
	)
}

// errDuplicateMethodSignature reports a duplicate method signature error
func (a *Analyzer) errDuplicateMethodSignature(method string, paramTypes []Type, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("duplicate method signature: '%s' already has an overload with parameters (%s)",
			method, formatParamTypes(paramTypes)),
		pos, pos,
	)
}

// ============================================================================
// Null safety errors
// ============================================================================

// errNullToNonNullable reports assigning null to a non-nullable type
func (a *Analyzer) errNullToNonNullable(targetType Type, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot assign null to non-nullable type '%s'", targetType.String()),
		pos, pos,
	)
}

// errNullableToNonNullable reports assigning a nullable type to a non-nullable type
func (a *Analyzer) errNullableToNonNullable(sourceType, targetType Type, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot assign nullable type '%s' to non-nullable type '%s'",
			sourceType.String(), targetType.String()),
		pos, pos,
	)
}

// ============================================================================
// Self/method context errors
// ============================================================================

// errSelfOutsideMethod reports using 'self' outside of a method
func (a *Analyzer) errSelfOutsideMethod(pos ast.Position) *errors.CompilerError {
	return a.addError("'self' can only be used within a method body", pos, pos)
}

// errSelfNotAvailable reports that 'self' is not available in the current context
func (a *Analyzer) errSelfNotAvailable(pos ast.Position) *errors.CompilerError {
	return a.addError("'self' is not available in this context", pos, pos)
}

// errSelfMustBePointer reports that 'self' must have a pointer type
func (a *Analyzer) errSelfMustBePointer(className string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("'self' must have a pointer type (&%s, &&%s, or *%s)", className, className, className),
		pos, pos,
	)
}

// errSelfWrongClass reports that 'self' references the wrong class
func (a *Analyzer) errSelfWrongClass(expected, actual string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("'self' type must reference the enclosing class '%s', not '%s'", expected, actual),
		pos, pos,
	)
}

// ============================================================================
// Control flow errors
// ============================================================================

// errBreakOutsideLoop reports a break statement outside of a loop
func (a *Analyzer) errBreakOutsideLoop(pos ast.Position) *errors.CompilerError {
	return a.addError("'break' statement not inside a loop", pos, pos).
		WithHint("break can only be used inside while or for loops")
}

// errContinueOutsideLoop reports a continue statement outside of a loop
func (a *Analyzer) errContinueOutsideLoop(pos ast.Position) *errors.CompilerError {
	return a.addError("'continue' statement not inside a loop", pos, pos).
		WithHint("continue can only be used inside while or for loops")
}

// errReturnOutsideFunction reports a return statement outside of a function
func (a *Analyzer) errReturnOutsideFunction(pos ast.Position) *errors.CompilerError {
	return a.addError("return statement outside of function", pos, pos)
}

// errVoidFunctionReturnsValue reports that a void function returned a value
func (a *Analyzer) errVoidFunctionReturnsValue(pos ast.Position) *errors.CompilerError {
	return a.addError("void function should not return a value", pos, pos)
}

// ============================================================================
// Struct/class literal errors
// ============================================================================

// errAnonStructNeedsType reports that an anonymous struct literal needs a type annotation
func (a *Analyzer) errAnonStructNeedsType(startPos, endPos ast.Position) *errors.CompilerError {
	return a.addError(
		"anonymous struct literal requires type annotation (e.g., val p: Point = { ... })",
		startPos, endPos,
	)
}

// errCannotInferFromNull reports that type cannot be inferred from null
func (a *Analyzer) errCannotInferFromNull(pos ast.Position) *errors.CompilerError {
	return a.addError("cannot infer type from null, add type annotation", pos, pos)
}

// errFieldCountMismatch reports a field count mismatch in struct/class literals
func (a *Analyzer) errFieldCountMismatch(typeName string, expected, got int, startPos, endPos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("type '%s' has %d field(s), but %d argument(s) were provided",
			typeName, expected, got),
		startPos, endPos,
	)
}

// errDuplicateField reports a duplicate field in a struct literal
func (a *Analyzer) errDuplicateField(fieldName, typeName string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("duplicate field '%s' in %s literal", fieldName, typeName),
		pos, pos,
	)
}

// errUndefinedField reports an undefined field access
func (a *Analyzer) errUndefinedField(fieldName, typeName string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("type '%s' has no field '%s'", typeName, fieldName),
		pos, pos,
	)
}

// errMissingField reports a missing field in a struct literal
func (a *Analyzer) errMissingField(fieldName, typeName string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("missing field '%s' in %s literal", fieldName, typeName),
		pos, pos,
	)
}

// errCannotInstantiateObject reports attempting to instantiate an object
func (a *Analyzer) errCannotInstantiateObject(name string, startPos, endPos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot instantiate object '%s' (objects are singletons)", name),
		startPos, endPos,
	)
}

// errImmutableFieldAssign reports an attempt to assign to an immutable field
func (a *Analyzer) errImmutableFieldAssign(fieldName, typeName string, pos ast.Position) *errors.CompilerError {
	return a.addError(
		fmt.Sprintf("cannot assign to immutable field '%s' of '%s'", fieldName, typeName),
		pos, pos,
	)
}

package ir

import (
	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// =============================================================================
// Nullable Helpers
// =============================================================================

// wrapIfNeeded wraps a value in a nullable type if the target is nullable
// but the value is not. Returns the value unchanged otherwise.
func (g *Generator) wrapIfNeeded(val *Value, targetType Type) *Value {
	if val == nil {
		return nil
	}

	_, targetIsNullable := targetType.(*NullableType)
	if !targetIsNullable {
		return val
	}

	_, valIsNullable := val.Type.(*NullableType)
	if valIsNullable {
		return val
	}

	wrapped := g.block.NewValue(OpWrap, targetType)
	wrapped.AddArg(val)
	return wrapped
}

// isNullLiteral returns true if the expression is a null literal.
func isNullLiteral(expr semantic.TypedExpression) bool {
	lit, ok := expr.(*semantic.TypedLiteralExpr)
	return ok && lit.LitType == ast.LiteralTypeNull
}

// =============================================================================
// Type Assertion Helpers
// =============================================================================

// asNullableType returns the type as *NullableType if it is one, nil otherwise.
func asNullableType(t Type) *NullableType {
	if nt, ok := t.(*NullableType); ok {
		return nt
	}
	return nil
}


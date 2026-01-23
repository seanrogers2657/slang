package semantic

import (
	"fmt"

	"github.com/seanrogers2657/slang/compiler/ast"
)

// FieldContainer is implemented by types that have fields (struct, class).
// This allows unified literal analysis for both struct and class types.
type FieldContainer interface {
	GetFields() []StructFieldInfo
	String() string // for error messages
}

// Ensure StructType implements FieldContainer
func (t StructType) GetFields() []StructFieldInfo {
	return t.Fields
}

// Ensure ClassType implements FieldContainer
func (t ClassType) GetFields() []StructFieldInfo {
	return t.Fields
}

// getTypeKindName returns "struct" or "class" based on the container type
func getTypeKindName(container FieldContainer) string {
	switch container.(type) {
	case ClassType:
		return "class"
	default:
		return "struct"
	}
}

// namedLiteralResult holds the result of analyzing a named literal
type namedLiteralResult struct {
	args  []TypedExpression
	valid bool
}

// analyzeNamedLiteral is the unified handler for named arguments in struct/class literals.
// It validates field names, checks for duplicates, and type-checks values.
// Returns the typed arguments in field order.
func (a *Analyzer) analyzeNamedLiteral(
	container FieldContainer,
	typeName string,
	namedArgs []ast.NamedArgument,
	leftBrace, rightBrace ast.Position,
) namedLiteralResult {
	fields := container.GetFields()

	// Build a map of field name -> index for quick lookup
	fieldIndex := make(map[string]int)
	for i, field := range fields {
		fieldIndex[field.Name] = i
	}

	// Check argument count matches field count
	if len(namedArgs) != len(fields) {
		a.addError(
			fmt.Sprintf("type '%s' has %d field(s), but %d argument(s) were provided",
				typeName, len(fields), len(namedArgs)),
			leftBrace, rightBrace,
		)
	}

	// Track which fields have been provided (for duplicate detection)
	providedFields := make(map[string]ast.Position)

	// Create typed arguments array in field order
	typedArgs := make([]TypedExpression, len(fields))

	// Process named arguments
	for _, namedArg := range namedArgs {
		// Check for duplicate field
		if prevPos, duplicate := providedFields[namedArg.Name]; duplicate {
			a.addError(
				fmt.Sprintf("field '%s' specified multiple times", namedArg.Name),
				namedArg.NamePos, namedArg.NamePos,
			).WithHint(fmt.Sprintf("first specified at line %d", prevPos.Line))
			continue
		}
		providedFields[namedArg.Name] = namedArg.NamePos

		// Look up field index
		idx, ok := fieldIndex[namedArg.Name]
		if !ok {
			a.addError(
				fmt.Sprintf("%s '%s' has no field '%s'", getTypeKindName(container), typeName, namedArg.Name),
				namedArg.NamePos, namedArg.NamePos,
			)
			continue
		}

		// Analyze the value expression
		typedValue := a.analyzeExpression(namedArg.Value)
		typedArgs[idx] = typedValue

		// Type check
		fieldType := fields[idx].Type
		a.checkTypeCompatibilityCore(fieldType, typedValue.GetType(), typedValue, namedArg.Value.Pos(), contextAssignment)
	}

	// Check all fields were provided
	for i, field := range fields {
		if typedArgs[i] == nil {
			// Only report if we haven't already reported a count mismatch
			if len(namedArgs) == len(fields) {
				a.addError(
					fmt.Sprintf("missing field '%s' in %s '%s'", field.Name, getTypeKindName(container), typeName),
					leftBrace, rightBrace,
				)
			}
			// Fill with error expression
			typedArgs[i] = &TypedLiteralExpr{Type: ErrorType{}}
		}
	}

	return namedLiteralResult{args: typedArgs, valid: true}
}

// analyzePositionalLiteral is the unified handler for positional arguments in struct/class literals.
// It validates argument count and type-checks values.
// Returns the typed arguments in order.
func (a *Analyzer) analyzePositionalLiteral(
	container FieldContainer,
	typeName string,
	args []ast.Expression,
	leftBrace, rightBrace ast.Position,
) []TypedExpression {
	fields := container.GetFields()

	// Check argument count matches field count
	if len(args) != len(fields) {
		a.addError(
			fmt.Sprintf("type '%s' has %d field(s), but %d argument(s) were provided",
				typeName, len(fields), len(args)),
			leftBrace, rightBrace,
		)
	}

	// Analyze arguments and check types
	typedArgs := make([]TypedExpression, len(args))
	for i, arg := range args {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding field
		if i < len(fields) {
			fieldType := fields[i].Type
			a.checkTypeCompatibilityCore(fieldType, typedArgs[i].GetType(), typedArgs[i], arg.Pos(), contextAssignment)
		}
	}

	return typedArgs
}

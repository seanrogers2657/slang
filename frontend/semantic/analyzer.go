package semantic

import (
	"fmt"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/errors"
)

// Analyzer performs semantic analysis on the AST
type Analyzer struct {
	filename string
	errors   []*errors.CompilerError
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(filename string) *Analyzer {
	return &Analyzer{
		filename: filename,
		errors:   make([]*errors.CompilerError, 0),
	}
}

// Analyze performs semantic analysis on a program
func (a *Analyzer) Analyze(program *ast.Program) ([]*errors.CompilerError, *TypedProgram) {
	typedStmts := make([]TypedStatement, 0, len(program.Statements))

	for _, stmt := range program.Statements {
		typedStmt := a.analyzeStatement(stmt)
		typedStmts = append(typedStmts, typedStmt)
	}

	typedProgram := &TypedProgram{
		Statements: typedStmts,
		StartPos:   program.StartPos,
		EndPos:     program.EndPos,
	}

	return a.errors, typedProgram
}

// analyzeStatement performs semantic analysis on a statement
func (a *Analyzer) analyzeStatement(stmt ast.Statement) TypedStatement {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return a.analyzeExprStatement(s)
	case *ast.PrintStmt:
		return a.analyzePrintStatement(s)
	default:
		a.addError("unknown statement type", stmt.Pos(), stmt.End())
		return &TypedExprStmt{
			Expr: &TypedLiteralExpr{
				Type:     TypeError,
				LitType:  ast.LiteralTypeNumber,
				Value:    "0",
				StartPos: stmt.Pos(),
				EndPos:   stmt.End(),
			},
		}
	}
}

// analyzeExprStatement analyzes an expression statement
func (a *Analyzer) analyzeExprStatement(stmt *ast.ExprStmt) TypedStatement {
	typedExpr := a.analyzeExpression(stmt.Expr)
	return &TypedExprStmt{
		Expr: typedExpr,
	}
}

// analyzePrintStatement analyzes a print statement
func (a *Analyzer) analyzePrintStatement(stmt *ast.PrintStmt) TypedStatement {
	typedExpr := a.analyzeExpression(stmt.Expr)

	// Print can handle any type, so no type checking needed here
	return &TypedPrintStmt{
		Keyword: stmt.Keyword,
		Expr:    typedExpr,
	}
}

// analyzeExpression performs semantic analysis on an expression
func (a *Analyzer) analyzeExpression(expr ast.Expression) TypedExpression {
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return a.analyzeLiteral(e)
	case *ast.BinaryExpr:
		return a.analyzeBinaryExpression(e)
	default:
		a.addError("unknown expression type", expr.Pos(), expr.End())
		return &TypedLiteralExpr{
			Type:     TypeError,
			LitType:  ast.LiteralTypeNumber,
			Value:    "0",
			StartPos: expr.Pos(),
			EndPos:   expr.End(),
		}
	}
}

// analyzeLiteral analyzes a literal expression
func (a *Analyzer) analyzeLiteral(lit *ast.LiteralExpr) TypedExpression {
	var typ Type
	switch lit.Kind {
	case ast.LiteralTypeNumber:
		typ = TypeInteger
	case ast.LiteralTypeString:
		typ = TypeString
	case ast.LiteralTypeBoolean:
		typ = TypeBoolean
	default:
		a.addError(fmt.Sprintf("unknown literal type: %v", lit.Kind), lit.StartPos, lit.EndPos)
		typ = TypeError
	}

	return &TypedLiteralExpr{
		Type:     typ,
		LitType:  lit.Kind,
		Value:    lit.Value,
		StartPos: lit.StartPos,
		EndPos:   lit.EndPos,
	}
}

// analyzeBinaryExpression analyzes a binary expression
func (a *Analyzer) analyzeBinaryExpression(expr *ast.BinaryExpr) TypedExpression {
	// Analyze left and right operands
	left := a.analyzeExpression(expr.Left)
	right := a.analyzeExpression(expr.Right)

	leftType := left.GetType()
	rightType := right.GetType()

	// Determine the result type and check type compatibility
	resultType := a.checkBinaryOperation(expr.Op, leftType, rightType, expr.LeftPos, expr.RightPos)

	return &TypedBinaryExpr{
		Type:     resultType,
		Left:     left,
		Op:       expr.Op,
		Right:    right,
		LeftPos:  expr.LeftPos,
		OpPos:    expr.OpPos,
		RightPos: expr.RightPos,
	}
}

// checkBinaryOperation checks if a binary operation is valid and returns the result type
func (a *Analyzer) checkBinaryOperation(op string, leftType, rightType Type, leftPos, rightPos ast.Position) Type {
	// Check for error types - propagate them
	if _, ok := leftType.(ErrorType); ok {
		return TypeError
	}
	if _, ok := rightType.(ErrorType); ok {
		return TypeError
	}

	// Arithmetic operators: +, -, *, /, %
	// These require integer operands and return integer
	if op == "+" || op == "-" || op == "*" || op == "/" || op == "%" {
		if !leftType.Equals(TypeInteger) {
			a.addError(
				fmt.Sprintf("operator '%s' requires integer operands, but left operand has type '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("arithmetic operators only work with integers")
			return TypeError
		}
		if !rightType.Equals(TypeInteger) {
			a.addError(
				fmt.Sprintf("operator '%s' requires integer operands, but right operand has type '%s'", op, rightType.String()),
				rightPos, rightPos,
			).WithHint("arithmetic operators only work with integers")
			return TypeError
		}
		return TypeInteger
	}

	// Comparison operators: ==, !=, <, >, <=, >=
	// These require matching operand types and return integer (0 or 1)
	if op == "==" || op == "!=" || op == "<" || op == ">" || op == "<=" || op == ">=" {
		// For now, we only support integer comparisons
		if !leftType.Equals(TypeInteger) {
			a.addError(
				fmt.Sprintf("operator '%s' requires integer operands, but left operand has type '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("comparison operators currently only work with integers")
			return TypeError
		}
		if !rightType.Equals(TypeInteger) {
			a.addError(
				fmt.Sprintf("operator '%s' requires integer operands, but right operand has type '%s'", op, rightType.String()),
				rightPos, rightPos,
			).WithHint("comparison operators currently only work with integers")
			return TypeError
		}

		// Comparison result is an integer (0 or 1) in our system
		return TypeInteger
	}

	// Unknown operator
	a.addError(fmt.Sprintf("unknown binary operator: '%s'", op), leftPos, rightPos)
	return TypeError
}

// addError adds a compiler error to the error list
func (a *Analyzer) addError(message string, startPos, endPos ast.Position) *errors.CompilerError {
	err := errors.NewErrorWithSpan(message, a.filename, startPos, endPos, "semantic")
	a.errors = append(a.errors, err)
	return err
}

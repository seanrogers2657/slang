package semantic

import (
	"fmt"

	"github.com/seanrogers2657/slang/errors"
	"github.com/seanrogers2657/slang/frontend/ast"
)

// toErrorPos converts an ast.Position to an errors.Position
func toErrorPos(p ast.Position) errors.Position {
	return errors.Position{Line: p.Line, Column: p.Column, Offset: p.Offset}
}

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
	typedProgram := &TypedProgram{
		Declarations: make([]TypedDeclaration, 0),
		Statements:   make([]TypedStatement, 0),
		StartPos:     program.StartPos,
		EndPos:       program.EndPos,
	}

	// Handle function-based programs
	if len(program.Declarations) > 0 {
		// Check that a main function exists
		hasMain := false
		for _, decl := range program.Declarations {
			typedDecl := a.analyzeDeclaration(decl)
			typedProgram.Declarations = append(typedProgram.Declarations, typedDecl)

			// Check if this is the main function
			if fnDecl, ok := decl.(*ast.FunctionDecl); ok {
				if fnDecl.Name == "main" {
					hasMain = true
				}
			}
		}

		if !hasMain {
			// Add error if no main function found
			a.addError("program must have a 'main' function", program.StartPos, program.EndPos)
		}
	} else {
		// Handle legacy statement-based programs
		for _, stmt := range program.Statements {
			typedStmt := a.analyzeStatement(stmt)
			typedProgram.Statements = append(typedProgram.Statements, typedStmt)
		}
	}

	return a.errors, typedProgram
}

// analyzeDeclaration performs semantic analysis on a declaration
func (a *Analyzer) analyzeDeclaration(decl ast.Declaration) TypedDeclaration {
	switch d := decl.(type) {
	case *ast.FunctionDecl:
		return a.analyzeFunctionDecl(d)
	default:
		a.addError("unknown declaration type", decl.Pos(), decl.End())
		return &TypedFunctionDecl{
			FnKeyword:  decl.Pos(),
			Name:       "error",
			NamePos:    decl.Pos(),
			LeftParen:  decl.Pos(),
			RightParen: decl.Pos(),
			Body: &TypedBlockStmt{
				LeftBrace:  decl.Pos(),
				Statements: []TypedStatement{},
				RightBrace: decl.End(),
			},
		}
	}
}

// analyzeFunctionDecl analyzes a function declaration
func (a *Analyzer) analyzeFunctionDecl(fn *ast.FunctionDecl) TypedDeclaration {
	// Analyze the function body
	typedBody := a.analyzeBlockStmt(fn.Body)

	return &TypedFunctionDecl{
		FnKeyword:  fn.FnKeyword,
		Name:       fn.Name,
		NamePos:    fn.NamePos,
		LeftParen:  fn.LeftParen,
		RightParen: fn.RightParen,
		Body:       typedBody,
	}
}

// analyzeBlockStmt analyzes a block statement
func (a *Analyzer) analyzeBlockStmt(block *ast.BlockStmt) *TypedBlockStmt {
	typedStmts := make([]TypedStatement, 0, len(block.Statements))

	for _, stmt := range block.Statements {
		typedStmt := a.analyzeStatement(stmt)
		typedStmts = append(typedStmts, typedStmt)
	}

	return &TypedBlockStmt{
		LeftBrace:  block.LeftBrace,
		Statements: typedStmts,
		RightBrace: block.RightBrace,
	}
}

// analyzeStatement performs semantic analysis on a statement
func (a *Analyzer) analyzeStatement(stmt ast.Statement) TypedStatement {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return a.analyzeExprStatement(s)
	case *ast.PrintStmt:
		return a.analyzePrintStatement(s)
	case *ast.BlockStmt:
		return a.analyzeBlockStmt(s)
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
	err := errors.NewErrorWithSpan(message, a.filename, toErrorPos(startPos), toErrorPos(endPos), "semantic")
	err.Tool = errors.ToolSL
	a.errors = append(a.errors, err)
	return err
}

package parser

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/lexer"
)

func TestParserLiterals(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *ast.LiteralExpr
	}{
		{
			name: "single digit",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeInteger,
				Value: "5",
			},
		},
		{
			name: "multiple digits",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "123", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeInteger,
				Value: "123",
			},
		},
		{
			name: "zero",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "0", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeInteger,
				Value: "0",
			},
		},
		{
			name: "simple string",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeString, Value: "hello", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeString,
				Value: "hello",
			},
		},
		{
			name: "empty string",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeString, Value: "", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeString,
				Value: "",
			},
		},
		{
			name: "string with spaces",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeString, Value: "hello world", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeString,
				Value: "hello world",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			literal := p.ParseLiteral()

			if literal == nil {
				t.Fatal("expected literal, got nil")
			}

			litExpr, ok := literal.(*ast.LiteralExpr)
			if !ok {
				t.Fatal("expected LiteralExpr")
			}

			if litExpr.Kind != tt.expected.Kind {
				t.Errorf("expected type %d, got %d", tt.expected.Kind, litExpr.Kind)
			}

			if litExpr.Value != tt.expected.Value {
				t.Errorf("expected value %q, got %q", tt.expected.Value, litExpr.Value)
			}
		})
	}
}

func TestParserBinaryExpressions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *ast.BinaryExpr
	}{
		{
			name: "addition",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
			},
			expected: &ast.BinaryExpr{
				Op: "+",
			},
		},
		{
			name: "subtraction",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "10", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeMinus, Value: "-", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: &ast.BinaryExpr{
				Op: "-",
			},
		},
		{
			name: "multiplication",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "4", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeMultiply, Value: "*", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "7", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
			},
			expected: &ast.BinaryExpr{
				Op: "*",
			},
		},
		{
			name: "division",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "20", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeDivide, Value: "/", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "4", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: &ast.BinaryExpr{
				Op: "/",
			},
		},
		{
			name: "modulo",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "10", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeModulo, Value: "%", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: &ast.BinaryExpr{
				Op: "%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			expr := p.ParseBinaryExpression()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", p.Errors)
			}

			if expr == nil {
				t.Fatal("expected expression, got nil")
			}

			binExpr, ok := expr.(*ast.BinaryExpr)
			if !ok {
				t.Fatal("expected BinaryExpr")
			}

			if binExpr.Op != tt.expected.Op {
				t.Errorf("expected operator %q, got %q", tt.expected.Op, binExpr.Op)
			}

			if binExpr.Left == nil || binExpr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}
		})
	}
}

func TestParserComparisonExpressions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected string
	}{
		{
			name: "equality",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeEqual, Value: "==", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: "==",
		},
		{
			name: "inequality",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeNotEqual, Value: "!=", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "4", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: "!=",
		},
		{
			name: "less than",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeLessThan, Value: "<", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "8", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
			},
			expected: "<",
		},
		{
			name: "greater than",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "9", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeGreaterThan, Value: ">", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "1", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
			},
			expected: ">",
		},
		{
			name: "less than or equal",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeLessThanOrEqual, Value: "<=", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: "<=",
		},
		{
			name: "greater than or equal",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "7", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeGreaterThanOrEqual, Value: ">=", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "7", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: ">=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			expr := p.ParseBinaryExpression()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", p.Errors)
			}

			if expr == nil {
				t.Fatal("expected expression, got nil")
			}

			binExpr, ok := expr.(*ast.BinaryExpr)
			if !ok {
				t.Fatal("expected BinaryExpr")
			}

			if binExpr.Op != tt.expected {
				t.Errorf("expected operator %q, got %q", tt.expected, binExpr.Op)
			}

			if binExpr.Left == nil || binExpr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}
		})
	}
}

func TestParserParse(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected string // expected operator
	}{
		{
			name: "simple addition expression",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
			},
			expected: "+",
		},
		{
			name: "with newline",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
			},
			expected: "+",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0]
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt, got different statement type")
			}

			binExpr, ok := exprStmt.Expr.(*ast.BinaryExpr)
			if !ok {
				t.Fatal("expected BinaryExpr")
			}

			if binExpr.Op != tt.expected {
				t.Errorf("expected operator %q, got %q", tt.expected, binExpr.Op)
			}

			if binExpr.Left == nil || binExpr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}
		})
	}
}

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name          string
		tokens        []lexer.Token
		expectedError string
	}{
		{
			name: "missing operand after operator",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
			},
			expectedError: "expected expression after operator '+'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			expr := p.ParseBinaryExpression()

			if len(p.Errors) == 0 {
				t.Fatal("expected error, got none")
			}

			if p.Errors[0].Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, p.Errors[0].Error())
			}

			if expr != nil {
				t.Error("expected nil expression on error")
			}
		})
	}
}

func TestParserMultipleStatements(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		numStmts int
		ops      []string // expected operators
	}{
		{
			name: "two statements",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
				{Type: lexer.TokenTypeInteger, Value: "10", Pos: ast.Position{Line: 2, Column: 1, Offset: 6}},
				{Type: lexer.TokenTypeMinus, Value: "-", Pos: ast.Position{Line: 2, Column: 4, Offset: 9}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 2, Column: 6, Offset: 11}},
			},
			numStmts: 2,
			ops:      []string{"+", "-"},
		},
		{
			name: "three statements with multiple newlines",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "1", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeInteger, Value: "1", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 2, Column: 1, Offset: 6}},
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 3, Column: 1, Offset: 7}},
				{Type: lexer.TokenTypeMultiply, Value: "*", Pos: ast.Position{Line: 3, Column: 3, Offset: 9}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 3, Column: 5, Offset: 11}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 3, Column: 6, Offset: 12}},
				{Type: lexer.TokenTypeInteger, Value: "4", Pos: ast.Position{Line: 4, Column: 1, Offset: 13}},
				{Type: lexer.TokenTypeDivide, Value: "/", Pos: ast.Position{Line: 4, Column: 3, Offset: 15}},
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 4, Column: 5, Offset: 17}},
			},
			numStmts: 3,
			ops:      []string{"+", "*", "/"},
		},
		{
			name: "leading and trailing newlines",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 2, Column: 1, Offset: 1}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 2, Column: 3, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 2, Column: 5, Offset: 5}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 2, Column: 6, Offset: 6}},
				{Type: lexer.TokenTypeNewline, Value: "\n", Pos: ast.Position{Line: 3, Column: 1, Offset: 7}},
			},
			numStmts: 1,
			ops:      []string{"+"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != tt.numStmts {
				t.Fatalf("expected %d statements, got %d", tt.numStmts, len(program.Statements))
			}

			for i, expectedOp := range tt.ops {
				stmt := program.Statements[i]
				exprStmt, ok := stmt.(*ast.ExprStmt)
				if !ok {
					t.Fatalf("statement %d: expected ExprStmt, got different statement type", i)
				}

				binExpr, ok := exprStmt.Expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("statement %d: expected BinaryExpr", i)
				}

				if binExpr.Op != expectedOp {
					t.Errorf("statement %d: expected operator %q, got %q", i, expectedOp, binExpr.Op)
				}
			}
		})
	}
}

func TestParserOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		description string
		validate    func(t *testing.T, expr ast.Expression)
	}{
		{
			name:        "multiplication before addition",
			source:      "2 + 3 * 4",
			description: "should parse as 2 + (3 * 4)",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be addition
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected BinaryExpr at top level")
				}
				if binExpr.Op != "+" {
					t.Errorf("expected top-level operator '+', got %q", binExpr.Op)
				}

				// Left should be literal 2
				leftLit, ok := binExpr.Left.(*ast.LiteralExpr)
				if !ok {
					t.Fatal("expected left operand to be literal")
				}
				if leftLit.Value != "2" {
					t.Errorf("expected left literal '2', got %q", leftLit.Value)
				}

				// Right should be multiplication (3 * 4)
				rightBin, ok := binExpr.Right.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected right operand to be BinaryExpr")
				}
				if rightBin.Op != "*" {
					t.Errorf("expected right operator '*', got %q", rightBin.Op)
				}
			},
		},
		{
			name:        "division before subtraction",
			source:      "10 - 6 / 2",
			description: "should parse as 10 - (6 / 2)",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr := expr.(*ast.BinaryExpr)
				if binExpr.Op != "-" {
					t.Errorf("expected top-level operator '-', got %q", binExpr.Op)
				}

				rightBin := binExpr.Right.(*ast.BinaryExpr)
				if rightBin.Op != "/" {
					t.Errorf("expected right operator '/', got %q", rightBin.Op)
				}
			},
		},
		{
			name:        "comparison has lower precedence than addition",
			source:      "2 + 3 < 10",
			description: "should parse as (2 + 3) < 10",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr := expr.(*ast.BinaryExpr)
				if binExpr.Op != "<" {
					t.Errorf("expected top-level operator '<', got %q", binExpr.Op)
				}

				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected left operand to be BinaryExpr")
				}
				if leftBin.Op != "+" {
					t.Errorf("expected left operator '+', got %q", leftBin.Op)
				}
			},
		},
		{
			name:        "left associativity for same precedence",
			source:      "2 + 3 + 4",
			description: "should parse as (2 + 3) + 4",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr := expr.(*ast.BinaryExpr)
				if binExpr.Op != "+" {
					t.Errorf("expected top-level operator '+', got %q", binExpr.Op)
				}

				// Right should be literal 4
				rightLit, ok := binExpr.Right.(*ast.LiteralExpr)
				if !ok {
					t.Fatal("expected right operand to be literal")
				}
				if rightLit.Value != "4" {
					t.Errorf("expected right literal '4', got %q", rightLit.Value)
				}

				// Left should be (2 + 3)
				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected left operand to be BinaryExpr")
				}
				if leftBin.Op != "+" {
					t.Errorf("expected left operator '+', got %q", leftBin.Op)
				}
			},
		},
		{
			name:        "complex mixed precedence",
			source:      "2 + 3 * 4 == 14",
			description: "should parse as (2 + (3 * 4)) == 14",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be ==
				binExpr := expr.(*ast.BinaryExpr)
				if binExpr.Op != "==" {
					t.Errorf("expected top-level operator '==', got %q", binExpr.Op)
				}

				// Left should be (2 + (3 * 4))
				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected left operand to be BinaryExpr")
				}
				if leftBin.Op != "+" {
					t.Errorf("expected left operator '+', got %q", leftBin.Op)
				}

				// Left's right should be (3 * 4)
				leftRightBin, ok := leftBin.Right.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected left's right operand to be BinaryExpr")
				}
				if leftRightBin.Op != "*" {
					t.Errorf("expected nested operator '*', got %q", leftRightBin.Op)
				}
			},
		},
		{
			name:        "modulo same precedence as multiplication",
			source:      "10 % 3 * 2",
			description: "should parse as (10 % 3) * 2",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr := expr.(*ast.BinaryExpr)
				if binExpr.Op != "*" {
					t.Errorf("expected top-level operator '*', got %q", binExpr.Op)
				}

				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatal("expected left operand to be BinaryExpr")
				}
				if leftBin.Op != "%" {
					t.Errorf("expected left operator '%%', got %q", leftBin.Op)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0]
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt")
			}

			tt.validate(t, exprStmt.Expr)
		})
	}
}

func TestParserIntegrationWithLexer(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string // expected operator
	}{
		{
			name:     "simple addition",
			source:   "2 + 5",
			expected: "+",
		},
		{
			name:     "subtraction",
			source:   "10 - 3",
			expected: "-",
		},
		{
			name:     "multiplication",
			source:   "4 * 7",
			expected: "*",
		},
		{
			name:     "division",
			source:   "20 / 4",
			expected: "/",
		},
		{
			name:     "comparison",
			source:   "5 == 5",
			expected: "==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0]
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt, got different statement type")
			}

			binExpr, ok := exprStmt.Expr.(*ast.BinaryExpr)
			if !ok {
				t.Fatal("expected BinaryExpr")
			}

			if binExpr.Op != tt.expected {
				t.Errorf("expected operator %q, got %q", tt.expected, binExpr.Op)
			}

			if binExpr.Left == nil || binExpr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}
		})
	}
}

func TestParserFunctionDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectedName string
		expectedBody int // number of statements in body
	}{
		{
			name:         "empty main function",
			source:       "main = () {}",
			expectedName: "main",
			expectedBody: 0,
		},
		{
			name:         "main function with single statement",
			source:       "main = () {\n    print(42)\n}",
			expectedName: "main",
			expectedBody: 1,
		},
		{
			name:         "main function with multiple statements",
			source:       "main = () {\n    print(1)\n    print(2)\n}",
			expectedName: "main",
			expectedBody: 2,
		},
		{
			name:         "function with expression statement",
			source:       "main = () {\n    5 + 3\n}",
			expectedName: "main",
			expectedBody: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			decl := program.Declarations[0]
			fnDecl, ok := decl.(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			if fnDecl.Name != tt.expectedName {
				t.Errorf("expected function name %q, got %q", tt.expectedName, fnDecl.Name)
			}

			if fnDecl.Body == nil {
				t.Fatal("expected function body, got nil")
			}

			if len(fnDecl.Body.Statements) != tt.expectedBody {
				t.Errorf("expected %d statements in body, got %d", tt.expectedBody, len(fnDecl.Body.Statements))
			}
		})
	}
}

func TestParserVariableDeclaration(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		varName     string
		expectError bool
	}{
		{
			name:    "simple variable declaration",
			source:  "main = () {\n    val x = 5\n}",
			varName: "x",
		},
		{
			name:    "variable with expression",
			source:  "main = () {\n    val result = 10 + 20\n}",
			varName: "result",
		},
		{
			name:    "variable with underscore name",
			source:  "main = () {\n    val my_var = 42\n}",
			varName: "my_var",
		},
		{
			name:    "variable with digits in name",
			source:  "main = () {\n    val x1 = 100\n}",
			varName: "x1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if tt.expectError {
				if len(p.Errors) == 0 {
					t.Fatal("expected parser error, got none")
				}
				return
			}

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			fnDecl := program.Declarations[0].(*ast.FunctionDecl)
			if len(fnDecl.Body.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(fnDecl.Body.Statements))
			}

			varDecl, ok := fnDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatalf("expected VarDeclStmt, got %T", fnDecl.Body.Statements[0])
			}

			if varDecl.Name != tt.varName {
				t.Errorf("expected variable name %q, got %q", tt.varName, varDecl.Name)
			}

			if varDecl.Initializer == nil {
				t.Error("expected initializer expression, got nil")
			}
		})
	}
}

func TestParserMutableVariableDeclaration(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		varName string
		mutable bool
	}{
		{
			name:    "immutable variable with val",
			source:  "main = () {\n    val x = 5\n}",
			varName: "x",
			mutable: false,
		},
		{
			name:    "mutable variable with var",
			source:  "main = () {\n    var x = 5\n}",
			varName: "x",
			mutable: true,
		},
		{
			name:    "mutable variable with expression",
			source:  "main = () {\n    var result = 10 + 20\n}",
			varName: "result",
			mutable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			fnDecl := program.Declarations[0].(*ast.FunctionDecl)
			varDecl, ok := fnDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatalf("expected VarDeclStmt, got %T", fnDecl.Body.Statements[0])
			}

			if varDecl.Name != tt.varName {
				t.Errorf("expected variable name %q, got %q", tt.varName, varDecl.Name)
			}

			if varDecl.Mutable != tt.mutable {
				t.Errorf("expected Mutable=%v, got %v", tt.mutable, varDecl.Mutable)
			}
		})
	}
}

func TestParserAssignmentStatement(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		varName string
	}{
		{
			name:    "simple assignment",
			source:  "main = () {\n    var x = 5\n    x = 10\n}",
			varName: "x",
		},
		{
			name:    "assignment with expression",
			source:  "main = () {\n    var x = 5\n    x = x + 10\n}",
			varName: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			fnDecl := program.Declarations[0].(*ast.FunctionDecl)
			if len(fnDecl.Body.Statements) < 2 {
				t.Fatalf("expected at least 2 statements, got %d", len(fnDecl.Body.Statements))
			}

			// Second statement should be assignment
			assignStmt, ok := fnDecl.Body.Statements[1].(*ast.AssignStmt)
			if !ok {
				t.Fatalf("expected AssignStmt, got %T", fnDecl.Body.Statements[1])
			}

			if assignStmt.Name != tt.varName {
				t.Errorf("expected variable name %q, got %q", tt.varName, assignStmt.Name)
			}

			if assignStmt.Value == nil {
				t.Error("expected value expression, got nil")
			}
		})
	}
}

func TestParserIdentifierExpression(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectedName string
	}{
		{
			name:         "simple identifier",
			source:       "main = () {\n    print(x)\n}",
			expectedName: "x",
		},
		{
			name:         "identifier with underscore",
			source:       "main = () {\n    print(my_var)\n}",
			expectedName: "my_var",
		},
		{
			name:         "identifier in binary expression",
			source:       "main = () {\n    val x = 5\n    print(x + 10)\n}",
			expectedName: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			// Find the identifier in the AST
			fnDecl := program.Declarations[0].(*ast.FunctionDecl)

			var foundIdent bool
			for _, stmt := range fnDecl.Body.Statements {
				if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
					if callExpr, ok := exprStmt.Expr.(*ast.CallExpr); ok && callExpr.Name == "print" {
						// Check if the print argument contains an identifier
						if len(callExpr.Arguments) > 0 {
							switch e := callExpr.Arguments[0].(type) {
							case *ast.IdentifierExpr:
								if e.Name == tt.expectedName {
									foundIdent = true
								}
							case *ast.BinaryExpr:
								if left, ok := e.Left.(*ast.IdentifierExpr); ok && left.Name == tt.expectedName {
									foundIdent = true
								}
							}
						}
					}
				}
			}

			if !foundIdent {
				t.Errorf("expected to find identifier %q", tt.expectedName)
			}
		})
	}
}

func TestParserFloatLiterals(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *ast.LiteralExpr
	}{
		{
			name: "simple float",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeFloat, Value: "3.14", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeFloat,
				Value: "3.14",
			},
		},
		{
			name: "scientific notation",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeFloat, Value: "1e10", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeFloat,
				Value: "1e10",
			},
		},
		{
			name: "negative exponent",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeFloat, Value: "2.5e-3", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeFloat,
				Value: "2.5e-3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			literal := p.ParseLiteral()

			if literal == nil {
				t.Fatal("expected literal, got nil")
			}

			litExpr, ok := literal.(*ast.LiteralExpr)
			if !ok {
				t.Fatal("expected LiteralExpr")
			}

			if litExpr.Kind != tt.expected.Kind {
				t.Errorf("expected type %d, got %d", tt.expected.Kind, litExpr.Kind)
			}

			if litExpr.Value != tt.expected.Value {
				t.Errorf("expected value %q, got %q", tt.expected.Value, litExpr.Value)
			}
		})
	}
}

func TestParserTypeAnnotation(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		varName      string
		expectedType string
	}{
		{
			name:         "i8 type annotation",
			source:       "main = () {\n    val x: i8 = 42\n}",
			varName:      "x",
			expectedType: "i8",
		},
		{
			name:         "i32 type annotation",
			source:       "main = () {\n    val y: i32 = 100\n}",
			varName:      "y",
			expectedType: "i32",
		},
		{
			name:         "u64 type annotation",
			source:       "main = () {\n    val z: u64 = 999\n}",
			varName:      "z",
			expectedType: "u64",
		},
		{
			name:         "f64 type annotation",
			source:       "main = () {\n    val pi: f64 = 3.14\n}",
			varName:      "pi",
			expectedType: "f64",
		},
		{
			name:         "no type annotation",
			source:       "main = () {\n    val x = 42\n}",
			varName:      "x",
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			fnDecl := program.Declarations[0].(*ast.FunctionDecl)
			varDecl, ok := fnDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatalf("expected VarDeclStmt, got %T", fnDecl.Body.Statements[0])
			}

			if varDecl.Name != tt.varName {
				t.Errorf("expected variable name %q, got %q", tt.varName, varDecl.Name)
			}

			if varDecl.TypeName != tt.expectedType {
				t.Errorf("expected type name %q, got %q", tt.expectedType, varDecl.TypeName)
			}
		})
	}
}

func TestParserOptionalReturnType(t *testing.T) {
	tests := []struct {
		name               string
		source             string
		expectedName       string
		expectedReturnType string
		expectedBody       int
	}{
		{
			name:               "function without return type defaults to void",
			source:             "main = () {}",
			expectedName:       "main",
			expectedReturnType: "void",
			expectedBody:       0,
		},
		{
			name:               "function without return type with body",
			source:             "main = () {\n    print(42)\n}",
			expectedName:       "main",
			expectedReturnType: "void",
			expectedBody:       1,
		},
		{
			name:               "function with explicit void return type",
			source:             "main = () {}",
			expectedName:       "main",
			expectedReturnType: "void",
			expectedBody:       0,
		},
		{
			name:               "function with explicit s64 return type",
			source:             "add = () -> s64 {\n    return 42\n}",
			expectedName:       "add",
			expectedReturnType: "s64",
			expectedBody:       1,
		},
		{
			name:               "function without return type and newline before body",
			source:             "main = ()\n{}",
			expectedName:       "main",
			expectedReturnType: "void",
			expectedBody:       0,
		},
		{
			name:               "function with generic return type Own<Point>",
			source:             "create = () -> Own<Point> {\n    return null\n}",
			expectedName:       "create",
			expectedReturnType: "Own<Point>",
			expectedBody:       1,
		},
		{
			name:               "function with array return type s64[]",
			source:             "getArray = () -> s64[] {\n    return [1, 2, 3]\n}",
			expectedName:       "getArray",
			expectedReturnType: "s64[]",
			expectedBody:       1,
		},
		{
			name:               "function with nullable generic return type Own<Point>?",
			source:             "maybeCreate = () -> Own<Point>? {\n    return null\n}",
			expectedName:       "maybeCreate",
			expectedReturnType: "Own<Point>?",
			expectedBody:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			fnDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			if fnDecl.Name != tt.expectedName {
				t.Errorf("expected function name %q, got %q", tt.expectedName, fnDecl.Name)
			}

			if fnDecl.ReturnType != tt.expectedReturnType {
				t.Errorf("expected return type %q, got %q", tt.expectedReturnType, fnDecl.ReturnType)
			}

			if fnDecl.Body == nil {
				t.Fatal("expected function body, got nil")
			}

			if len(fnDecl.Body.Statements) != tt.expectedBody {
				t.Errorf("expected %d statements in body, got %d", tt.expectedBody, len(fnDecl.Body.Statements))
			}
		})
	}
}

func TestParserBooleanLiterals(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *ast.LiteralExpr
	}{
		{
			name: "true literal",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeTrue, Value: "true", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeBoolean,
				Value: "true",
			},
		},
		{
			name: "false literal",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeFalse, Value: "false", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeBoolean,
				Value: "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			literal := p.ParseLiteral()

			if literal == nil {
				t.Fatal("expected literal, got nil")
			}

			litExpr, ok := literal.(*ast.LiteralExpr)
			if !ok {
				t.Fatal("expected LiteralExpr")
			}

			if litExpr.Kind != tt.expected.Kind {
				t.Errorf("expected type %d, got %d", tt.expected.Kind, litExpr.Kind)
			}

			if litExpr.Value != tt.expected.Value {
				t.Errorf("expected value %q, got %q", tt.expected.Value, litExpr.Value)
			}
		})
	}
}

func TestParserUnaryNot(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		description string
		validate    func(t *testing.T, expr ast.Expression)
	}{
		{
			name:        "not true",
			source:      "!true",
			description: "should parse as UnaryExpr with ! operator",
			validate: func(t *testing.T, expr ast.Expression) {
				unaryExpr, ok := expr.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr, got %T", expr)
				}
				if unaryExpr.Op != "!" {
					t.Errorf("expected operator '!', got %q", unaryExpr.Op)
				}
				litExpr, ok := unaryExpr.Operand.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr operand, got %T", unaryExpr.Operand)
				}
				if litExpr.Value != "true" {
					t.Errorf("expected operand 'true', got %q", litExpr.Value)
				}
			},
		},
		{
			name:        "not false",
			source:      "!false",
			description: "should parse as UnaryExpr with ! operator",
			validate: func(t *testing.T, expr ast.Expression) {
				unaryExpr, ok := expr.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr, got %T", expr)
				}
				litExpr, ok := unaryExpr.Operand.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr operand, got %T", unaryExpr.Operand)
				}
				if litExpr.Value != "false" {
					t.Errorf("expected operand 'false', got %q", litExpr.Value)
				}
			},
		},
		{
			name:        "double not",
			source:      "!!true",
			description: "should parse as nested UnaryExpr",
			validate: func(t *testing.T, expr ast.Expression) {
				outerUnary, ok := expr.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr, got %T", expr)
				}
				innerUnary, ok := outerUnary.Operand.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected nested UnaryExpr, got %T", outerUnary.Operand)
				}
				litExpr, ok := innerUnary.Operand.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr innermost, got %T", innerUnary.Operand)
				}
				if litExpr.Value != "true" {
					t.Errorf("expected innermost operand 'true', got %q", litExpr.Value)
				}
			},
		},
		{
			name:        "not identifier",
			source:      "!x",
			description: "should parse as UnaryExpr with identifier operand",
			validate: func(t *testing.T, expr ast.Expression) {
				unaryExpr, ok := expr.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr, got %T", expr)
				}
				identExpr, ok := unaryExpr.Operand.(*ast.IdentifierExpr)
				if !ok {
					t.Fatalf("expected IdentifierExpr operand, got %T", unaryExpr.Operand)
				}
				if identExpr.Name != "x" {
					t.Errorf("expected identifier 'x', got %q", identExpr.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0]
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt")
			}

			tt.validate(t, exprStmt.Expr)
		})
	}
}

func TestParserLogicalOperators(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		description string
		validate    func(t *testing.T, expr ast.Expression)
	}{
		{
			name:        "logical and",
			source:      "a && b",
			description: "should parse as BinaryExpr with && operator",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "&&" {
					t.Errorf("expected operator '&&', got %q", binExpr.Op)
				}
			},
		},
		{
			name:        "logical or",
			source:      "a || b",
			description: "should parse as BinaryExpr with || operator",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "||" {
					t.Errorf("expected operator '||', got %q", binExpr.Op)
				}
			},
		},
		{
			name:        "true and false",
			source:      "true && false",
			description: "should parse boolean literals with && operator",
			validate: func(t *testing.T, expr ast.Expression) {
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "&&" {
					t.Errorf("expected operator '&&', got %q", binExpr.Op)
				}
				leftLit, ok := binExpr.Left.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr left, got %T", binExpr.Left)
				}
				if leftLit.Value != "true" {
					t.Errorf("expected left 'true', got %q", leftLit.Value)
				}
				rightLit, ok := binExpr.Right.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr right, got %T", binExpr.Right)
				}
				if rightLit.Value != "false" {
					t.Errorf("expected right 'false', got %q", rightLit.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0]
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt")
			}

			tt.validate(t, exprStmt.Expr)
		})
	}
}

func TestParserBooleanPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		description string
		validate    func(t *testing.T, expr ast.Expression)
	}{
		{
			name:        "and binds tighter than or",
			source:      "a || b && c",
			description: "should parse as a || (b && c)",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be ||
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "||" {
					t.Errorf("expected top-level operator '||', got %q", binExpr.Op)
				}

				// Left should be identifier 'a'
				leftIdent, ok := binExpr.Left.(*ast.IdentifierExpr)
				if !ok {
					t.Fatalf("expected IdentifierExpr left, got %T", binExpr.Left)
				}
				if leftIdent.Name != "a" {
					t.Errorf("expected left 'a', got %q", leftIdent.Name)
				}

				// Right should be && expression
				rightBin, ok := binExpr.Right.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr right, got %T", binExpr.Right)
				}
				if rightBin.Op != "&&" {
					t.Errorf("expected right operator '&&', got %q", rightBin.Op)
				}
			},
		},
		{
			name:        "comparison binds tighter than and",
			source:      "a == b && c == d",
			description: "should parse as (a == b) && (c == d)",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be &&
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "&&" {
					t.Errorf("expected top-level operator '&&', got %q", binExpr.Op)
				}

				// Left should be == expression
				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr left, got %T", binExpr.Left)
				}
				if leftBin.Op != "==" {
					t.Errorf("expected left operator '==', got %q", leftBin.Op)
				}

				// Right should be == expression
				rightBin, ok := binExpr.Right.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr right, got %T", binExpr.Right)
				}
				if rightBin.Op != "==" {
					t.Errorf("expected right operator '==', got %q", rightBin.Op)
				}
			},
		},
		{
			name:        "not binds tighter than and",
			source:      "!a && b",
			description: "should parse as (!a) && b",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be &&
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "&&" {
					t.Errorf("expected top-level operator '&&', got %q", binExpr.Op)
				}

				// Left should be unary !
				leftUnary, ok := binExpr.Left.(*ast.UnaryExpr)
				if !ok {
					t.Fatalf("expected UnaryExpr left, got %T", binExpr.Left)
				}
				if leftUnary.Op != "!" {
					t.Errorf("expected left operator '!', got %q", leftUnary.Op)
				}
			},
		},
		{
			name:        "arithmetic binds tighter than comparison in boolean context",
			source:      "x + 1 < 10 && y > 0",
			description: "should parse as ((x + 1) < 10) && (y > 0)",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be &&
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "&&" {
					t.Errorf("expected top-level operator '&&', got %q", binExpr.Op)
				}

				// Left should be < expression
				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr left, got %T", binExpr.Left)
				}
				if leftBin.Op != "<" {
					t.Errorf("expected left operator '<', got %q", leftBin.Op)
				}

				// Left's left should be + expression
				leftLeftBin, ok := leftBin.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr left.left, got %T", leftBin.Left)
				}
				if leftLeftBin.Op != "+" {
					t.Errorf("expected left.left operator '+', got %q", leftLeftBin.Op)
				}
			},
		},
		{
			name:        "complex boolean expression",
			source:      "a || b && c || d",
			description: "should parse as (a || (b && c)) || d due to left-associativity of ||",
			validate: func(t *testing.T, expr ast.Expression) {
				// Top level should be ||
				binExpr, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr, got %T", expr)
				}
				if binExpr.Op != "||" {
					t.Errorf("expected top-level operator '||', got %q", binExpr.Op)
				}

				// Right should be identifier 'd'
				rightIdent, ok := binExpr.Right.(*ast.IdentifierExpr)
				if !ok {
					t.Fatalf("expected IdentifierExpr right, got %T", binExpr.Right)
				}
				if rightIdent.Name != "d" {
					t.Errorf("expected right 'd', got %q", rightIdent.Name)
				}

				// Left should be || expression
				leftBin, ok := binExpr.Left.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr left, got %T", binExpr.Left)
				}
				if leftBin.Op != "||" {
					t.Errorf("expected left operator '||', got %q", leftBin.Op)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0]
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt")
			}

			tt.validate(t, exprStmt.Expr)
		})
	}
}

func TestParserBooleanInFunction(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "boolean variable declaration",
			source: `main = () {
    val x = true
}`,
		},
		{
			name: "boolean expression in variable",
			source: `main = () {
    val x = true && false
}`,
		},
		{
			name: "comparison result in variable",
			source: `main = () {
    val x = 5 < 10
}`,
		},
		{
			name: "complex boolean in variable",
			source: `main = () {
    val x = 5 < 10 && 3 > 1
}`,
		},
		{
			name: "negation in variable",
			source: `main = () {
    val x = !true
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			fnDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			if len(fnDecl.Body.Statements) != 1 {
				t.Fatalf("expected 1 statement in body, got %d", len(fnDecl.Body.Statements))
			}

			_, ok = fnDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatalf("expected VarDeclStmt, got %T", fnDecl.Body.Statements[0])
			}
		})
	}
}

func TestParserMultipleVariables(t *testing.T) {
	source := `main = () {
    val x = 5
    val y = 10
    val z = x + y
    print(z)
}`

	l := lexer.NewLexer([]byte(source))
	l.Parse()

	if len(l.Errors) > 0 {
		t.Fatalf("lexer errors: %v", l.Errors)
	}

	p := NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		t.Fatalf("parser errors: %v", p.Errors)
	}

	fnDecl := program.Declarations[0].(*ast.FunctionDecl)

	if len(fnDecl.Body.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(fnDecl.Body.Statements))
	}

	// Check first three are variable declarations
	expectedNames := []string{"x", "y", "z"}
	for i, name := range expectedNames {
		varDecl, ok := fnDecl.Body.Statements[i].(*ast.VarDeclStmt)
		if !ok {
			t.Fatalf("statement %d: expected VarDeclStmt, got %T", i, fnDecl.Body.Statements[i])
		}
		if varDecl.Name != name {
			t.Errorf("statement %d: expected name %q, got %q", i, name, varDecl.Name)
		}
	}

	// Check last is print() call
	exprStmt, ok := fnDecl.Body.Statements[3].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected ExprStmt for statement 3, got %T", fnDecl.Body.Statements[3])
	}
	callExpr, ok := exprStmt.Expr.(*ast.CallExpr)
	if !ok || callExpr.Name != "print" {
		t.Fatalf("expected print() call, got %T", exprStmt.Expr)
	}
}

func TestParserGroupingExpressions(t *testing.T) {
	tests := []struct {
		name   string
		tokens []lexer.Token
		check  func(t *testing.T, expr ast.Expression)
	}{
		{
			name: "simple grouping",
			// (5)
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
			},
			check: func(t *testing.T, expr ast.Expression) {
				group, ok := expr.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected GroupingExpr, got %T", expr)
				}
				lit, ok := group.Expr.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr inside grouping, got %T", group.Expr)
				}
				if lit.Value != "5" {
					t.Errorf("expected value 5, got %s", lit.Value)
				}
			},
		},
		{
			name: "grouped addition",
			// (2 + 3)
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 7, Offset: 6}},
			},
			check: func(t *testing.T, expr ast.Expression) {
				group, ok := expr.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected GroupingExpr, got %T", expr)
				}
				bin, ok := group.Expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr inside grouping, got %T", group.Expr)
				}
				if bin.Op != "+" {
					t.Errorf("expected operator +, got %s", bin.Op)
				}
			},
		},
		{
			name: "precedence override - grouped addition then multiply",
			// (2 + 3) * 4
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 7, Offset: 6}},
				{Type: lexer.TokenTypeMultiply, Value: "*", Pos: ast.Position{Line: 1, Column: 9, Offset: 8}},
				{Type: lexer.TokenTypeInteger, Value: "4", Pos: ast.Position{Line: 1, Column: 11, Offset: 10}},
			},
			check: func(t *testing.T, expr ast.Expression) {
				// Should be: BinaryExpr(GroupingExpr(2+3), *, 4)
				bin, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr at top level, got %T", expr)
				}
				if bin.Op != "*" {
					t.Errorf("expected top-level operator *, got %s", bin.Op)
				}
				group, ok := bin.Left.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected GroupingExpr on left, got %T", bin.Left)
				}
				innerBin, ok := group.Expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr inside grouping, got %T", group.Expr)
				}
				if innerBin.Op != "+" {
					t.Errorf("expected inner operator +, got %s", innerBin.Op)
				}
			},
		},
		{
			name: "nested grouping",
			// ((5))
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 3, Offset: 2}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 5, Offset: 4}},
			},
			check: func(t *testing.T, expr ast.Expression) {
				outer, ok := expr.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected outer GroupingExpr, got %T", expr)
				}
				inner, ok := outer.Expr.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected inner GroupingExpr, got %T", outer.Expr)
				}
				lit, ok := inner.Expr.(*ast.LiteralExpr)
				if !ok {
					t.Fatalf("expected LiteralExpr inside inner grouping, got %T", inner.Expr)
				}
				if lit.Value != "5" {
					t.Errorf("expected value 5, got %s", lit.Value)
				}
			},
		},
		{
			name: "complex expression with multiple groups",
			// (1 + 2) * (3 + 4)
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeInteger, Value: "1", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 4, Offset: 3}},
				{Type: lexer.TokenTypeInteger, Value: "2", Pos: ast.Position{Line: 1, Column: 6, Offset: 5}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 7, Offset: 6}},
				{Type: lexer.TokenTypeMultiply, Value: "*", Pos: ast.Position{Line: 1, Column: 9, Offset: 8}},
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 11, Offset: 10}},
				{Type: lexer.TokenTypeInteger, Value: "3", Pos: ast.Position{Line: 1, Column: 12, Offset: 11}},
				{Type: lexer.TokenTypePlus, Value: "+", Pos: ast.Position{Line: 1, Column: 14, Offset: 13}},
				{Type: lexer.TokenTypeInteger, Value: "4", Pos: ast.Position{Line: 1, Column: 16, Offset: 15}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 17, Offset: 16}},
			},
			check: func(t *testing.T, expr ast.Expression) {
				// Should be: BinaryExpr(GroupingExpr(1+2), *, GroupingExpr(3+4))
				bin, ok := expr.(*ast.BinaryExpr)
				if !ok {
					t.Fatalf("expected BinaryExpr at top level, got %T", expr)
				}
				if bin.Op != "*" {
					t.Errorf("expected top-level operator *, got %s", bin.Op)
				}
				leftGroup, ok := bin.Left.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected GroupingExpr on left, got %T", bin.Left)
				}
				rightGroup, ok := bin.Right.(*ast.GroupingExpr)
				if !ok {
					t.Fatalf("expected GroupingExpr on right, got %T", bin.Right)
				}
				leftBin, ok := leftGroup.Expr.(*ast.BinaryExpr)
				if !ok || leftBin.Op != "+" {
					t.Errorf("expected left inner + operator")
				}
				rightBin, ok := rightGroup.Expr.(*ast.BinaryExpr)
				if !ok || rightBin.Op != "+" {
					t.Errorf("expected right inner + operator")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			expr := p.ParseBinaryExpression()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", p.Errors)
			}

			if expr == nil {
				t.Fatal("expected expression, got nil")
			}

			tt.check(t, expr)
		})
	}
}

func TestParserGroupingErrors(t *testing.T) {
	tests := []struct {
		name   string
		tokens []lexer.Token
	}{
		{
			name: "unclosed grouping",
			// (5
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeInteger, Value: "5", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
			},
		},
		{
			name: "empty grouping",
			// ()
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeLParen, Value: "(", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
				{Type: lexer.TokenTypeRParen, Value: ")", Pos: ast.Position{Line: 1, Column: 2, Offset: 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.tokens)
			expr := p.ParseBinaryExpression()

			if expr != nil && len(p.Errors) == 0 {
				t.Fatal("expected error, got successful parse")
			}
		})
	}
}

func TestParserWhenStatement(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		caseCount int
	}{
		{
			name: "simple when with else",
			source: `main = () {
    when {
        true -> exit(0)
        else -> exit(1)
    }
}`,
			caseCount: 2,
		},
		{
			name: "when with multiple conditions",
			source: `main = () {
    val x = 5
    when {
        x > 10 -> exit(100)
        x > 5 -> exit(50)
        else -> exit(0)
    }
}`,
			caseCount: 3,
		},
		{
			name: "when with literal true (exhaustive without else)",
			source: `main = () {
    val x = 5
    when {
        x > 10 -> exit(100)
        true -> exit(0)
    }
}`,
			caseCount: 2,
		},
		{
			name: "when with block body",
			source: `main = () {
    when {
        true -> {
            val x = 1
            exit(x)
        }
        else -> exit(0)
    }
}`,
			caseCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			fnDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			// Find the when statement (may not be the first statement)
			var whenExpr *ast.WhenExpr
			for _, stmt := range fnDecl.Body.Statements {
				if we, ok := stmt.(*ast.WhenExpr); ok {
					whenExpr = we
					break
				}
			}

			if whenExpr == nil {
				t.Fatal("expected WhenExpr in function body")
			}

			if len(whenExpr.Cases) != tt.caseCount {
				t.Errorf("expected %d cases, got %d", tt.caseCount, len(whenExpr.Cases))
			}
		})
	}
}

func TestParserWhenExpression(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "when as expression in variable",
			source: `main = () {
    val x = when {
        true -> 42
        else -> 0
    }
}`,
		},
		{
			name: "when as expression in return",
			source: `foo = () -> s64 {
    return when {
        true -> 42
        else -> 0
    }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}
		})
	}
}

func TestParserWhenErrors(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		expectedError string
	}{
		{
			name: "when with subject syntax",
			source: `main = () {
    when (x) {
        true -> exit(0)
    }
}`,
			expectedError: "when (subject) { } syntax is not supported",
		},
		{
			name: "when missing brace",
			source: `main = () {
    when
        true -> exit(0)
    }
}`,
			expectedError: "expected '{' after when",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			_ = p.Parse()

			if len(p.Errors) == 0 {
				t.Fatal("expected parser error, got none")
			}

			found := false
			for _, err := range p.Errors {
				if strings.Contains(err.Error(), tt.expectedError) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got errors: %v", tt.expectedError, p.Errors)
			}
		})
	}
}

func TestParserWhileStatement(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		hasParens bool
	}{
		{
			name: "simple while without parens",
			source: `main = () {
    var i = 0
    while i < 5 {
        i = i + 1
    }
}`,
			hasParens: false,
		},
		{
			name: "simple while with parens",
			source: `main = () {
    var i = 0
    while (i < 5) {
        i = i + 1
    }
}`,
			hasParens: true,
		},
		{
			name: "while with break",
			source: `main = () {
    var i = 0
    while i < 10 {
        if i == 5 {
            break
        }
        i = i + 1
    }
}`,
			hasParens: false,
		},
		{
			name: "while with continue",
			source: `main = () {
    var i = 0
    while i < 10 {
        i = i + 1
        if i == 5 {
            continue
        }
    }
}`,
			hasParens: false,
		},
		{
			name: "nested while loops",
			source: `main = () {
    var i = 0
    while i < 3 {
        var j = 0
        while j < 2 {
            j = j + 1
        }
        i = i + 1
    }
}`,
			hasParens: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("lexer errors: %v", l.Errors)
			}

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("parser errors: %v", p.Errors)
			}

			if program == nil {
				t.Fatal("expected program, got nil")
			}

			fnDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			// Find the while statement (may not be the first statement)
			var whileStmt *ast.WhileStmt
			for _, stmt := range fnDecl.Body.Statements {
				if ws, ok := stmt.(*ast.WhileStmt); ok {
					whileStmt = ws
					break
				}
			}

			if whileStmt == nil {
				t.Fatal("expected WhileStmt in function body")
			}

			if whileStmt.HasParens != tt.hasParens {
				t.Errorf("expected HasParens=%v, got %v", tt.hasParens, whileStmt.HasParens)
			}

			if whileStmt.Condition == nil {
				t.Error("expected non-nil Condition")
			}

			if whileStmt.Body == nil {
				t.Error("expected non-nil Body")
			}
		})
	}
}

func TestParserWhileErrors(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		expectedError string
	}{
		{
			name: "while missing closing paren",
			source: `main = () {
    while (i < 5 {
        i = i + 1
    }
}`,
			expectedError: "expected ')'",
		},
		{
			name: "while missing body",
			source: `main = () {
    while i < 5
}`,
			expectedError: "expected '{'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			p.Parse()

			if len(p.Errors) == 0 {
				t.Fatal("expected parser errors")
			}

			found := false
			for _, err := range p.Errors {
				if strings.Contains(err.Message, tt.expectedError) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got errors: %v", tt.expectedError, p.Errors)
			}
		})
	}
}

func TestParserNullLiteral(t *testing.T) {
	t.Run("null literal parses correctly", func(t *testing.T) {
		tokens := []lexer.Token{
			{Type: lexer.TokenTypeNull, Value: "null", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
		}
		p := NewParser(tokens)
		literal := p.ParseLiteral()

		if literal == nil {
			t.Fatal("expected literal, got nil")
		}

		litExpr, ok := literal.(*ast.LiteralExpr)
		if !ok {
			t.Fatal("expected LiteralExpr")
		}

		if litExpr.Kind != ast.LiteralTypeNull {
			t.Errorf("expected LiteralTypeNull, got %d", litExpr.Kind)
		}

		if litExpr.Value != "null" {
			t.Errorf("expected value 'null', got %q", litExpr.Value)
		}
	})
}

func TestParserNullableType(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectedType string
	}{
		{
			name:         "nullable s64",
			source:       "main = () { val x: s64? = null }",
			expectedType: "s64?",
		},
		{
			name:         "nullable bool",
			source:       "main = () { val x: bool? = null }",
			expectedType: "bool?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			funcDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			if len(funcDecl.Body.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funcDecl.Body.Statements))
			}

			varDecl, ok := funcDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatal("expected VarDeclStmt")
			}

			if varDecl.TypeName != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, varDecl.TypeName)
			}
		})
	}
}

func TestParserSafeCall(t *testing.T) {
	t.Run("safe call expression parses correctly", func(t *testing.T) {
		source := `main = () { x?.field }`
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("unexpected parser errors: %v", p.Errors)
		}

		funcDecl := program.Declarations[0].(*ast.FunctionDecl)
		exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)
		safeCall, ok := exprStmt.Expr.(*ast.SafeCallExpr)
		if !ok {
			t.Fatalf("expected SafeCallExpr, got %T", exprStmt.Expr)
		}

		if safeCall.Field != "field" {
			t.Errorf("expected field name 'field', got %q", safeCall.Field)
		}

		ident, ok := safeCall.Object.(*ast.IdentifierExpr)
		if !ok {
			t.Fatalf("expected IdentifierExpr as object, got %T", safeCall.Object)
		}

		if ident.Name != "x" {
			t.Errorf("expected object name 'x', got %q", ident.Name)
		}
	})

	t.Run("chained safe calls", func(t *testing.T) {
		source := `main = () { a?.b?.c }`
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("unexpected parser errors: %v", p.Errors)
		}

		funcDecl := program.Declarations[0].(*ast.FunctionDecl)
		exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)
		outerSafeCall, ok := exprStmt.Expr.(*ast.SafeCallExpr)
		if !ok {
			t.Fatalf("expected SafeCallExpr, got %T", exprStmt.Expr)
		}

		if outerSafeCall.Field != "c" {
			t.Errorf("expected outer field 'c', got %q", outerSafeCall.Field)
		}

		innerSafeCall, ok := outerSafeCall.Object.(*ast.SafeCallExpr)
		if !ok {
			t.Fatalf("expected inner SafeCallExpr, got %T", outerSafeCall.Object)
		}

		if innerSafeCall.Field != "b" {
			t.Errorf("expected inner field 'b', got %q", innerSafeCall.Field)
		}
	})

	t.Run("safe method call", func(t *testing.T) {
		source := `main = () { x?.method() }`
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("unexpected parser errors: %v", p.Errors)
		}

		funcDecl := program.Declarations[0].(*ast.FunctionDecl)
		exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)
		methodCall, ok := exprStmt.Expr.(*ast.MethodCallExpr)
		if !ok {
			t.Fatalf("expected MethodCallExpr, got %T", exprStmt.Expr)
		}

		if methodCall.Method != "method" {
			t.Errorf("expected method name 'method', got %q", methodCall.Method)
		}

		if !methodCall.SafeNavigation {
			t.Error("expected SafeNavigation to be true")
		}

		ident, ok := methodCall.Object.(*ast.IdentifierExpr)
		if !ok {
			t.Fatalf("expected IdentifierExpr as object, got %T", methodCall.Object)
		}

		if ident.Name != "x" {
			t.Errorf("expected object name 'x', got %q", ident.Name)
		}
	})

	t.Run("safe method call with args", func(t *testing.T) {
		source := `main = () { x?.method(1, 2) }`
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("unexpected parser errors: %v", p.Errors)
		}

		funcDecl := program.Declarations[0].(*ast.FunctionDecl)
		exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)
		methodCall, ok := exprStmt.Expr.(*ast.MethodCallExpr)
		if !ok {
			t.Fatalf("expected MethodCallExpr, got %T", exprStmt.Expr)
		}

		if methodCall.Method != "method" {
			t.Errorf("expected method name 'method', got %q", methodCall.Method)
		}

		if !methodCall.SafeNavigation {
			t.Error("expected SafeNavigation to be true")
		}

		if len(methodCall.Arguments) != 2 {
			t.Errorf("expected 2 arguments, got %d", len(methodCall.Arguments))
		}
	})

	t.Run("regular method call has SafeNavigation false", func(t *testing.T) {
		source := `main = () { x.method() }`
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("unexpected parser errors: %v", p.Errors)
		}

		funcDecl := program.Declarations[0].(*ast.FunctionDecl)
		exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)
		methodCall, ok := exprStmt.Expr.(*ast.MethodCallExpr)
		if !ok {
			t.Fatalf("expected MethodCallExpr, got %T", exprStmt.Expr)
		}

		if methodCall.SafeNavigation {
			t.Error("expected SafeNavigation to be false for regular method call")
		}
	})
}

func TestParserNestedNullableError(t *testing.T) {
	source := `main = () { val x: s64?? = null }`
	l := lexer.NewLexer([]byte(source))
	l.Parse()

	p := NewParser(l.Tokens)
	p.Parse()

	if len(p.Errors) == 0 {
		t.Fatal("expected parser error for nested nullable")
	}

	found := false
	for _, err := range p.Errors {
		if strings.Contains(err.Message, "nested nullable") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error containing 'nested nullable', got: %v", p.Errors)
	}
}

// Tests for SEP-1: Pointer Type Parsing

func TestParserGenericTypes(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectedType string
	}{
		{
			name:         "Own<Point>",
			source:       "main = () { val p: Own<Point> = null }",
			expectedType: "Own<Point>",
		},
		{
			name:         "Ref<Point>",
			source:       "main = () { val p: Ref<Point> = null }",
			expectedType: "Ref<Point>",
		},
		{
			name:         "Own<Point>? nullable",
			source:       "main = () { val p: Own<Point>? = null }",
			expectedType: "Own<Point>?",
		},
		{
			name:         "nested generic Own<s64[]>",
			source:       "main = () { val p: Own<s64[]> = null }",
			expectedType: "Own<s64[]>",
		},
		{
			name:         "Own with nullable inner type",
			source:       "main = () { val p: Own<s64?> = null }",
			expectedType: "Own<s64?>",
		},
		{
			name:         "Ref<Point>? nullable ref",
			source:       "main = () { val p: Ref<Point>? = null }",
			expectedType: "Ref<Point>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			funcDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			if len(funcDecl.Body.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funcDecl.Body.Statements))
			}

			varDecl, ok := funcDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatal("expected VarDeclStmt")
			}

			if varDecl.TypeName != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, varDecl.TypeName)
			}
		})
	}
}

func TestParserArrayTypeSyntax(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectedType string
	}{
		{
			name:         "s64[] array type",
			source:       "main = () { val arr: s64[] = [1, 2, 3] }",
			expectedType: "s64[]",
		},
		{
			name:         "bool[] array type",
			source:       "main = () { val flags: bool[] = [true, false] }",
			expectedType: "bool[]",
		},
		{
			name:         "*Point[] array of pointers",
			source:       "main = () { val pts: *Point[] = null }",
			expectedType: "*Point[]",
		},
		{
			name:         "s64[]? nullable array",
			source:       "main = () { val arr: s64[]? = null }",
			expectedType: "s64[]?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			funcDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			varDecl, ok := funcDecl.Body.Statements[0].(*ast.VarDeclStmt)
			if !ok {
				t.Fatal("expected VarDeclStmt")
			}

			if varDecl.TypeName != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, varDecl.TypeName)
			}
		})
	}
}

func TestParserArrayLiteral(t *testing.T) {
	source := "main = () { [1, 2, 3] }"
	l := lexer.NewLexer([]byte(source))
	l.Parse()

	p := NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors)
	}

	funcDecl := program.Declarations[0].(*ast.FunctionDecl)
	exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)

	arrLit, ok := exprStmt.Expr.(*ast.ArrayLiteralExpr)
	if !ok {
		t.Fatalf("expected ArrayLiteralExpr, got %T", exprStmt.Expr)
	}

	if len(arrLit.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arrLit.Elements))
	}
}

func TestParserIndexExpr(t *testing.T) {
	source := "main = () { arr[0] }"
	l := lexer.NewLexer([]byte(source))
	l.Parse()

	p := NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors)
	}

	funcDecl := program.Declarations[0].(*ast.FunctionDecl)
	exprStmt := funcDecl.Body.Statements[0].(*ast.ExprStmt)

	indexExpr, ok := exprStmt.Expr.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("expected IndexExpr, got %T", exprStmt.Expr)
	}

	ident, ok := indexExpr.Array.(*ast.IdentifierExpr)
	if !ok {
		t.Fatalf("expected IdentifierExpr for array, got %T", indexExpr.Array)
	}
	if ident.Name != "arr" {
		t.Errorf("expected array name 'arr', got %q", ident.Name)
	}
}

func TestParserVarParameter(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		expectedMutable []bool
	}{
		{
			name:            "immutable ref parameter",
			source:          "foo = (p: Ref<Point>) { }",
			expectedMutable: []bool{false},
		},
		{
			name:            "mutable ref parameter with var",
			source:          "foo = (var p: Ref<Point>) { }",
			expectedMutable: []bool{true},
		},
		{
			name:            "mixed mutable and immutable",
			source:          "foo = (a: s64, var b: Ref<Point>, c: bool) { }",
			expectedMutable: []bool{false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			funcDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			if len(funcDecl.Parameters) != len(tt.expectedMutable) {
				t.Fatalf("expected %d parameters, got %d", len(tt.expectedMutable), len(funcDecl.Parameters))
			}

			for i, param := range funcDecl.Parameters {
				if param.Mutable != tt.expectedMutable[i] {
					t.Errorf("parameter %d: expected mutable=%v, got %v", i, tt.expectedMutable[i], param.Mutable)
				}
			}
		})
	}
}

func TestParserNewExpr(t *testing.T) {
	source := "main = () { new Point{ 1, 2 } }"
	l := lexer.NewLexer([]byte(source))
	l.Parse()

	p := NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors)
	}

	funcDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
	if !ok {
		t.Fatal("expected FunctionDecl")
	}

	exprStmt, ok := funcDecl.Body.Statements[0].(*ast.ExprStmt)
	if !ok {
		t.Fatal("expected ExprStmt")
	}

	newExpr, ok := exprStmt.Expr.(*ast.NewExpr)
	if !ok {
		t.Fatalf("expected NewExpr, got %T", exprStmt.Expr)
	}

	// The operand should be a struct literal
	_, ok = newExpr.Operand.(*ast.StructLiteral)
	if !ok {
		t.Fatalf("expected StructLiteral operand, got %T", newExpr.Operand)
	}
}

func TestParserMethodCall(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedObject string
		expectedMethod string
		expectedArgs   int
	}{
		{
			name:           "p.copy() no args",
			source:         "main = () { p.copy() }",
			expectedObject: "p",
			expectedMethod: "copy",
			expectedArgs:   0,
		},
		{
			name:           "method with multiple args",
			source:         "main = () { obj.method(a, b, c) }",
			expectedObject: "obj",
			expectedMethod: "method",
			expectedArgs:   3,
		},
		{
			name:           "chained field then method",
			source:         "main = () { a.b.method() }",
			expectedObject: "a.b", // a.b is the object
			expectedMethod: "method",
			expectedArgs:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			funcDecl, ok := program.Declarations[0].(*ast.FunctionDecl)
			if !ok {
				t.Fatal("expected FunctionDecl")
			}

			exprStmt, ok := funcDecl.Body.Statements[0].(*ast.ExprStmt)
			if !ok {
				t.Fatal("expected ExprStmt")
			}

			methodCall, ok := exprStmt.Expr.(*ast.MethodCallExpr)
			if !ok {
				t.Fatalf("expected MethodCallExpr, got %T", exprStmt.Expr)
			}

			if methodCall.Method != tt.expectedMethod {
				t.Errorf("expected method %q, got %q", tt.expectedMethod, methodCall.Method)
			}

			if len(methodCall.Arguments) != tt.expectedArgs {
				t.Errorf("expected %d arguments, got %d", tt.expectedArgs, len(methodCall.Arguments))
			}

			// Check object (simplified - just check if it's an identifier for simple cases)
			if tt.expectedObject != "a.b" {
				ident, ok := methodCall.Object.(*ast.IdentifierExpr)
				if !ok {
					t.Fatalf("expected IdentifierExpr as object, got %T", methodCall.Object)
				}
				if ident.Name != tt.expectedObject {
					t.Errorf("expected object %q, got %q", tt.expectedObject, ident.Name)
				}
			}
		})
	}
}

func TestParserClassDecl(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		expectedName    string
		expectedFields  int
		expectedMethods int
	}{
		{
			name:            "empty class",
			source:          "Point = class { }",
			expectedName:    "Point",
			expectedFields:  0,
			expectedMethods: 0,
		},
		{
			name: "class with one field",
			source: `Point = class {
				val x: s64
			}`,
			expectedName:    "Point",
			expectedFields:  1,
			expectedMethods: 0,
		},
		{
			name: "class with multiple fields",
			source: `Point = class {
				val x: s64
				var y: s64
			}`,
			expectedName:    "Point",
			expectedFields:  2,
			expectedMethods: 0,
		},
		{
			name: "class with one method",
			source: `Counter = class {
				getCount = (self: &Counter) -> s64 {
					self.count
				}
			}`,
			expectedName:    "Counter",
			expectedFields:  0,
			expectedMethods: 1,
		},
		{
			name: "class with fields and methods",
			source: `Counter = class {
				var count: s64
				increment = (self: &&Counter) {
					self.count = self.count + 1
				}
				getCount = (self: &Counter) -> s64 {
					self.count
				}
			}`,
			expectedName:    "Counter",
			expectedFields:  1,
			expectedMethods: 2,
		},
		{
			name: "class with static method",
			source: `Point = class {
				val x: s64
				val y: s64
				origin = () -> Point {
					Point{ 0, 0 }
				}
			}`,
			expectedName:    "Point",
			expectedFields:  2,
			expectedMethods: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			classDecl, ok := program.Declarations[0].(*ast.ClassDecl)
			if !ok {
				t.Fatalf("expected ClassDecl, got %T", program.Declarations[0])
			}

			if classDecl.Name != tt.expectedName {
				t.Errorf("expected class name %q, got %q", tt.expectedName, classDecl.Name)
			}

			if len(classDecl.Fields) != tt.expectedFields {
				t.Errorf("expected %d fields, got %d", tt.expectedFields, len(classDecl.Fields))
			}

			if len(classDecl.Methods) != tt.expectedMethods {
				t.Errorf("expected %d methods, got %d", tt.expectedMethods, len(classDecl.Methods))
			}
		})
	}
}

func TestParserObjectDecl(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		expectedName    string
		expectedMethods int
	}{
		{
			name:            "empty object",
			source:          "Math = object { }",
			expectedName:    "Math",
			expectedMethods: 0,
		},
		{
			name: "object with one method",
			source: `Math = object {
				max = (a: s64, b: s64) -> s64 {
					if a > b { a } else { b }
				}
			}`,
			expectedName:    "Math",
			expectedMethods: 1,
		},
		{
			name: "object with multiple methods",
			source: `Math = object {
				max = (a: s64, b: s64) -> s64 {
					if a > b { a } else { b }
				}
				min = (a: s64, b: s64) -> s64 {
					if a < b { a } else { b }
				}
			}`,
			expectedName:    "Math",
			expectedMethods: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
			}

			objectDecl, ok := program.Declarations[0].(*ast.ObjectDecl)
			if !ok {
				t.Fatalf("expected ObjectDecl, got %T", program.Declarations[0])
			}

			if objectDecl.Name != tt.expectedName {
				t.Errorf("expected object name %q, got %q", tt.expectedName, objectDecl.Name)
			}

			if len(objectDecl.Methods) != tt.expectedMethods {
				t.Errorf("expected %d methods, got %d", tt.expectedMethods, len(objectDecl.Methods))
			}
		})
	}
}

func TestParserSelfExpr(t *testing.T) {
	source := `Counter = class {
		var count: s64
		getCount = (self: &Counter) -> s64 {
			self.count
		}
	}`

	l := lexer.NewLexer([]byte(source))
	l.Parse()

	p := NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors)
	}

	classDecl, ok := program.Declarations[0].(*ast.ClassDecl)
	if !ok {
		t.Fatal("expected ClassDecl")
	}

	if len(classDecl.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(classDecl.Methods))
	}

	method := classDecl.Methods[0]
	if method.Name != "getCount" {
		t.Errorf("expected method name 'getCount', got %q", method.Name)
	}

	// Check that the method has 'self' as first parameter
	if len(method.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(method.Parameters))
	}

	if method.Parameters[0].Name != "self" {
		t.Errorf("expected parameter name 'self', got %q", method.Parameters[0].Name)
	}

	// Check IsInstance() helper
	if !method.IsInstance() {
		t.Error("expected method.IsInstance() to be true")
	}

	// Check the body contains a field access on self
	if len(method.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in body, got %d", len(method.Body.Statements))
	}

	exprStmt, ok := method.Body.Statements[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected ExprStmt, got %T", method.Body.Statements[0])
	}

	fieldAccess, ok := exprStmt.Expr.(*ast.FieldAccessExpr)
	if !ok {
		t.Fatalf("expected FieldAccessExpr, got %T", exprStmt.Expr)
	}

	selfExpr, ok := fieldAccess.Object.(*ast.SelfExpr)
	if !ok {
		t.Fatalf("expected SelfExpr, got %T", fieldAccess.Object)
	}

	// Verify position is set
	if selfExpr.SelfPos.Line == 0 {
		t.Error("expected SelfExpr position to be set")
	}

	if fieldAccess.Field != "count" {
		t.Errorf("expected field 'count', got %q", fieldAccess.Field)
	}
}

func TestParserClassMethodIsInstance(t *testing.T) {
	tests := []struct {
		name             string
		source           string
		expectedInstance bool
	}{
		{
			name: "instance method with self",
			source: `Point = class {
				getX = (self: &Point) -> s64 { 0 }
			}`,
			expectedInstance: true,
		},
		{
			name: "static method without self",
			source: `Point = class {
				origin = () -> Point { Point{ 0, 0 } }
			}`,
			expectedInstance: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			p := NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors)
			}

			classDecl := program.Declarations[0].(*ast.ClassDecl)
			method := classDecl.Methods[0]

			if method.IsInstance() != tt.expectedInstance {
				t.Errorf("expected IsInstance()=%v, got %v", tt.expectedInstance, method.IsInstance())
			}
		})
	}
}

func TestParserObjectFieldError(t *testing.T) {
	source := `Math = object {
		val x: s64
	}`

	l := lexer.NewLexer([]byte(source))
	l.Parse()

	p := NewParser(l.Tokens)
	p.Parse()

	if len(p.Errors) == 0 {
		t.Fatal("expected parser error for field in object")
	}

	// Check that error mentions fields not allowed
	found := false
	for _, err := range p.Errors {
		if err.Error() == "objects cannot have fields, only methods" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected error about objects not having fields, got: %v", p.Errors)
	}
}

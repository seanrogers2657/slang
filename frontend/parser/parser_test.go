package parser

import (
	"testing"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/lexer"
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
				Kind:  ast.LiteralTypeNumber,
				Value: "5",
			},
		},
		{
			name: "multiple digits",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "123", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeNumber,
				Value: "123",
			},
		},
		{
			name: "zero",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "0", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}},
			},
			expected: &ast.LiteralExpr{
				Kind:  ast.LiteralTypeNumber,
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
			source:       "fn main() {}",
			expectedName: "main",
			expectedBody: 0,
		},
		{
			name:         "main function with single statement",
			source:       "fn main() {\n    print 42\n}",
			expectedName: "main",
			expectedBody: 1,
		},
		{
			name:         "main function with multiple statements",
			source:       "fn main() {\n    print 1\n    print 2\n}",
			expectedName: "main",
			expectedBody: 2,
		},
		{
			name:         "function with expression statement",
			source:       "fn main() {\n    5 + 3\n}",
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
			source:  "fn main() {\n    val x = 5\n}",
			varName: "x",
		},
		{
			name:    "variable with expression",
			source:  "fn main() {\n    val result = 10 + 20\n}",
			varName: "result",
		},
		{
			name:    "variable with underscore name",
			source:  "fn main() {\n    val my_var = 42\n}",
			varName: "my_var",
		},
		{
			name:    "variable with digits in name",
			source:  "fn main() {\n    val x1 = 100\n}",
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
		name        string
		source      string
		varName     string
		mutable     bool
	}{
		{
			name:    "immutable variable with val",
			source:  "fn main() {\n    val x = 5\n}",
			varName: "x",
			mutable: false,
		},
		{
			name:    "mutable variable with var",
			source:  "fn main() {\n    var x = 5\n}",
			varName: "x",
			mutable: true,
		},
		{
			name:    "mutable variable with expression",
			source:  "fn main() {\n    var result = 10 + 20\n}",
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
		name     string
		source   string
		varName  string
	}{
		{
			name:    "simple assignment",
			source:  "fn main() {\n    var x = 5\n    x = 10\n}",
			varName: "x",
		},
		{
			name:    "assignment with expression",
			source:  "fn main() {\n    var x = 5\n    x = x + 10\n}",
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
			source:       "fn main() {\n    print x\n}",
			expectedName: "x",
		},
		{
			name:         "identifier with underscore",
			source:       "fn main() {\n    print my_var\n}",
			expectedName: "my_var",
		},
		{
			name:         "identifier in binary expression",
			source:       "fn main() {\n    val x = 5\n    print x + 10\n}",
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
				if printStmt, ok := stmt.(*ast.PrintStmt); ok {
					// Check if the print expression contains an identifier
					switch e := printStmt.Expr.(type) {
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

			if !foundIdent {
				t.Errorf("expected to find identifier %q", tt.expectedName)
			}
		})
	}
}

func TestParserMultipleVariables(t *testing.T) {
	source := `fn main() {
    val x = 5
    val y = 10
    val z = x + y
    print z
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

	// Check last is print statement
	_, ok := fnDecl.Body.Statements[3].(*ast.PrintStmt)
	if !ok {
		t.Fatalf("expected PrintStmt for statement 3, got %T", fnDecl.Body.Statements[3])
	}
}

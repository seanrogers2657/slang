package parser

import (
	"testing"

	"github.com/seanrogers2657/slang/frontend/lexer"
)

func TestParserLiterals(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *Literal
	}{
		{
			name: "single digit",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5"},
			},
			expected: &Literal{
				Type:  LiteralTypeNumber,
				Value: "5",
			},
		},
		{
			name: "multiple digits",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "123"},
			},
			expected: &Literal{
				Type:  LiteralTypeNumber,
				Value: "123",
			},
		},
		{
			name: "zero",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "0"},
			},
			expected: &Literal{
				Type:  LiteralTypeNumber,
				Value: "0",
			},
		},
		{
			name: "simple string",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeString, Value: "hello"},
			},
			expected: &Literal{
				Type:  LiteralTypeString,
				Value: "hello",
			},
		},
		{
			name: "empty string",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeString, Value: ""},
			},
			expected: &Literal{
				Type:  LiteralTypeString,
				Value: "",
			},
		},
		{
			name: "string with spaces",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeString, Value: "hello world"},
			},
			expected: &Literal{
				Type:  LiteralTypeString,
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

			if literal.Type != tt.expected.Type {
				t.Errorf("expected type %d, got %d", tt.expected.Type, literal.Type)
			}

			if literal.Value != tt.expected.Value {
				t.Errorf("expected value %q, got %q", tt.expected.Value, literal.Value)
			}
		})
	}
}

func TestParserBinaryExpressions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *Expr
	}{
		{
			name: "addition",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2"},
				{Type: lexer.TokenTypePlus, Value: "+"},
				{Type: lexer.TokenTypeInteger, Value: "5"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
				Op:    "+",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
			},
		},
		{
			name: "subtraction",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "10"},
				{Type: lexer.TokenTypeMinus, Value: "-"},
				{Type: lexer.TokenTypeInteger, Value: "3"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "10"},
				Op:    "-",
				Right: &Literal{Type: LiteralTypeNumber, Value: "3"},
			},
		},
		{
			name: "multiplication",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "4"},
				{Type: lexer.TokenTypeMultiply, Value: "*"},
				{Type: lexer.TokenTypeInteger, Value: "7"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "4"},
				Op:    "*",
				Right: &Literal{Type: LiteralTypeNumber, Value: "7"},
			},
		},
		{
			name: "division",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "20"},
				{Type: lexer.TokenTypeDivide, Value: "/"},
				{Type: lexer.TokenTypeInteger, Value: "4"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "20"},
				Op:    "/",
				Right: &Literal{Type: LiteralTypeNumber, Value: "4"},
			},
		},
		{
			name: "modulo",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "10"},
				{Type: lexer.TokenTypeModulo, Value: "%"},
				{Type: lexer.TokenTypeInteger, Value: "3"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "10"},
				Op:    "%",
				Right: &Literal{Type: LiteralTypeNumber, Value: "3"},
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

			if expr.Op != tt.expected.Op {
				t.Errorf("expected operator %q, got %q", tt.expected.Op, expr.Op)
			}

			if expr.Left == nil || expr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}

			if expr.Left.Value != tt.expected.Left.Value {
				t.Errorf("expected left value %q, got %q", tt.expected.Left.Value, expr.Left.Value)
			}

			if expr.Right.Value != tt.expected.Right.Value {
				t.Errorf("expected right value %q, got %q", tt.expected.Right.Value, expr.Right.Value)
			}
		})
	}
}

func TestParserComparisonExpressions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *Expr
	}{
		{
			name: "equality",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypeEqual, Value: "=="},
				{Type: lexer.TokenTypeInteger, Value: "5"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "5"},
				Op:    "==",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
			},
		},
		{
			name: "inequality",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "3"},
				{Type: lexer.TokenTypeNotEqual, Value: "!="},
				{Type: lexer.TokenTypeInteger, Value: "4"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "3"},
				Op:    "!=",
				Right: &Literal{Type: LiteralTypeNumber, Value: "4"},
			},
		},
		{
			name: "less than",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2"},
				{Type: lexer.TokenTypeLessThan, Value: "<"},
				{Type: lexer.TokenTypeInteger, Value: "8"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
				Op:    "<",
				Right: &Literal{Type: LiteralTypeNumber, Value: "8"},
			},
		},
		{
			name: "greater than",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "9"},
				{Type: lexer.TokenTypeGreaterThan, Value: ">"},
				{Type: lexer.TokenTypeInteger, Value: "1"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "9"},
				Op:    ">",
				Right: &Literal{Type: LiteralTypeNumber, Value: "1"},
			},
		},
		{
			name: "less than or equal",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypeLessThanOrEqual, Value: "<="},
				{Type: lexer.TokenTypeInteger, Value: "5"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "5"},
				Op:    "<=",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
			},
		},
		{
			name: "greater than or equal",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "7"},
				{Type: lexer.TokenTypeGreaterThanOrEqual, Value: ">="},
				{Type: lexer.TokenTypeInteger, Value: "7"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "7"},
				Op:    ">=",
				Right: &Literal{Type: LiteralTypeNumber, Value: "7"},
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

			if expr.Op != tt.expected.Op {
				t.Errorf("expected operator %q, got %q", tt.expected.Op, expr.Op)
			}

			if expr.Left == nil || expr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}

			if expr.Left.Value != tt.expected.Left.Value {
				t.Errorf("expected left value %q, got %q", tt.expected.Left.Value, expr.Left.Value)
			}

			if expr.Right.Value != tt.expected.Right.Value {
				t.Errorf("expected right value %q, got %q", tt.expected.Right.Value, expr.Right.Value)
			}
		})
	}
}

func TestParserParse(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []lexer.Token
		expected *Expr
	}{
		{
			name: "simple addition expression",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2"},
				{Type: lexer.TokenTypePlus, Value: "+"},
				{Type: lexer.TokenTypeInteger, Value: "5"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
				Op:    "+",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
			},
		},
		{
			name: "with newline",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2"},
				{Type: lexer.TokenTypePlus, Value: "+"},
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
			},
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
				Op:    "+",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
			},
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

			expr := program.Statements[0]

			if expr.Op != tt.expected.Op {
				t.Errorf("expected operator %q, got %q", tt.expected.Op, expr.Op)
			}

			if expr.Left == nil || expr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}

			if expr.Left.Value != tt.expected.Left.Value {
				t.Errorf("expected left value %q, got %q", tt.expected.Left.Value, expr.Left.Value)
			}

			if expr.Right.Value != tt.expected.Right.Value {
				t.Errorf("expected right value %q, got %q", tt.expected.Right.Value, expr.Right.Value)
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
			name: "unsupported operation - newline",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
			},
			expectedError: "unsupported operation: \n",
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
		name      string
		tokens    []lexer.Token
		numStmts  int
		stmts     []*Expr
	}{
		{
			name: "two statements",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "2"},
				{Type: lexer.TokenTypePlus, Value: "+"},
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
				{Type: lexer.TokenTypeInteger, Value: "10"},
				{Type: lexer.TokenTypeMinus, Value: "-"},
				{Type: lexer.TokenTypeInteger, Value: "3"},
			},
			numStmts: 2,
			stmts: []*Expr{
				{
					Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
					Op:    "+",
					Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
				},
				{
					Left:  &Literal{Type: LiteralTypeNumber, Value: "10"},
					Op:    "-",
					Right: &Literal{Type: LiteralTypeNumber, Value: "3"},
				},
			},
		},
		{
			name: "three statements with multiple newlines",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeInteger, Value: "1"},
				{Type: lexer.TokenTypePlus, Value: "+"},
				{Type: lexer.TokenTypeInteger, Value: "1"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
				{Type: lexer.TokenTypeInteger, Value: "2"},
				{Type: lexer.TokenTypeMultiply, Value: "*"},
				{Type: lexer.TokenTypeInteger, Value: "3"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
				{Type: lexer.TokenTypeInteger, Value: "4"},
				{Type: lexer.TokenTypeDivide, Value: "/"},
				{Type: lexer.TokenTypeInteger, Value: "2"},
			},
			numStmts: 3,
			stmts: []*Expr{
				{
					Left:  &Literal{Type: LiteralTypeNumber, Value: "1"},
					Op:    "+",
					Right: &Literal{Type: LiteralTypeNumber, Value: "1"},
				},
				{
					Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
					Op:    "*",
					Right: &Literal{Type: LiteralTypeNumber, Value: "3"},
				},
				{
					Left:  &Literal{Type: LiteralTypeNumber, Value: "4"},
					Op:    "/",
					Right: &Literal{Type: LiteralTypeNumber, Value: "2"},
				},
			},
		},
		{
			name: "leading and trailing newlines",
			tokens: []lexer.Token{
				{Type: lexer.TokenTypeNewline, Value: "\n"},
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypePlus, Value: "+"},
				{Type: lexer.TokenTypeInteger, Value: "5"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
				{Type: lexer.TokenTypeNewline, Value: "\n"},
			},
			numStmts: 1,
			stmts: []*Expr{
				{
					Left:  &Literal{Type: LiteralTypeNumber, Value: "5"},
					Op:    "+",
					Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
				},
			},
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

			for i, expectedStmt := range tt.stmts {
				stmt := program.Statements[i]

				if stmt.Op != expectedStmt.Op {
					t.Errorf("statement %d: expected operator %q, got %q", i, expectedStmt.Op, stmt.Op)
				}

				if stmt.Left.Value != expectedStmt.Left.Value {
					t.Errorf("statement %d: expected left value %q, got %q", i, expectedStmt.Left.Value, stmt.Left.Value)
				}

				if stmt.Right.Value != expectedStmt.Right.Value {
					t.Errorf("statement %d: expected right value %q, got %q", i, expectedStmt.Right.Value, stmt.Right.Value)
				}
			}
		})
	}
}

func TestParserIntegrationWithLexer(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected *Expr
	}{
		{
			name:   "simple addition",
			source: "2 + 5",
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "2"},
				Op:    "+",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
			},
		},
		{
			name:   "subtraction",
			source: "10 - 3",
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "10"},
				Op:    "-",
				Right: &Literal{Type: LiteralTypeNumber, Value: "3"},
			},
		},
		{
			name:   "multiplication",
			source: "4 * 7",
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "4"},
				Op:    "*",
				Right: &Literal{Type: LiteralTypeNumber, Value: "7"},
			},
		},
		{
			name:   "division",
			source: "20 / 4",
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "20"},
				Op:    "/",
				Right: &Literal{Type: LiteralTypeNumber, Value: "4"},
			},
		},
		{
			name:   "comparison",
			source: "5 == 5",
			expected: &Expr{
				Left:  &Literal{Type: LiteralTypeNumber, Value: "5"},
				Op:    "==",
				Right: &Literal{Type: LiteralTypeNumber, Value: "5"},
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

			expr := program.Statements[0]

			if expr.Op != tt.expected.Op {
				t.Errorf("expected operator %q, got %q", tt.expected.Op, expr.Op)
			}

			if expr.Left == nil || expr.Right == nil {
				t.Fatal("expected left and right operands, got nil")
			}

			if expr.Left.Value != tt.expected.Left.Value {
				t.Errorf("expected left value %q, got %q", tt.expected.Left.Value, expr.Left.Value)
			}

			if expr.Right.Value != tt.expected.Right.Value {
				t.Errorf("expected right value %q, got %q", tt.expected.Right.Value, expr.Right.Value)
			}
		})
	}
}

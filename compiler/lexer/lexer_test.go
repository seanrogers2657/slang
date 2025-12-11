package lexer

import (
	"testing"
)

func TestLexerNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "single digit",
			input: "5",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "multiple digits",
			input: "123",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "123"},
			},
		},
		{
			name:  "large number",
			input: "999999",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "999999"},
			},
		},
		{
			name:  "zero",
			input: "0",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(l.Tokens))
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %d, got %d", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerArithmeticOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "addition",
			input: "+",
			expected: []Token{
				{Type: TokenTypePlus, Value: "+"},
			},
		},
		{
			name:  "subtraction",
			input: "-",
			expected: []Token{
				{Type: TokenTypeMinus, Value: "-"},
			},
		},
		{
			name:  "multiplication",
			input: "*",
			expected: []Token{
				{Type: TokenTypeMultiply, Value: "*"},
			},
		},
		{
			name:  "division",
			input: "/",
			expected: []Token{
				{Type: TokenTypeDivide, Value: "/"},
			},
		},
		{
			name:  "modulo",
			input: "%",
			expected: []Token{
				{Type: TokenTypeModulo, Value: "%"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(l.Tokens))
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %d, got %d", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "equality",
			input: "==",
			expected: []Token{
				{Type: TokenTypeEqual, Value: "=="},
			},
		},
		{
			name:  "inequality",
			input: "!=",
			expected: []Token{
				{Type: TokenTypeNotEqual, Value: "!="},
			},
		},
		{
			name:  "less than",
			input: "<",
			expected: []Token{
				{Type: TokenTypeLessThan, Value: "<"},
			},
		},
		{
			name:  "greater than",
			input: ">",
			expected: []Token{
				{Type: TokenTypeGreaterThan, Value: ">"},
			},
		},
		{
			name:  "less than or equal",
			input: "<=",
			expected: []Token{
				{Type: TokenTypeLessThanOrEqual, Value: "<="},
			},
		},
		{
			name:  "greater than or equal",
			input: ">=",
			expected: []Token{
				{Type: TokenTypeGreaterThanOrEqual, Value: ">="},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(l.Tokens))
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %d, got %d", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple addition",
			input: "2 + 5",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "simple subtraction",
			input: "10 - 3",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "10"},
				{Type: TokenTypeMinus, Value: "-"},
				{Type: TokenTypeInteger, Value: "3"},
			},
		},
		{
			name:  "multiplication",
			input: "4 * 7",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "4"},
				{Type: TokenTypeMultiply, Value: "*"},
				{Type: TokenTypeInteger, Value: "7"},
			},
		},
		{
			name:  "division",
			input: "20 / 4",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "20"},
				{Type: TokenTypeDivide, Value: "/"},
				{Type: TokenTypeInteger, Value: "4"},
			},
		},
		{
			name:  "modulo",
			input: "10 % 3",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "10"},
				{Type: TokenTypeModulo, Value: "%"},
				{Type: TokenTypeInteger, Value: "3"},
			},
		},
		{
			name:  "complex expression",
			input: "2 + 3 * 4",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "3"},
				{Type: TokenTypeMultiply, Value: "*"},
				{Type: TokenTypeInteger, Value: "4"},
			},
		},
		{
			name:  "comparison expression",
			input: "5 == 5",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "5"},
				{Type: TokenTypeEqual, Value: "=="},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "inequality comparison",
			input: "3 != 4",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "3"},
				{Type: TokenTypeNotEqual, Value: "!="},
				{Type: TokenTypeInteger, Value: "4"},
			},
		},
		{
			name:  "no spaces",
			input: "2+5",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "with newline",
			input: "2 + 5\n",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "5"},
				{Type: TokenTypeNewline, Value: "\n"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(l.Tokens))
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %d, got %d", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "unexpected character",
			input:         "2 @ 5",
			expectedError: "unexpected character: '@'",
		},
		{
			name:          "single ampersand",
			input:         "a & b",
			expectedError: "unexpected character: '&' (bitwise & not supported, use && for logical AND)",
		},
		{
			name:          "single pipe",
			input:         "a | b",
			expectedError: "unexpected character: '|' (bitwise | not supported, use || for logical OR)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) == 0 {
				t.Fatalf("expected error, got none")
			}

			if l.Errors[0].Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, l.Errors[0].Error())
			}
		})
	}
}

func TestLexerWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "multiple spaces",
			input: "2    +    5",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "tabs",
			input: "2\t+\t5",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "mixed whitespace",
			input: "  2  \t + \t  5  ",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "2"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(l.Tokens))
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %d, got %d", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple string",
			input: "\"hello\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello"},
			},
		},
		{
			name:  "empty string",
			input: "\"\"",
			expected: []Token{
				{Type: TokenTypeString, Value: ""},
			},
		},
		{
			name:  "string with spaces",
			input: "\"hello world\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello world"},
			},
		},
		{
			name:  "string with escape sequences",
			input: "\"hello\nworld\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello\nworld"},
			},
		},
		{
			name:  "string with tab escape",
			input: "\"hello\tworld\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello\tworld"},
			},
		},
		{
			name:  "string with escaped quote",
			input: "\"hello \\\"world\\\"\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello \"world\""},
			},
		},
		{
			name:  "string with escaped backslash",
			input: "\"hello\\world\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello\\world"},
			},
		},
		{
			name:  "multiple strings",
			input: "\"hello\" \"world\"",
			expected: []Token{
				{Type: TokenTypeString, Value: "hello"},
				{Type: TokenTypeString, Value: "world"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(l.Tokens))
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %d, got %d", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerStringErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "unterminated string",
			input:         "\"hello",
			expectedError: "unterminated string literal",
		},
		{
			name:          "unterminated string with newline",
			input:         "\"hello\n",
			expectedError: "unterminated string literal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) == 0 {
				t.Fatalf("expected error, got none")
			}

			if l.Errors[0].Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, l.Errors[0].Error())
			}
		})
	}
}

func TestLexerFunctionDeclaration(t *testing.T) {
	input := "fn main() { }"
	expected := []Token{
		{Type: TokenTypeFn, Value: "fn"},
		{Type: TokenTypeIdentifier, Value: "main"},
		{Type: TokenTypeLParen, Value: "("},
		{Type: TokenTypeRParen, Value: ")"},
		{Type: TokenTypeLBrace, Value: "{"},
		{Type: TokenTypeRBrace, Value: "}"},
	}

	l := NewLexer([]byte(input))
	l.Parse()

	if len(l.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", l.Errors)
	}

	if len(l.Tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(l.Tokens))
	}

	for i, token := range l.Tokens {
		if token.Type != expected[i].Type {
			t.Errorf("token %d: expected type %v, got %v", i, expected[i].Type, token.Type)
		}
		if token.Value != expected[i].Value {
			t.Errorf("token %d: expected value %q, got %q", i, expected[i].Value, token.Value)
		}
	}
}

func TestLexerVariableDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "val keyword",
			input: "val",
			expected: []Token{
				{Type: TokenTypeVal, Value: "val"},
			},
		},
		{
			name:  "var keyword",
			input: "var",
			expected: []Token{
				{Type: TokenTypeVar, Value: "var"},
			},
		},
		{
			name:  "assignment operator",
			input: "=",
			expected: []Token{
				{Type: TokenTypeAssign, Value: "="},
			},
		},
		{
			name:  "simple variable declaration",
			input: "val x = 5",
			expected: []Token{
				{Type: TokenTypeVal, Value: "val"},
				{Type: TokenTypeIdentifier, Value: "x"},
				{Type: TokenTypeAssign, Value: "="},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "simple mutable variable declaration",
			input: "var x = 5",
			expected: []Token{
				{Type: TokenTypeVar, Value: "var"},
				{Type: TokenTypeIdentifier, Value: "x"},
				{Type: TokenTypeAssign, Value: "="},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "variable declaration with expression",
			input: "val result = 10 + 20",
			expected: []Token{
				{Type: TokenTypeVal, Value: "val"},
				{Type: TokenTypeIdentifier, Value: "result"},
				{Type: TokenTypeAssign, Value: "="},
				{Type: TokenTypeInteger, Value: "10"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "20"},
			},
		},
		{
			name:  "mutable variable declaration with expression",
			input: "var counter = 10 + 20",
			expected: []Token{
				{Type: TokenTypeVar, Value: "var"},
				{Type: TokenTypeIdentifier, Value: "counter"},
				{Type: TokenTypeAssign, Value: "="},
				{Type: TokenTypeInteger, Value: "10"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "20"},
			},
		},
		{
			name:  "identifier with underscore",
			input: "my_var",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "my_var"},
			},
		},
		{
			name:  "identifier with digits",
			input: "var123",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "var123"},
			},
		},
		{
			name:  "identifier with mixed",
			input: "my_var_123",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "my_var_123"},
			},
		},
		{
			name:  "equality vs assignment",
			input: "x = 5 == 5",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "x"},
				{Type: TokenTypeAssign, Value: "="},
				{Type: TokenTypeInteger, Value: "5"},
				{Type: TokenTypeEqual, Value: "=="},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(l.Tokens), l.Tokens)
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %v, got %v", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerFloatLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple float",
			input: "3.14",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "3.14"},
			},
		},
		{
			name:  "float with many decimals",
			input: "3.14159265",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "3.14159265"},
			},
		},
		{
			name:  "float starting with zero",
			input: "0.5",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "0.5"},
			},
		},
		{
			name:  "scientific notation lowercase",
			input: "1e10",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "1e10"},
			},
		},
		{
			name:  "scientific notation uppercase",
			input: "1E10",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "1E10"},
			},
		},
		{
			name:  "scientific with positive exponent",
			input: "1.5e+10",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "1.5e+10"},
			},
		},
		{
			name:  "scientific with negative exponent",
			input: "2.5e-3",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "2.5e-3"},
			},
		},
		{
			name:  "float in expression",
			input: "3.14 + 2.0",
			expected: []Token{
				{Type: TokenTypeFloat, Value: "3.14"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeFloat, Value: "2.0"},
			},
		},
		{
			name:  "integer not confused with float",
			input: "42",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "42"},
			},
		},
		{
			name:  "mixed int and float",
			input: "42 + 3.14",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "42"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeFloat, Value: "3.14"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(l.Tokens), l.Tokens)
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %v, got %v", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerFloatErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "exponent without digits",
			input:         "1e",
			expectedError: "invalid float literal: exponent has no digits",
		},
		{
			name:          "exponent with sign but no digits",
			input:         "1e+",
			expectedError: "invalid float literal: exponent has no digits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) == 0 {
				t.Fatalf("expected error, got none")
			}

			if l.Errors[0].Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, l.Errors[0].Error())
			}
		})
	}
}

func TestLexerComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:     "line comment only",
			input:    "// this is a comment",
			expected: []Token{},
		},
		{
			name:  "line comment after code",
			input: "5 + 3 // add numbers",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "5"},
				{Type: TokenTypePlus, Value: "+"},
				{Type: TokenTypeInteger, Value: "3"},
			},
		},
		{
			name:  "line comment with newline",
			input: "// comment\n5",
			expected: []Token{
				{Type: TokenTypeNewline, Value: "\n"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "multiple comments",
			input: "// first\n// second\n5",
			expected: []Token{
				{Type: TokenTypeNewline, Value: "\n"},
				{Type: TokenTypeNewline, Value: "\n"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "comment with @test directive",
			input: "// @test: exit_code=0\nfn main() { }",
			expected: []Token{
				{Type: TokenTypeNewline, Value: "\n"},
				{Type: TokenTypeFn, Value: "fn"},
				{Type: TokenTypeIdentifier, Value: "main"},
				{Type: TokenTypeLParen, Value: "("},
				{Type: TokenTypeRParen, Value: ")"},
				{Type: TokenTypeLBrace, Value: "{"},
				{Type: TokenTypeRBrace, Value: "}"},
			},
		},
		{
			name:  "division not confused with comment",
			input: "10 / 2",
			expected: []Token{
				{Type: TokenTypeInteger, Value: "10"},
				{Type: TokenTypeDivide, Value: "/"},
				{Type: TokenTypeInteger, Value: "2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(l.Tokens), l.Tokens)
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %v, got %v", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

func TestLexerBooleanTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "true keyword",
			input: "true",
			expected: []Token{
				{Type: TokenTypeTrue, Value: "true"},
			},
		},
		{
			name:  "false keyword",
			input: "false",
			expected: []Token{
				{Type: TokenTypeFalse, Value: "false"},
			},
		},
		{
			name:  "logical AND",
			input: "&&",
			expected: []Token{
				{Type: TokenTypeAnd, Value: "&&"},
			},
		},
		{
			name:  "logical OR",
			input: "||",
			expected: []Token{
				{Type: TokenTypeOr, Value: "||"},
			},
		},
		{
			name:  "logical NOT",
			input: "!",
			expected: []Token{
				{Type: TokenTypeNot, Value: "!"},
			},
		},
		{
			name:  "not equal unchanged",
			input: "!=",
			expected: []Token{
				{Type: TokenTypeNotEqual, Value: "!="},
			},
		},
		{
			name:  "not followed by true",
			input: "!true",
			expected: []Token{
				{Type: TokenTypeNot, Value: "!"},
				{Type: TokenTypeTrue, Value: "true"},
			},
		},
		{
			name:  "not followed by false",
			input: "!false",
			expected: []Token{
				{Type: TokenTypeNot, Value: "!"},
				{Type: TokenTypeFalse, Value: "false"},
			},
		},
		{
			name:  "not followed by number",
			input: "!5",
			expected: []Token{
				{Type: TokenTypeNot, Value: "!"},
				{Type: TokenTypeInteger, Value: "5"},
			},
		},
		{
			name:  "logical AND expression",
			input: "a && b",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "a"},
				{Type: TokenTypeAnd, Value: "&&"},
				{Type: TokenTypeIdentifier, Value: "b"},
			},
		},
		{
			name:  "logical OR expression",
			input: "a || b",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "a"},
				{Type: TokenTypeOr, Value: "||"},
				{Type: TokenTypeIdentifier, Value: "b"},
			},
		},
		{
			name:  "complex boolean expression",
			input: "a && b || c",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "a"},
				{Type: TokenTypeAnd, Value: "&&"},
				{Type: TokenTypeIdentifier, Value: "b"},
				{Type: TokenTypeOr, Value: "||"},
				{Type: TokenTypeIdentifier, Value: "c"},
			},
		},
		{
			name:  "boolean literals in expression",
			input: "true && false",
			expected: []Token{
				{Type: TokenTypeTrue, Value: "true"},
				{Type: TokenTypeAnd, Value: "&&"},
				{Type: TokenTypeFalse, Value: "false"},
			},
		},
		{
			name:  "mixed comparison and logical",
			input: "x < 5 && y > 3",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "x"},
				{Type: TokenTypeLessThan, Value: "<"},
				{Type: TokenTypeInteger, Value: "5"},
				{Type: TokenTypeAnd, Value: "&&"},
				{Type: TokenTypeIdentifier, Value: "y"},
				{Type: TokenTypeGreaterThan, Value: ">"},
				{Type: TokenTypeInteger, Value: "3"},
			},
		},
		{
			name:  "identifiers starting with true/false",
			input: "trueValue falseFlag",
			expected: []Token{
				{Type: TokenTypeIdentifier, Value: "trueValue"},
				{Type: TokenTypeIdentifier, Value: "falseFlag"},
			},
		},
		{
			name:  "val with boolean",
			input: "val x = true",
			expected: []Token{
				{Type: TokenTypeVal, Value: "val"},
				{Type: TokenTypeIdentifier, Value: "x"},
				{Type: TokenTypeAssign, Value: "="},
				{Type: TokenTypeTrue, Value: "true"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Parse()

			if len(l.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", l.Errors)
			}

			if len(l.Tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(l.Tokens), l.Tokens)
			}

			for i, token := range l.Tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("token %d: expected type %v, got %v", i, tt.expected[i].Type, token.Type)
				}
				if token.Value != tt.expected[i].Value {
					t.Errorf("token %d: expected value %q, got %q", i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

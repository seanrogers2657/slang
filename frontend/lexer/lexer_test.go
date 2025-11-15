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
			name:          "single equals sign",
			input:         "5 = 5",
			expectedError: "unexpected character: '=' (did you mean '=='?)",
		},
		{
			name:          "exclamation without equals",
			input:         "!5",
			expectedError: "unexpected character: '!'",
		},
		{
			name:          "unexpected character",
			input:         "2 @ 5",
			expectedError: "unexpected character: '@'",
		},
		{
			name:          "letter character",
			input:         "abc",
			expectedError: "unexpected character: 'a'",
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
			input: `"hello"`,
			expected: []Token{
				{Type: TokenTypeString, Value: "hello"},
			},
		},
		{
			name:  "empty string",
			input: `""`,
			expected: []Token{
				{Type: TokenTypeString, Value: ""},
			},
		},
		{
			name:  "string with spaces",
			input: `"hello world"`,
			expected: []Token{
				{Type: TokenTypeString, Value: "hello world"},
			},
		},
		{
			name:  "string with escape sequences",
			input: `"hello\nworld"`,
			expected: []Token{
				{Type: TokenTypeString, Value: "hello\nworld"},
			},
		},
		{
			name:  "string with tab escape",
			input: `"hello\tworld"`,
			expected: []Token{
				{Type: TokenTypeString, Value: "hello\tworld"},
			},
		},
		{
			name:  "string with escaped quote",
			input: `"hello \"world\""`,
			expected: []Token{
				{Type: TokenTypeString, Value: `hello "world"`},
			},
		},
		{
			name:  "string with escaped backslash",
			input: `"hello\\world"`,
			expected: []Token{
				{Type: TokenTypeString, Value: `hello\world`},
			},
		},
		{
			name:  "multiple strings",
			input: `"hello" "world"`,
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
			input:         `"hello`,
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

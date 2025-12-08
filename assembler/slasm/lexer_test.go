package slasm

import (
	"testing"
)

func TestLexer_TokenTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:  "directive",
			input: ".global _start",
			expected: []TokenType{
				TokenDirective,
				TokenIdentifier,
				TokenEOF,
			},
		},
		{
			name:  "label definition",
			input: "main:",
			expected: []TokenType{
				TokenIdentifier,
				TokenColon,
				TokenEOF,
			},
		},
		{
			name:  "instruction with registers",
			input: "mov x0, x1",
			expected: []TokenType{
				TokenIdentifier, // mov
				TokenRegister,   // x0
				TokenComma,
				TokenRegister, // x1
				TokenEOF,
			},
		},
		{
			name:  "instruction with immediate",
			input: "mov x0, #42",
			expected: []TokenType{
				TokenIdentifier, // mov
				TokenRegister,   // x0
				TokenComma,
				TokenHash,
				TokenInteger,
				TokenEOF,
			},
		},
		{
			name:  "memory operand",
			input: "ldr x0, [sp, #16]",
			expected: []TokenType{
				TokenIdentifier, // ldr
				TokenRegister,   // x0
				TokenComma,
				TokenLBracket,
				TokenRegister, // sp
				TokenComma,
				TokenHash,
				TokenInteger,
				TokenRBracket,
				TokenEOF,
			},
		},
		{
			name:  "page offset relocation",
			input: "adrp x0, buffer@PAGE",
			expected: []TokenType{
				TokenIdentifier, // adrp
				TokenRegister,   // x0
				TokenComma,
				TokenIdentifier, // buffer
				TokenAt,
				TokenIdentifier, // PAGE
				TokenEOF,
			},
		},
		{
			name:  "comment",
			input: "    // This is a comment",
			expected: []TokenType{
				TokenComment,
				TokenEOF,
			},
		},
		{
			name:  "multiple lines",
			input: "mov x0, #1\nadd x1, x0, x2",
			expected: []TokenType{
				TokenIdentifier, // mov
				TokenRegister,   // x0
				TokenComma,
				TokenHash,
				TokenInteger,
				TokenNewline,
				TokenIdentifier, // add
				TokenRegister,   // x1
				TokenComma,
				TokenRegister, // x0
				TokenComma,
				TokenRegister, // x2
				TokenEOF,
			},
		},
		{
			name:  "data section directives",
			input: ".data\n.align 3\nbuffer: .space 32",
			expected: []TokenType{
				TokenDirective, // .data
				TokenNewline,
				TokenDirective, // .align
				TokenInteger,   // 3
				TokenNewline,
				TokenIdentifier, // buffer
				TokenColon,
				TokenDirective, // .space
				TokenInteger,   // 32
				TokenEOF,
			},
		},
		{
			name:  "string literal",
			input: `.asciz "Hello, World!"`,
			expected: []TokenType{
				TokenDirective, // .asciz
				TokenString,    // "Hello, World!"
				TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("token %d: expected type %v, got %v (value: %q)",
						i, expectedType, tokens[i].Type, tokens[i].Value)
				}
			}
		})
	}
}

func TestLexer_TokenValues(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		tokenIndex    int
	}{
		{
			name:          "directive name",
			input:         ".global",
			expectedValue: ".global",
			tokenIndex:    0,
		},
		{
			name:          "register name x0",
			input:         "x0",
			expectedValue: "x0",
			tokenIndex:    0,
		},
		{
			name:          "register name sp",
			input:         "sp",
			expectedValue: "sp",
			tokenIndex:    0,
		},
		{
			name:          "immediate value",
			input:         "#42",
			expectedValue: "42",
			tokenIndex:    1, // Hash is token 0, value is token 1
		},
		{
			name:          "label name",
			input:         "_start:",
			expectedValue: "_start",
			tokenIndex:    0,
		},
		{
			name:          "instruction mnemonic",
			input:         "mov",
			expectedValue: "mov",
			tokenIndex:    0,
		},
		{
			name:          "negative number",
			input:         "#-16",
			expectedValue: "-16",
			tokenIndex:    1,
		},
		{
			name:          "string literal with escapes",
			input:         `.asciz "Hello\nWorld"`,
			expectedValue: "Hello\\nWorld",
			tokenIndex:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			if tt.tokenIndex >= len(tokens) {
				t.Fatalf("token index %d out of range (only %d tokens)", tt.tokenIndex, len(tokens))
			}

			if tokens[tt.tokenIndex].Value != tt.expectedValue {
				t.Errorf("expected value %q, got %q", tt.expectedValue, tokens[tt.tokenIndex].Value)
			}
		})
	}
}

func TestLexer_LineAndColumn(t *testing.T) {
	input := `mov x0, #1
add x1, x0, x2
sub x2, x1, x0`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()

	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	// Find the "add" token (should be on line 2)
	addTokenIndex := -1
	for i, tok := range tokens {
		if tok.Value == "add" {
			addTokenIndex = i
			break
		}
	}

	if addTokenIndex == -1 {
		t.Fatal("could not find 'add' token")
	}

	if tokens[addTokenIndex].Line != 2 {
		t.Errorf("expected 'add' token on line 2, got line %d", tokens[addTokenIndex].Line)
	}

	// Find the "sub" token (should be on line 3)
	subTokenIndex := -1
	for i, tok := range tokens {
		if tok.Value == "sub" {
			subTokenIndex = i
			break
		}
	}

	if subTokenIndex == -1 {
		t.Fatal("could not find 'sub' token")
	}

	if tokens[subTokenIndex].Line != 3 {
		t.Errorf("expected 'sub' token on line 3, got line %d", tokens[subTokenIndex].Line)
	}
}

func TestLexer_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "leading whitespace",
			input: "    mov x0, #1",
		},
		{
			name:  "trailing whitespace",
			input: "mov x0, #1    ",
		},
		{
			name:  "tabs",
			input: "\tmov\tx0,\t#1",
		},
		{
			name:  "multiple spaces",
			input: "mov     x0,     #1",
		},
		{
			name:  "mixed whitespace",
			input: "  \t  mov   \t  x0,  \t #1  \t  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			// Should tokenize to: mov, x0, comma, hash, 1, EOF
			// Whitespace should be ignored (except newlines)
			expectedTokens := []string{"mov", "x0", ",", "#", "1"}
			actualTokens := []string{}

			for _, tok := range tokens {
				if tok.Type == TokenEOF {
					break
				}
				actualTokens = append(actualTokens, tok.Value)
			}

			if len(actualTokens) != len(expectedTokens) {
				t.Errorf("expected %d tokens, got %d", len(expectedTokens), len(actualTokens))
			}

			for i := 0; i < len(expectedTokens) && i < len(actualTokens); i++ {
				if actualTokens[i] != expectedTokens[i] {
					t.Errorf("token %d: expected %q, got %q", i, expectedTokens[i], actualTokens[i])
				}
			}
		})
	}
}

func TestLexer_ConditionalBranch(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
	}{
		{"b.eq", "b.eq target", "b.eq"},
		{"b.ne", "b.ne target", "b.ne"},
		{"b.cs", "b.cs target", "b.cs"},
		{"b.hs", "b.hs target", "b.hs"},
		{"b.cc", "b.cc target", "b.cc"},
		{"b.lo", "b.lo target", "b.lo"},
		{"b.mi", "b.mi target", "b.mi"},
		{"b.pl", "b.pl target", "b.pl"},
		{"b.vs", "b.vs target", "b.vs"},
		{"b.vc", "b.vc target", "b.vc"},
		{"b.hi", "b.hi target", "b.hi"},
		{"b.ls", "b.ls target", "b.ls"},
		{"b.ge", "b.ge target", "b.ge"},
		{"b.lt", "b.lt target", "b.lt"},
		{"b.gt", "b.gt target", "b.gt"},
		{"b.le", "b.le target", "b.le"},
		{"b.al", "b.al target", "b.al"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			if len(tokens) < 1 {
				t.Fatal("expected at least one token")
			}

			if tokens[0].Type != TokenIdentifier {
				t.Errorf("expected TokenIdentifier, got %v", tokens[0].Type)
			}

			if tokens[0].Value != tt.expectedValue {
				t.Errorf("expected value %q, got %q", tt.expectedValue, tokens[0].Value)
			}
		})
	}
}

func TestLexer_RegisterRecognition(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldBeReg bool
	}{
		{"x0", "x0", true},
		{"x30", "x30", true},
		{"w0", "w0", true},
		{"w30", "w30", true},
		{"sp", "sp", true},
		{"lr", "lr", true},
		{"xzr", "xzr", true},
		{"wzr", "wzr", true},
		{"x31", "x31", false},   // Invalid - should be identifier
		{"x99", "x99", false},   // Invalid - should be identifier
		{"main", "main", false}, // Regular identifier
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			if len(tokens) < 1 {
				t.Fatal("expected at least one token")
			}

			isReg := tokens[0].Type == TokenRegister
			if isReg != tt.shouldBeReg {
				t.Errorf("expected register=%v, got register=%v for %q", tt.shouldBeReg, isReg, tt.input)
			}
		})
	}
}

func TestLexer_UnterminatedStrings(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "unterminated at EOF",
			input:     `"hello`,
			wantError: true,
			errorMsg:  "unterminated string literal",
		},
		{
			name:      "unterminated at newline",
			input:     "\"hello\nworld\"",
			wantError: true,
			errorMsg:  "unterminated string literal",
		},
		{
			name:      "unterminated empty string at EOF",
			input:     `"`,
			wantError: true,
			errorMsg:  "unterminated string literal",
		},
		{
			name:      "properly terminated",
			input:     `"hello"`,
			wantError: false,
		},
		{
			name:      "empty string properly terminated",
			input:     `""`,
			wantError: false,
		},
		{
			name:      "string with escapes properly terminated",
			input:     `"hello\nworld"`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
					return
				}
				// Check that we got a TokenError with the right message
				foundError := false
				for _, tok := range tokens {
					if tok.Type == TokenError && tok.Value == tt.errorMsg {
						foundError = true
						break
					}
				}
				if !foundError {
					t.Errorf("expected TokenError with message %q, got tokens: %v", tt.errorMsg, tokens)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

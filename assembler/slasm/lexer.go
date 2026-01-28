package slasm

import (
	"fmt"
)

// TokenType represents the type of token in assembly source
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenNewline
	TokenComment
	TokenError // Error token for invalid input

	// Identifiers and literals
	TokenIdentifier // label names, instruction mnemonics
	TokenInteger    // immediate values
	TokenString     // string literals

	// Directives
	TokenDirective // .data, .text, .global, .align, etc.

	// Registers
	TokenRegister // x0-x30, sp, lr, etc.

	// Operators and punctuation
	TokenComma
	TokenColon
	TokenHash        // # for immediates
	TokenLBracket    // [
	TokenRBracket    // ]
	TokenAt          // @ for PAGE/PAGEOFF
	TokenExclamation // ! for writeback addressing
	TokenEquals      // = for constant assignments
)

// Token represents a single token in assembly source
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// Lexer tokenizes assembly source code
type Lexer struct {
	source  string
	pos     int
	line    int
	column  int
	current byte
}

// registerSet contains all valid ARM64 register names for O(1) lookup
var registerSet = func() map[string]struct{} {
	regs := make(map[string]struct{}, 70)
	// x0-x30, w0-w30
	for i := 0; i <= 30; i++ {
		regs[fmt.Sprintf("x%d", i)] = struct{}{}
		regs[fmt.Sprintf("w%d", i)] = struct{}{}
	}
	// Special registers
	regs["sp"] = struct{}{}
	regs["xzr"] = struct{}{}
	regs["wzr"] = struct{}{}
	regs["lr"] = struct{}{}
	return regs
}()

// isRegisterFast checks if a string is a valid ARM64 register using map lookup
func isRegisterFast(name string) bool {
	_, ok := registerSet[name]
	return ok
}

// NewLexer creates a new lexer for the given source code
func NewLexer(source string) *Lexer {
	l := &Lexer{
		source: source,
		pos:    0,
		line:   1,
		column: 1,
	}
	if len(source) > 0 {
		l.current = source[0]
	}
	return l
}

// NextToken returns the next token from the source
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	// Save position for this token
	line := l.line
	column := l.column

	// Check for EOF
	if l.pos >= len(l.source) {
		return Token{Type: TokenEOF, Line: line, Column: column}
	}

	// Newline
	if l.current == '\n' {
		l.advance()
		return Token{Type: TokenNewline, Value: "\n", Line: line, Column: column}
	}

	// Comments
	if l.current == '/' && l.peek() == '/' {
		return l.readComment(line, column)
	}
	if l.current == ';' {
		return l.readComment(line, column)
	}

	// Directives (start with .)
	if l.current == '.' {
		return l.readDirective(line, column)
	}

	// Numbers (for immediates after #)
	if l.current >= '0' && l.current <= '9' {
		return l.readNumber(line, column)
	}
	if l.current == '-' && l.peek() >= '0' && l.peek() <= '9' {
		return l.readNumber(line, column)
	}

	// String literals
	if l.current == '"' {
		return l.readString(line, column)
	}

	// Identifiers, registers, instructions
	if l.isIdentifierStart(l.current) {
		return l.readIdentifier(line, column)
	}

	// Punctuation
	switch l.current {
	case ':':
		l.advance()
		return Token{Type: TokenColon, Value: ":", Line: line, Column: column}
	case ',':
		l.advance()
		return Token{Type: TokenComma, Value: ",", Line: line, Column: column}
	case '#':
		l.advance()
		return Token{Type: TokenHash, Value: "#", Line: line, Column: column}
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Value: "[", Line: line, Column: column}
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Value: "]", Line: line, Column: column}
	case '@':
		l.advance()
		return Token{Type: TokenAt, Value: "@", Line: line, Column: column}
	case '!':
		l.advance()
		return Token{Type: TokenExclamation, Value: "!", Line: line, Column: column}
	case '=':
		l.advance()
		return Token{Type: TokenEquals, Value: "=", Line: line, Column: column}
	}

	// Unknown character - return error token
	char := string(l.current)
	l.advance()
	return Token{
		Type:   TokenError,
		Value:  char,
		Line:   line,
		Column: column,
	}
}

// LexerErrors represents multiple lexer errors
type LexerErrors struct {
	Errors []error
}

func (e *LexerErrors) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d lexer errors: %v (and %d more)", len(e.Errors), e.Errors[0], len(e.Errors)-1)
}

// Tokenize returns all tokens from the source
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	var errors []error

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)

		// Collect lexer errors
		if tok.Type == TokenError {
			errors = append(errors, fmt.Errorf("line %d:%d: unexpected character '%s'",
				tok.Line, tok.Column, tok.Value))
		}

		if tok.Type == TokenEOF {
			break
		}
	}

	// Return all errors wrapped in LexerErrors
	if len(errors) > 0 {
		return tokens, &LexerErrors{Errors: errors}
	}

	return tokens, nil
}

// Helper methods

func (l *Lexer) advance() {
	if l.current == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.pos++
	if l.pos < len(l.source) {
		l.current = l.source[l.pos]
	}
}

func (l *Lexer) peek() byte {
	if l.pos+1 < len(l.source) {
		return l.source[l.pos+1]
	}
	return 0
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.source) && (l.current == ' ' || l.current == '\t' || l.current == '\r') {
		l.advance()
	}
}

func (l *Lexer) isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func (l *Lexer) isIdentifierChar(ch byte) bool {
	return l.isIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func (l *Lexer) readComment(line, column int) Token {
	start := l.pos
	for l.pos < len(l.source) && l.current != '\n' {
		l.advance()
	}
	return Token{Type: TokenComment, Value: l.source[start:l.pos], Line: line, Column: column}
}

func (l *Lexer) readDirective(line, column int) Token {
	start := l.pos
	l.advance() // skip the leading '.'

	// Read the rest of the directive name
	for l.pos < len(l.source) && l.isIdentifierChar(l.current) {
		l.advance()
	}
	return Token{Type: TokenDirective, Value: l.source[start:l.pos], Line: line, Column: column}
}

func (l *Lexer) readNumber(line, column int) Token {
	start := l.pos

	// Handle negative numbers
	if l.current == '-' {
		l.advance()
	}

	// Check for hex prefix (0x or 0X)
	if l.current == '0' && l.pos+1 < len(l.source) {
		next := l.source[l.pos+1]
		if next == 'x' || next == 'X' {
			l.advance() // '0'
			l.advance() // 'x' or 'X'
			// Read hex digits
			for l.pos < len(l.source) && l.isHexDigit(l.current) {
				l.advance()
			}
			return Token{Type: TokenInteger, Value: l.source[start:l.pos], Line: line, Column: column}
		}
	}

	// Read decimal digits
	for l.pos < len(l.source) && l.current >= '0' && l.current <= '9' {
		l.advance()
	}
	return Token{Type: TokenInteger, Value: l.source[start:l.pos], Line: line, Column: column}
}

func (l *Lexer) isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

func (l *Lexer) readString(line, column int) Token {
	l.advance() // Skip opening quote
	start := l.pos

	// Fast path: check if string has no escapes
	hasEscape := false
	for l.pos < len(l.source) && l.current != '"' && l.current != '\n' {
		if l.current == '\\' {
			hasEscape = true
		}
		l.advance()
	}

	// Check for unterminated string (hit EOF or newline before closing quote)
	if l.pos >= len(l.source) || l.current == '\n' {
		return Token{Type: TokenError, Value: "unterminated string literal", Line: line, Column: column}
	}

	value := l.source[start:l.pos]

	// Only process escapes if needed (rare case)
	if hasEscape {
		value = processEscapes(value)
	}

	l.advance() // Skip closing quote
	return Token{Type: TokenString, Value: value, Line: line, Column: column}
}

// processEscapes handles escape sequences in strings (only called when needed)
func processEscapes(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				result = append(result, '\\', 'n')
			case 't':
				result = append(result, '\\', 't')
			case '\\':
				result = append(result, '\\', '\\')
			case '"':
				result = append(result, '"')
			default:
				result = append(result, s[i])
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

func (l *Lexer) readIdentifier(line, column int) Token {
	start := l.pos
	for l.pos < len(l.source) && l.isIdentifierChar(l.current) {
		l.advance()
	}
	value := l.source[start:l.pos]

	// Check for conditional branch: b.cond (e.g., b.eq, b.ne, b.lt, b.gt, etc.)
	if value == "b" && l.current == '.' {
		l.advance() // consume the '.'
		condStart := l.pos
		for l.pos < len(l.source) && l.isIdentifierChar(l.current) {
			l.advance()
		}
		// Return b.cond regardless of whether it's a valid condition code
		// (let the encoder handle validation)
		return Token{Type: TokenIdentifier, Value: "b." + l.source[condStart:l.pos], Line: line, Column: column}
	}

	// Check if it's a register using fast map lookup
	if isRegisterFast(value) {
		return Token{Type: TokenRegister, Value: value, Line: line, Column: column}
	}

	return Token{Type: TokenIdentifier, Value: value, Line: line, Column: column}
}

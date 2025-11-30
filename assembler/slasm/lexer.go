package slasm

import "fmt"

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
	TokenHash      // # for immediates
	TokenLBracket  // [
	TokenRBracket  // ]
	TokenAt        // @ for PAGE/PAGEOFF
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
	current rune
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
		l.current = rune(source[0])
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

	// Return first error if any
	if len(errors) > 0 {
		return tokens, errors[0]
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
		l.current = rune(l.source[l.pos])
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 < len(l.source) {
		return rune(l.source[l.pos+1])
	}
	return 0
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.source) && (l.current == ' ' || l.current == '\t' || l.current == '\r') {
		l.advance()
	}
}

func (l *Lexer) isIdentifierStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func (l *Lexer) isIdentifierChar(ch rune) bool {
	return l.isIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func (l *Lexer) readComment(line, column int) Token {
	value := ""
	for l.pos < len(l.source) && l.current != '\n' {
		value += string(l.current)
		l.advance()
	}
	return Token{Type: TokenComment, Value: value, Line: line, Column: column}
}

func (l *Lexer) readDirective(line, column int) Token {
	value := ""
	// Include the leading .
	value += string(l.current)
	l.advance()

	// Read the rest of the directive name
	for l.pos < len(l.source) && l.isIdentifierChar(l.current) {
		value += string(l.current)
		l.advance()
	}
	return Token{Type: TokenDirective, Value: value, Line: line, Column: column}
}

func (l *Lexer) readNumber(line, column int) Token {
	value := ""

	// Handle negative numbers
	if l.current == '-' {
		value += string(l.current)
		l.advance()
	}

	// Read digits
	for l.pos < len(l.source) && l.current >= '0' && l.current <= '9' {
		value += string(l.current)
		l.advance()
	}
	return Token{Type: TokenInteger, Value: value, Line: line, Column: column}
}

func (l *Lexer) readString(line, column int) Token {
	value := ""
	l.advance() // Skip opening quote

	for l.pos < len(l.source) && l.current != '"' {
		if l.current == '\\' {
			// Handle escape sequences
			l.advance()
			if l.pos < len(l.source) {
				switch l.current {
				case 'n':
					value += "\\n"
				case 't':
					value += "\\t"
				case '\\':
					value += "\\\\"
				case '"':
					value += "\""
				default:
					value += string(l.current)
				}
				l.advance()
			}
		} else {
			value += string(l.current)
			l.advance()
		}
	}

	if l.current == '"' {
		l.advance() // Skip closing quote
	}

	return Token{Type: TokenString, Value: value, Line: line, Column: column}
}

func (l *Lexer) readIdentifier(line, column int) Token {
	value := ""
	for l.pos < len(l.source) && l.isIdentifierChar(l.current) {
		value += string(l.current)
		l.advance()
	}

	// Check if it's a register using shared utility
	if IsRegister(value) {
		return Token{Type: TokenRegister, Value: value, Line: line, Column: column}
	}

	return Token{Type: TokenIdentifier, Value: value, Line: line, Column: column}
}

package asm

// TokenType represents the type of token in assembly source
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenNewline
	TokenComment

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
	// TODO: Implement lexer initialization
	return &Lexer{
		source: source,
		line:   1,
		column: 1,
	}
}

// NextToken returns the next token from the source
func (l *Lexer) NextToken() Token {
	// TODO: Implement tokenization
	// 1. Skip whitespace (except newlines)
	// 2. Handle comments (// or ; style)
	// 3. Recognize directives (.data, .text, etc.)
	// 4. Recognize registers (x0-x30, w0-w30, sp, lr, etc.)
	// 5. Recognize identifiers (labels, instructions)
	// 6. Recognize integers and immediates
	// 7. Recognize strings
	// 8. Recognize operators and punctuation
	return Token{Type: TokenEOF}
}

// Tokenize returns all tokens from the source
func (l *Lexer) Tokenize() ([]Token, error) {
	// TODO: Implement full tokenization
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}

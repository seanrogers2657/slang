package lexer

import (
	"fmt"
	"unicode"

	"github.com/seanrogers2657/slang/frontend/ast"
)

type TokenType int

const (
	TokenTypeInteger TokenType = iota
	TokenTypeString
	TokenTypePlus
	TokenTypeMinus
	TokenTypeMultiply
	TokenTypeDivide
	TokenTypeModulo
	TokenTypeEqual
	TokenTypeNotEqual
	TokenTypeLessThan
	TokenTypeGreaterThan
	TokenTypeLessThanOrEqual
	TokenTypeGreaterThanOrEqual
	TokenTypeNewline
	TokenTypePrint
)

type Token struct {
	Type  TokenType
	Value string
	Pos   ast.Position // position where token starts
}

type lexer struct {
	Source []byte
	Index  int

	// Position tracking
	Line   int // current line (1-indexed)
	Column int // current column (1-indexed)

	Errors []error
	Tokens []Token
}

func NewLexer(source []byte) *lexer {
	lexer := &lexer{
		Source: source,
		Index:  0,
		Line:   1,
		Column: 1,
	}

	return lexer
}

// currentPos returns the current position in the source
func (p *lexer) currentPos() ast.Position {
	return ast.Position{
		Line:   p.Line,
		Column: p.Column,
		Offset: p.Index,
	}
}

// advance moves to the next character and updates position tracking
func (p *lexer) advance() {
	if p.Index < len(p.Source) {
		if p.Source[p.Index] == '\n' {
			p.Line++
			p.Column = 1
		} else {
			p.Column++
		}
		p.Index++
	}
}

func (p *lexer) ParseNumber() {
	startPos := p.currentPos()
	number := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]
		//spew.Dump("current char", string(currentChar), unicode.IsDigit(rune(currentChar)))

		if !unicode.IsDigit(rune(currentChar)) {
			break
		}

		number += string(currentChar)
		p.advance()
	}

	p.Tokens = append(p.Tokens, Token{
		Type:  TokenTypeInteger,
		Value: string(number),
		Pos:   startPos,
	})
}

func (p *lexer) ParseString() {
	startPos := p.currentPos()
	// Skip opening quote
	p.advance()

	str := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Check for closing quote
		if currentChar == '"' {
			p.advance() // Skip closing quote
			p.Tokens = append(p.Tokens, Token{
				Type:  TokenTypeString,
				Value: str,
				Pos:   startPos,
			})
			return
		}

		// Handle escape sequences
		if currentChar == '\\' && p.Index+1 < len(p.Source) {
			p.advance()
			nextChar := p.Source[p.Index]
			switch nextChar {
			case 'n':
				str += "\n"
			case 't':
				str += "\t"
			case 'r':
				str += "\r"
			case '\\':
				str += "\\"
			case '"':
				str += "\""
			default:
				// Unknown escape sequence, just include the backslash
				str += "\\" + string(nextChar)
			}
			p.advance()
		} else {
			str += string(currentChar)
			p.advance()
		}
	}

	// If we reach here, the string wasn't closed
	p.Errors = append(p.Errors, fmt.Errorf("unterminated string literal"))
}

func (p *lexer) ParseKeyword() {
	startPos := p.currentPos()
	keyword := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Keywords are alphabetic characters
		if !unicode.IsLetter(rune(currentChar)) {
			break
		}

		keyword += string(currentChar)
		p.advance()
	}

	// Check if it's a recognized keyword
	switch keyword {
	case "print":
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypePrint,
			Value: keyword,
			Pos:   startPos,
		})
	default:
		p.Errors = append(p.Errors, fmt.Errorf("unknown keyword: %q", keyword))
	}
}

func (p *lexer) Parse() {
	for p.Index < len(p.Source) {
		b := p.Source[p.Index]
		// spew.Printf("parsing %v\n", string(b))

		if b == '\n' {
			// spew.Dump("is newline")
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeNewline, Value: "\n", Pos: pos})
			p.advance()
		} else if unicode.IsSpace(rune(b)) {
			// spew.Dump("is space")
			p.advance()
		} else if unicode.IsLetter(rune(b)) {
			p.ParseKeyword()
		} else if p.Source[p.Index] >= '0' && p.Source[p.Index] <= '9' {
			p.ParseNumber()
		} else if b == '"' {
			p.ParseString()
		} else if b == '+' {
			// spew.Dump("is plus")
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypePlus, Value: "+", Pos: pos})
			p.advance()
		} else if b == '-' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeMinus, Value: "-", Pos: pos})
			p.advance()
		} else if b == '*' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeMultiply, Value: "*", Pos: pos})
			p.advance()
		} else if b == '/' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeDivide, Value: "/", Pos: pos})
			p.advance()
		} else if b == '%' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeModulo, Value: "%", Pos: pos})
			p.advance()
		} else if b == '=' {
			// Check for ==
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeEqual, Value: "==", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q (did you mean '=='?)", b))
				return
			}
		} else if b == '!' {
			// Check for !=
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeNotEqual, Value: "!=", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q", b))
				return
			}
		} else if b == '<' {
			// Check for <=
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeLessThanOrEqual, Value: "<=", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeLessThan, Value: "<", Pos: pos})
				p.advance()
			}
		} else if b == '>' {
			// Check for >=
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeGreaterThanOrEqual, Value: ">=", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeGreaterThan, Value: ">", Pos: pos})
				p.advance()
			}
		} else {
			// spew.Dump("has parsing error")
			p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q", b))
			return
		}
	}
}

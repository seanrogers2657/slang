package lexer

import (
	"fmt"
	"unicode"

	"github.com/davecgh/go-spew/spew"
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
}

type lexer struct {
	Source []byte
	Index  int

	Errors []error
	Tokens []Token
}

func NewLexer(source []byte) *lexer {
	lexer := &lexer{
		Source: source,
		Index:  0,
	}

	return lexer
}

func (p *lexer) ParseNumber() {
	number := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]
		//spew.Dump("current char", string(currentChar), unicode.IsDigit(rune(currentChar)))

		if !unicode.IsDigit(rune(currentChar)) {
			break
		}

		number += string(currentChar)
		p.Index++
	}

	p.Tokens = append(p.Tokens, Token{Type: TokenTypeInteger, Value: string(number)})
}

func (p *lexer) ParseString() {
	// Skip opening quote
	p.Index++

	str := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Check for closing quote
		if currentChar == '"' {
			p.Index++ // Skip closing quote
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeString, Value: str})
			return
		}

		// Handle escape sequences
		if currentChar == '\\' && p.Index+1 < len(p.Source) {
			p.Index++
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
			p.Index++
		} else {
			str += string(currentChar)
			p.Index++
		}
	}

	// If we reach here, the string wasn't closed
	p.Errors = append(p.Errors, fmt.Errorf("unterminated string literal"))
}

func (p *lexer) ParseKeyword() {
	keyword := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Keywords are alphabetic characters
		if !unicode.IsLetter(rune(currentChar)) {
			break
		}

		keyword += string(currentChar)
		p.Index++
	}

	// Check if it's a recognized keyword
	switch keyword {
	case "print":
		p.Tokens = append(p.Tokens, Token{Type: TokenTypePrint, Value: keyword})
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
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeNewline, Value: "\n"})
			p.Index++
		} else if unicode.IsSpace(rune(b)) {
			// spew.Dump("is space")
			p.Index++
		} else if unicode.IsLetter(rune(b)) {
			p.ParseKeyword()
		} else if p.Source[p.Index] >= '0' && p.Source[p.Index] <= '9' {
			p.ParseNumber()
		} else if b == '"' {
			p.ParseString()
		} else if b == '+' {
			// spew.Dump("is plus")
			p.Tokens = append(p.Tokens, Token{Type: TokenTypePlus, Value: "+"})
			p.Index++
		} else if b == '-' {
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeMinus, Value: "-"})
			p.Index++
		} else if b == '*' {
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeMultiply, Value: "*"})
			p.Index++
		} else if b == '/' {
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeDivide, Value: "/"})
			p.Index++
		} else if b == '%' {
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeModulo, Value: "%"})
			p.Index++
		} else if b == '=' {
			// Check for ==
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeEqual, Value: "=="})
				p.Index += 2
			} else {
				p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q (did you mean '=='?)", b))
				return
			}
		} else if b == '!' {
			// Check for !=
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeNotEqual, Value: "!="})
				p.Index += 2
			} else {
				p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q", b))
				return
			}
		} else if b == '<' {
			// Check for <=
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeLessThanOrEqual, Value: "<="})
				p.Index += 2
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeLessThan, Value: "<"})
				p.Index++
			}
		} else if b == '>' {
			// Check for >=
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeGreaterThanOrEqual, Value: ">="})
				p.Index += 2
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeGreaterThan, Value: ">"})
				p.Index++
			}
		} else {
			spew.Dump("has parsing error")
			p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q", b))
			return
		}
	}
}

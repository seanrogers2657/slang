package lexer

import (
	"fmt"
	"unicode"

	"github.com/davecgh/go-spew/spew"
)

type TokenType int

const (
	TokenTypeInteger TokenType = iota
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

func (p *lexer) Parse() {
	for p.Index < len(p.Source) {
		b := p.Source[p.Index]
		// spew.Printf("parsing %v\n", string(b))

		if unicode.IsSpace(rune(b)) {
			// spew.Dump("is space")
			p.Index++
		} else if p.Source[p.Index] >= '0' && p.Source[p.Index] <= '9' {
			p.ParseNumber()
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
		} else if b == '\n' {
			// spew.Dump("is newline")
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeNewline, Value: "\n"})
			p.Index++
		} else {
			spew.Dump("has parsing error")
			p.Errors = append(p.Errors, fmt.Errorf("unexpected character: %q", b))
			return
		}
	}
}

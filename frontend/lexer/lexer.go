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
	TokenTypeFn
	TokenTypeVal
	TokenTypeVar
	TokenTypeAssign
	TokenTypeLParen
	TokenTypeRParen
	TokenTypeLBrace
	TokenTypeRBrace
	TokenTypeIdentifier
	TokenTypeComma
	TokenTypeColon
	TokenTypeReturn
)

// String returns a human-readable name for the token type
func (t TokenType) String() string {
	switch t {
	case TokenTypeInteger:
		return "INTEGER"
	case TokenTypeString:
		return "STRING"
	case TokenTypePlus:
		return "PLUS"
	case TokenTypeMinus:
		return "MINUS"
	case TokenTypeMultiply:
		return "MULTIPLY"
	case TokenTypeDivide:
		return "DIVIDE"
	case TokenTypeModulo:
		return "MODULO"
	case TokenTypeEqual:
		return "EQUAL"
	case TokenTypeNotEqual:
		return "NOT_EQUAL"
	case TokenTypeLessThan:
		return "LESS_THAN"
	case TokenTypeGreaterThan:
		return "GREATER_THAN"
	case TokenTypeLessThanOrEqual:
		return "LESS_EQUAL"
	case TokenTypeGreaterThanOrEqual:
		return "GREATER_EQUAL"
	case TokenTypeNewline:
		return "NEWLINE"
	case TokenTypePrint:
		return "PRINT"
	case TokenTypeFn:
		return "FN"
	case TokenTypeVal:
		return "VAL"
	case TokenTypeVar:
		return "VAR"
	case TokenTypeAssign:
		return "ASSIGN"
	case TokenTypeLParen:
		return "LPAREN"
	case TokenTypeRParen:
		return "RPAREN"
	case TokenTypeLBrace:
		return "LBRACE"
	case TokenTypeRBrace:
		return "RBRACE"
	case TokenTypeIdentifier:
		return "IDENTIFIER"
	case TokenTypeComma:
		return "COMMA"
	case TokenTypeColon:
		return "COLON"
	case TokenTypeReturn:
		return "RETURN"
	default:
		return "UNKNOWN"
	}
}

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

// skipLineComment skips a // comment until end of line
func (p *lexer) skipLineComment() {
	// Skip the //
	p.advance()
	p.advance()

	// Skip until end of line or end of file
	for p.Index < len(p.Source) && p.Source[p.Index] != '\n' {
		p.advance()
	}
	// Don't consume the newline - let the main loop handle it
}

func (p *lexer) ParseIdentifierOrKeyword() {
	startPos := p.currentPos()
	ident := ""
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Identifiers start with a letter, then can contain letters, digits, or underscores
		if len(ident) == 0 {
			// First character must be a letter
			if !unicode.IsLetter(rune(currentChar)) {
				break
			}
		} else {
			// Subsequent characters can be letters, digits, or underscores
			if !unicode.IsLetter(rune(currentChar)) && !unicode.IsDigit(rune(currentChar)) && currentChar != '_' {
				break
			}
		}

		ident += string(currentChar)
		p.advance()
	}

	// Check if it's a recognized keyword
	switch ident {
	case "print":
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypePrint,
			Value: ident,
			Pos:   startPos,
		})
	case "fn":
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypeFn,
			Value: ident,
			Pos:   startPos,
		})
	case "val":
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypeVal,
			Value: ident,
			Pos:   startPos,
		})
	case "var":
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypeVar,
			Value: ident,
			Pos:   startPos,
		})
	case "return":
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypeReturn,
			Value: ident,
			Pos:   startPos,
		})
	default:
		p.Tokens = append(p.Tokens, Token{
			Type:  TokenTypeIdentifier,
			Value: ident,
			Pos:   startPos,
		})
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
			p.ParseIdentifierOrKeyword()
		} else if p.Source[p.Index] >= '0' && p.Source[p.Index] <= '9' {
			p.ParseNumber()
		} else if b == '"' {
			p.ParseString()
		} else if b == '(' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeLParen, Value: "(", Pos: pos})
			p.advance()
		} else if b == ')' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeRParen, Value: ")", Pos: pos})
			p.advance()
		} else if b == '{' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeLBrace, Value: "{", Pos: pos})
			p.advance()
		} else if b == '}' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeRBrace, Value: "}", Pos: pos})
			p.advance()
		} else if b == ',' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeComma, Value: ",", Pos: pos})
			p.advance()
		} else if b == ':' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeColon, Value: ":", Pos: pos})
			p.advance()
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
			// Check for // comment
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '/' {
				p.skipLineComment()
			} else {
				pos := p.currentPos()
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeDivide, Value: "/", Pos: pos})
				p.advance()
			}
		} else if b == '%' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeModulo, Value: "%", Pos: pos})
			p.advance()
		} else if b == '=' {
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				// ==
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeEqual, Value: "==", Pos: pos})
				p.advance()
				p.advance()
			} else {
				// Single = (assignment)
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeAssign, Value: "=", Pos: pos})
				p.advance()
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

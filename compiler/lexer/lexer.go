package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/errors"
)

// keywords maps keyword strings to their token types
var keywords = map[string]TokenType{
	"val":      TokenTypeVal,
	"var":      TokenTypeVar,
	"return":   TokenTypeReturn,
	"true":     TokenTypeTrue,
	"false":    TokenTypeFalse,
	"if":       TokenTypeIf,
	"else":     TokenTypeElse,
	"struct":   TokenTypeStruct,
	"for":      TokenTypeFor,
	"break":    TokenTypeBreak,
	"continue": TokenTypeContinue,
	"when":     TokenTypeWhen,
	"while":    TokenTypeWhile,
	"null":     TokenTypeNull,
	"class":    TokenTypeClass,
	"self":     TokenTypeSelf,
	"object":   TokenTypeObject,
	"new":      TokenTypeNew,
	"import":   TokenTypeImport,
}

type TokenType int

const (
	TokenTypeInvalid TokenType = iota // Zero value, used to detect uninitialized tokens
	TokenTypeInteger
	TokenTypeFloat
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
	TokenTypeTrue
	TokenTypeFalse
	TokenTypeAnd       // && logical AND
	TokenTypeAmpersand // & for &T borrow syntax
	TokenTypeOr
	TokenTypeNot
	TokenTypeIf
	TokenTypeElse
	TokenTypeStruct
	TokenTypeDot
	TokenTypeFor
	TokenTypeBreak
	TokenTypeContinue
	TokenTypeSemicolon
	TokenTypeWhen
	TokenTypeArrow
	TokenTypeLBracket
	TokenTypeRBracket
	TokenTypeWhile
	TokenTypeNull     // 'null' keyword
	TokenTypeQuestion // '?' for nullable type syntax
	TokenTypeSafeCall // '?.' safe call operator
	TokenTypeElvis    // '?:' elvis operator
	TokenTypeClass    // 'class' keyword
	TokenTypeSelf     // 'self' keyword
	TokenTypeObject   // 'object' keyword
	TokenTypeNew      // 'new' keyword
	TokenTypeImport   // 'import' keyword
)

// String returns a human-readable name for the token type
func (t TokenType) String() string {
	switch t {
	case TokenTypeInvalid:
		return "INVALID"
	case TokenTypeInteger:
		return "INTEGER"
	case TokenTypeFloat:
		return "FLOAT"
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
	case TokenTypeTrue:
		return "TRUE"
	case TokenTypeFalse:
		return "FALSE"
	case TokenTypeAnd:
		return "AND"
	case TokenTypeAmpersand:
		return "AMPERSAND"
	case TokenTypeOr:
		return "OR"
	case TokenTypeNot:
		return "NOT"
	case TokenTypeIf:
		return "IF"
	case TokenTypeElse:
		return "ELSE"
	case TokenTypeStruct:
		return "STRUCT"
	case TokenTypeDot:
		return "DOT"
	case TokenTypeFor:
		return "FOR"
	case TokenTypeBreak:
		return "BREAK"
	case TokenTypeContinue:
		return "CONTINUE"
	case TokenTypeSemicolon:
		return "SEMICOLON"
	case TokenTypeWhen:
		return "WHEN"
	case TokenTypeArrow:
		return "ARROW"
	case TokenTypeLBracket:
		return "LBRACKET"
	case TokenTypeRBracket:
		return "RBRACKET"
	case TokenTypeWhile:
		return "WHILE"
	case TokenTypeNull:
		return "NULL"
	case TokenTypeQuestion:
		return "QUESTION"
	case TokenTypeSafeCall:
		return "SAFE_CALL"
	case TokenTypeElvis:
		return "ELVIS"
	case TokenTypeClass:
		return "CLASS"
	case TokenTypeSelf:
		return "SELF"
	case TokenTypeObject:
		return "OBJECT"
	case TokenTypeNew:
		return "NEW"
	case TokenTypeImport:
		return "IMPORT"
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
	Source   []byte
	Index    int
	Filename string // source filename for error reporting

	// Position tracking
	Line   int // current line (1-indexed)
	Column int // current column (1-indexed)

	Errors []*errors.CompilerError
	Tokens []Token
}

func NewLexer(source []byte) *lexer {
	return NewLexerWithFilename(source, "")
}

// NewLexerWithFilename creates a new lexer with a source filename for error reporting
func NewLexerWithFilename(source []byte, filename string) *lexer {
	return &lexer{
		Source:   source,
		Filename: filename,
		Index:    0,
		Line:     1,
		Column:   1,
	}
}

// addError creates and adds a compiler error at the current position
func (p *lexer) addError(message string) *errors.CompilerError {
	pos := errors.Position{Line: p.Line, Column: p.Column, Offset: p.Index}
	err := errors.NewError(message, p.Filename, pos, "lexer")
	err.Tool = errors.ToolSL
	p.Errors = append(p.Errors, err)
	return err
}

// addErrorAt creates and adds a compiler error at a specific position
func (p *lexer) addErrorAt(message string, startPos ast.Position) *errors.CompilerError {
	pos := errors.Position{Line: startPos.Line, Column: startPos.Column, Offset: startPos.Offset}
	err := errors.NewError(message, p.Filename, pos, "lexer")
	err.Tool = errors.ToolSL
	p.Errors = append(p.Errors, err)
	return err
}

// currentPos returns the current position in the source
func (p *lexer) currentPos() ast.Position {
	return ast.Position{
		File:   p.Filename,
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
	var number strings.Builder
	isFloat := false

	// Parse integer part
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]
		if !unicode.IsDigit(rune(currentChar)) {
			break
		}
		number.WriteByte(currentChar)
		p.advance()
	}

	// Check for decimal point
	if p.Index < len(p.Source) && p.Source[p.Index] == '.' {
		// Look ahead to make sure there's a digit after the dot
		// (to avoid treating "5.method()" as a float)
		if p.Index+1 < len(p.Source) && unicode.IsDigit(rune(p.Source[p.Index+1])) {
			isFloat = true
			number.WriteByte('.')
			p.advance()

			// Parse fractional part
			for p.Index < len(p.Source) {
				currentChar := p.Source[p.Index]
				if !unicode.IsDigit(rune(currentChar)) {
					break
				}
				number.WriteByte(currentChar)
				p.advance()
			}
		}
	}

	// Check for exponent (e or E)
	if p.Index < len(p.Source) && (p.Source[p.Index] == 'e' || p.Source[p.Index] == 'E') {
		isFloat = true
		number.WriteByte(p.Source[p.Index])
		p.advance()

		// Check for optional sign
		if p.Index < len(p.Source) && (p.Source[p.Index] == '+' || p.Source[p.Index] == '-') {
			number.WriteByte(p.Source[p.Index])
			p.advance()
		}

		// Parse exponent digits
		hasExponentDigits := false
		for p.Index < len(p.Source) {
			currentChar := p.Source[p.Index]
			if !unicode.IsDigit(rune(currentChar)) {
				break
			}
			number.WriteByte(currentChar)
			p.advance()
			hasExponentDigits = true
		}

		if !hasExponentDigits {
			p.addErrorAt("invalid float literal: exponent has no digits", startPos)
			return
		}
	}

	tokenType := TokenTypeInteger
	if isFloat {
		tokenType = TokenTypeFloat
	}

	p.Tokens = append(p.Tokens, Token{
		Type:  tokenType,
		Value: number.String(),
		Pos:   startPos,
	})
}

func (p *lexer) ParseString() {
	startPos := p.currentPos()
	// Skip opening quote
	p.advance()

	var str strings.Builder
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Check for closing quote
		if currentChar == '"' {
			p.advance() // Skip closing quote
			p.Tokens = append(p.Tokens, Token{
				Type:  TokenTypeString,
				Value: str.String(),
				Pos:   startPos,
			})
			return
		}

		// Handle escape sequences
		if currentChar == '\\' {
			p.advance()
			// Check bounds after advancing (fixes potential out-of-bounds read)
			if p.Index >= len(p.Source) {
				p.addErrorAt("unterminated escape sequence", startPos)
				return
			}
			nextChar := p.Source[p.Index]
			switch nextChar {
			case 'n':
				str.WriteByte('\n')
			case 't':
				str.WriteByte('\t')
			case 'r':
				str.WriteByte('\r')
			case '\\':
				str.WriteByte('\\')
			case '"':
				str.WriteByte('"')
			default:
				// Unknown escape sequence, just include the backslash
				str.WriteByte('\\')
				str.WriteByte(nextChar)
			}
			p.advance()
		} else {
			str.WriteByte(currentChar)
			p.advance()
		}
	}

	// If we reach here, the string wasn't closed
	p.addErrorAt("unterminated string literal", startPos)
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
	var ident strings.Builder
	isFirst := true
	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Identifiers start with a letter, then can contain letters, digits, or underscores
		if isFirst {
			// First character must be a letter
			if !unicode.IsLetter(rune(currentChar)) {
				break
			}
			isFirst = false
		} else {
			// Subsequent characters can be letters, digits, or underscores
			if !unicode.IsLetter(rune(currentChar)) && !unicode.IsDigit(rune(currentChar)) && currentChar != '_' {
				break
			}
		}

		ident.WriteByte(currentChar)
		p.advance()
	}

	identStr := ident.String()

	// Check if it's a recognized keyword, otherwise it's an identifier
	tokenType := TokenTypeIdentifier
	if kwType, ok := keywords[identStr]; ok {
		tokenType = kwType
	}

	p.Tokens = append(p.Tokens, Token{
		Type:  tokenType,
		Value: identStr,
		Pos:   startPos,
	})
}

func (p *lexer) Parse() {
	for p.Index < len(p.Source) {
		b := p.Source[p.Index]

		if b == '\n' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeNewline, Value: "\n", Pos: pos})
			p.advance()
		} else if unicode.IsSpace(rune(b)) {
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
		} else if b == '[' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeLBracket, Value: "[", Pos: pos})
			p.advance()
		} else if b == ']' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeRBracket, Value: "]", Pos: pos})
			p.advance()
		} else if b == ',' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeComma, Value: ",", Pos: pos})
			p.advance()
		} else if b == ':' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeColon, Value: ":", Pos: pos})
			p.advance()
		} else if b == ';' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeSemicolon, Value: ";", Pos: pos})
			p.advance()
		} else if b == '+' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypePlus, Value: "+", Pos: pos})
			p.advance()
		} else if b == '-' {
			pos := p.currentPos()
			// Check for -> (arrow)
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '>' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeArrow, Value: "->", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeMinus, Value: "-", Pos: pos})
				p.advance()
			}
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
		} else if b == '.' {
			pos := p.currentPos()
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeDot, Value: ".", Pos: pos})
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
			// Check for != or standalone !
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '=' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeNotEqual, Value: "!=", Pos: pos})
				p.advance()
				p.advance()
			} else {
				// Standalone ! (logical NOT)
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeNot, Value: "!", Pos: pos})
				p.advance()
			}
		} else if b == '&' {
			// Check for &&
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '&' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeAnd, Value: "&&", Pos: pos})
				p.advance()
				p.advance()
			} else {
				// Single & for &T borrow type syntax
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeAmpersand, Value: "&", Pos: pos})
				p.advance()
			}
		} else if b == '|' {
			// Check for ||
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '|' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeOr, Value: "||", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.addError(fmt.Sprintf("unexpected character: %q (bitwise | not supported, use || for logical OR)", b))
				p.advance() // skip invalid character and continue lexing
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
		} else if b == '?' {
			// Check for ?: (elvis), ?. (safe call), or standalone ? (nullable type)
			pos := p.currentPos()
			if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == ':' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeElvis, Value: "?:", Pos: pos})
				p.advance()
				p.advance()
			} else if p.Index+1 < len(p.Source) && p.Source[p.Index+1] == '.' {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeSafeCall, Value: "?.", Pos: pos})
				p.advance()
				p.advance()
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeQuestion, Value: "?", Pos: pos})
				p.advance()
			}
		} else {
			p.addError(fmt.Sprintf("unexpected character: %q", b))
			p.advance() // skip invalid character and continue lexing
		}
	}
}

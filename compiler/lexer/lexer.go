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

	// Interpolated string tokens. A plain string is still a single
	// TokenTypeString; these are only emitted when a string contains a
	// $-interpolation. The stream shape is:
	//   StrChunk (InterpStart <expr tokens> InterpEnd StrChunk)+
	TokenTypeStrChunk    // literal text segment of an interpolated string
	TokenTypeInterpStart // '${' or '$' beginning an interpolation
	TokenTypeInterpEnd   // end of an interpolation
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
	case TokenTypeStrChunk:
		return "STR_CHUNK"
	case TokenTypeInterpStart:
		return "INTERP_START"
	case TokenTypeInterpEnd:
		return "INTERP_END"
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

	// interpStack tracks suspended interpolated-string scans. Each frame is
	// pushed when a "${" is opened (ParseString hands control back to the main
	// loop to tokenize the embedded expression) and popped when the matching
	// "}" is seen, at which point string scanning resumes. braceDepth tracks
	// nested "{...}" within the interpolation so the correct "}" closes it.
	interpStack []interpFrame
}

// interpFrame is one level of suspended interpolated-string scanning.
type interpFrame struct {
	braceDepth int          // nested brace depth inside the interpolation
	startPos   ast.Position // start position of the enclosing string literal
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

// ParseString scans a string literal beginning at the opening quote. A string
// with no $-interpolation produces a single TokenTypeString (unchanged
// behavior). A string containing "${expr}" or "$name" produces an interpolation
// token stream: StrChunk (InterpStart <expr tokens> InterpEnd StrChunk)+.
func (p *lexer) ParseString() {
	startPos := p.currentPos()
	// Skip opening quote
	p.advance()
	p.scanStringChunk(startPos, false)
}

// scanStringChunk scans literal text of a string from the current position
// (just past the opening quote, or just past a closing "}" when resuming after
// an interpolation) up to the closing quote or the next interpolation.
//
// interpolated reports whether an interpolation has already been emitted for
// this string. When true, the closing quote emits a final StrChunk so the
// stream is well-formed; when false and no interpolation is found, a single
// TokenTypeString is emitted to preserve the plain-string fast path.
//
// On encountering "${" it emits StrChunk + InterpStart, pushes an interpFrame,
// and returns so the main Parse loop tokenizes the embedded expression; the
// matching "}" resumes scanning by calling this method again.
func (p *lexer) scanStringChunk(startPos ast.Position, interpolated bool) {
	chunkPos := p.currentPos()
	var str strings.Builder

	emitChunk := func() {
		p.Tokens = append(p.Tokens, Token{Type: TokenTypeStrChunk, Value: str.String(), Pos: chunkPos})
	}

	for p.Index < len(p.Source) {
		currentChar := p.Source[p.Index]

		// Closing quote ends the string.
		if currentChar == '"' {
			p.advance()
			if interpolated {
				emitChunk()
			} else {
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeString, Value: str.String(), Pos: startPos})
			}
			return
		}

		// Interpolation: "${" (expression) or "$name" (bare identifier).
		if currentChar == '$' && p.Index+1 < len(p.Source) {
			next := p.Source[p.Index+1]
			if next == '{' {
				emitChunk()
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeInterpStart, Value: "${", Pos: p.currentPos()})
				p.advance() // skip '$'
				p.advance() // skip '{'
				p.interpStack = append(p.interpStack, interpFrame{braceDepth: 0, startPos: startPos})
				return
			}
			if unicode.IsLetter(rune(next)) {
				emitChunk()
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeInterpStart, Value: "$", Pos: p.currentPos()})
				p.advance() // skip '$'
				p.scanInterpIdentifier()
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeInterpEnd, Value: "", Pos: p.currentPos()})
				// Continue scanning the rest of this string in-place.
				interpolated = true
				chunkPos = p.currentPos()
				str.Reset()
				continue
			}
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
			case '$':
				str.WriteByte('$')
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

// scanInterpIdentifier emits a single identifier token for a "$name" shorthand
// interpolation. The leading '$' has already been consumed.
func (p *lexer) scanInterpIdentifier() {
	startPos := p.currentPos()
	var ident strings.Builder
	for p.Index < len(p.Source) {
		c := p.Source[p.Index]
		if !unicode.IsLetter(rune(c)) && !unicode.IsDigit(rune(c)) && c != '_' {
			break
		}
		ident.WriteByte(c)
		p.advance()
	}
	p.Tokens = append(p.Tokens, Token{Type: TokenTypeIdentifier, Value: ident.String(), Pos: startPos})
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

// isContinuationLeft reports whether a token appearing at the end of a line
// means the statement continues on the next line, so the following newline is
// not a statement separator (e.g. `a +` <newline> `b`).
func isContinuationLeft(t TokenType) bool {
	switch t {
	case TokenTypePlus, TokenTypeMinus, TokenTypeMultiply, TokenTypeDivide, TokenTypeModulo,
		TokenTypeAssign, TokenTypeEqual, TokenTypeNotEqual,
		TokenTypeLessThan, TokenTypeGreaterThan, TokenTypeLessThanOrEqual, TokenTypeGreaterThanOrEqual,
		TokenTypeAnd, TokenTypeOr, TokenTypeComma, TokenTypeDot, TokenTypeArrow,
		TokenTypeElvis, TokenTypeSafeCall, TokenTypeColon, TokenTypeQuestion,
		TokenTypeLParen, TokenTypeLBracket:
		return true
	}
	return false
}

// isContinuationRight reports whether a token appearing at the start of a line
// continues the previous line, because a binary/postfix operator can never
// begin a statement (e.g. `a` <newline> `+ b`).
func isContinuationRight(t TokenType) bool {
	switch t {
	case TokenTypePlus, TokenTypeMinus, TokenTypeMultiply, TokenTypeDivide, TokenTypeModulo,
		TokenTypeEqual, TokenTypeNotEqual,
		TokenTypeLessThan, TokenTypeGreaterThan, TokenTypeLessThanOrEqual, TokenTypeGreaterThanOrEqual,
		TokenTypeAnd, TokenTypeOr, TokenTypeDot, TokenTypeElvis, TokenTypeSafeCall:
		return true
	}
	return false
}

// joinContinuationLines drops newline tokens that are line continuations rather
// than statement separators: newlines inside () or [], newlines after a
// trailing operator, and newlines before a leading binary operator. Braces ({})
// do not suppress newlines, since blocks rely on them as separators.
func joinContinuationLines(tokens []Token) []Token {
	out := make([]Token, 0, len(tokens))
	depth := 0
	for i := range tokens {
		tok := tokens[i]
		switch tok.Type {
		case TokenTypeLParen, TokenTypeLBracket:
			depth++
		case TokenTypeRParen, TokenTypeRBracket:
			if depth > 0 {
				depth--
			}
		}
		if tok.Type == TokenTypeNewline {
			if depth > 0 {
				continue
			}
			if n := len(out); n > 0 && isContinuationLeft(out[n-1].Type) {
				continue
			}
			j := i + 1
			for j < len(tokens) && tokens[j].Type == TokenTypeNewline {
				j++
			}
			if j < len(tokens) && isContinuationRight(tokens[j].Type) {
				continue
			}
		}
		out = append(out, tok)
	}
	return out
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
			if n := len(p.interpStack); n > 0 {
				p.interpStack[n-1].braceDepth++
			}
			p.Tokens = append(p.Tokens, Token{Type: TokenTypeLBrace, Value: "{", Pos: pos})
			p.advance()
		} else if b == '}' {
			pos := p.currentPos()
			if n := len(p.interpStack); n > 0 && p.interpStack[n-1].braceDepth == 0 {
				// This "}" closes the active interpolation: emit InterpEnd and
				// resume scanning the enclosing string literal.
				frame := p.interpStack[n-1]
				p.interpStack = p.interpStack[:n-1]
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeInterpEnd, Value: "}", Pos: pos})
				p.advance() // skip '}'
				p.scanStringChunk(frame.startPos, true)
			} else {
				if n > 0 {
					p.interpStack[n-1].braceDepth--
				}
				p.Tokens = append(p.Tokens, Token{Type: TokenTypeRBrace, Value: "}", Pos: pos})
				p.advance()
			}
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

	p.Tokens = joinContinuationLines(p.Tokens)
}

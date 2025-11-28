package slasm

// Parser parses tokens into an intermediate representation (IR)
type Parser struct {
	tokens  []Token
	current int
}

// NewParser creates a new parser for the given tokens
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

// Parse parses the tokens into a Program IR
func (p *Parser) Parse() (*Program, error) {
	// TODO: Implement parsing
	// 1. Parse directives (.data, .text, .global, .align, etc.)
	// 2. Parse labels (identifier followed by :)
	// 3. Parse instructions with operands
	// 4. Build Program IR with sections
	return nil, nil
}

// Helper methods for parsing

func (p *Parser) peek() Token {
	if p.current >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.current]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	if p.current < len(p.tokens) {
		p.current++
	}
	return tok
}

func (p *Parser) expect(tokenType TokenType) (Token, error) {
	tok := p.peek()
	if tok.Type != tokenType {
		// TODO: Better error message
		return tok, nil
	}
	return p.advance(), nil
}

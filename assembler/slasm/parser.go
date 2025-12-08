package slasm

import "fmt"

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
	program := &Program{
		Sections: []*Section{},
	}

	// Start with a default text section
	currentSection := &Section{
		Type:  SectionText,
		Items: []Item{},
	}

	for p.peek().Type != TokenEOF {
		// Skip newlines and comments
		if p.peek().Type == TokenNewline || p.peek().Type == TokenComment {
			p.advance()
			continue
		}

		// Parse directives
		if p.peek().Type == TokenDirective {
			// Check if it's a data directive
			directiveName := p.peek().Value
			if len(directiveName) > 0 && directiveName[0] == '.' {
				directiveName = directiveName[1:]
			}

			// Section switching directives (peek without consuming)
			if directiveName == "text" {
				p.advance() // consume the directive
				if len(currentSection.Items) > 0 {
					program.Sections = append(program.Sections, currentSection)
				}
				currentSection = &Section{Type: SectionText, Items: []Item{}}
				continue
			} else if directiveName == "data" {
				p.advance() // consume the directive
				if len(currentSection.Items) > 0 {
					program.Sections = append(program.Sections, currentSection)
				}
				currentSection = &Section{Type: SectionData, Items: []Item{}}
				continue
			}

			// Data declaration directives
			if isDataDirective(directiveName) {
				dataDecl, err := p.parseDataDirective()
				if err != nil {
					return nil, err
				}
				currentSection.Items = append(currentSection.Items, dataDecl)
				continue
			}

			// Other directives (like .global, .align)
			directive, err := p.parseDirective()
			if err != nil {
				return nil, err
			}
			currentSection.Items = append(currentSection.Items, directive)
			continue
		}

		// Parse labels
		if p.peek().Type == TokenIdentifier && p.peekAhead(1).Type == TokenColon {
			label := p.parseLabel()
			currentSection.Items = append(currentSection.Items, label)
			continue
		}

		// Parse constant assignments: identifier = value
		if p.peek().Type == TokenIdentifier && p.peekAhead(1).Type == TokenEquals {
			constDef, err := p.parseConstantDef()
			if err != nil {
				return nil, err
			}
			currentSection.Items = append(currentSection.Items, constDef)
			continue
		}

		// Parse instructions
		if p.peek().Type == TokenIdentifier {
			inst, err := p.parseInstruction()
			if err != nil {
				return nil, err
			}
			currentSection.Items = append(currentSection.Items, inst)
			continue
		}

		// Unknown token - return error instead of silently skipping
		tok := p.peek()
		return nil, fmt.Errorf("line %d:%d: unexpected token %v '%s'", tok.Line, tok.Column, tok.Type, tok.Value)
	}

	// Add the last section
	if len(currentSection.Items) > 0 {
		program.Sections = append(program.Sections, currentSection)
	}

	return program, nil
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
		return tok, fmt.Errorf("expected token type %v, got %v", tokenType, tok.Type)
	}
	return p.advance(), nil
}

func (p *Parser) peekAhead(n int) Token {
	pos := p.current + n
	if pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[pos]
}

func (p *Parser) parseDirective() (*Directive, error) {
	tok := p.advance() // consume directive token

	// Remove the leading .
	name := tok.Value
	if len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}

	directive := &Directive{
		Name:   name,
		Args:   []string{},
		Line:   tok.Line,
		Column: tok.Column,
	}

	// Parse arguments until newline or EOF
	for p.peek().Type != TokenNewline && p.peek().Type != TokenEOF && p.peek().Type != TokenComment {
		tok := p.advance()
		if tok.Type != TokenComma {
			directive.Args = append(directive.Args, tok.Value)
		}
	}

	return directive, nil
}

// isDataDirective returns true if the directive name is a data declaration directive
func isDataDirective(name string) bool {
	switch name {
	case "byte", "2byte", "4byte", "8byte", "quad", "word", "hword", "space", "zero", "asciz", "ascii", "string":
		return true
	}
	return false
}

// parseDataDirective parses a data directive and returns a DataDeclaration
func (p *Parser) parseDataDirective() (*DataDeclaration, error) {
	tok := p.advance() // consume directive token

	// Remove the leading .
	name := tok.Value
	if len(name) > 0 && name[0] == '.' {
		name = name[1:]
	}

	decl := &DataDeclaration{
		Type: name,
	}

	// Parse the value based on directive type
	switch name {
	case "asciz", "ascii", "string":
		// String literal - expect a quoted string
		if p.peek().Type == TokenString {
			decl.Value = p.advance().Value
		} else {
			return nil, fmt.Errorf("expected string literal after .%s", name)
		}

	case "byte", "2byte", "4byte", "8byte", "quad", "word", "hword":
		// Integer values - can be comma-separated
		var values []string
		for p.peek().Type != TokenNewline && p.peek().Type != TokenEOF && p.peek().Type != TokenComment {
			if p.peek().Type == TokenComma {
				p.advance()
				continue
			}
			values = append(values, p.advance().Value)
		}
		// Store all values, comma-separated for multiple values
		if len(values) > 0 {
			decl.Value = values[0]
			for i := 1; i < len(values); i++ {
				decl.Value += "," + values[i]
			}
		}

	case "space", "zero":
		// Size in bytes
		if p.peek().Type == TokenInteger {
			decl.Value = p.advance().Value
		} else {
			return nil, fmt.Errorf("expected integer after .%s", name)
		}
	}

	return decl, nil
}

func (p *Parser) parseLabel() *Label {
	tok := p.advance() // consume identifier
	p.advance()        // consume colon

	return &Label{
		Name:   tok.Value,
		Line:   tok.Line,
		Column: tok.Column,
	}
}

func (p *Parser) parseConstantDef() (*ConstantDef, error) {
	nameTok := p.advance() // consume identifier
	p.advance()            // consume =

	if p.peek().Type != TokenInteger {
		return nil, fmt.Errorf("line %d:%d: expected integer value after '='",
			nameTok.Line, nameTok.Column)
	}

	valueTok := p.advance()
	value, err := ParseInt64(valueTok.Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: invalid constant value: %w",
			valueTok.Line, valueTok.Column, err)
	}

	return &ConstantDef{
		Name:   nameTok.Value,
		Value:  value,
		Line:   nameTok.Line,
		Column: nameTok.Column,
	}, nil
}

func (p *Parser) parseInstruction() (*Instruction, error) {
	// Get instruction mnemonic
	mnemonic := p.advance()

	inst := &Instruction{
		Mnemonic: mnemonic.Value,
		Operands: []*Operand{},
		Line:     mnemonic.Line,
		Column:   mnemonic.Column,
	}

	// Parse operands until newline or EOF
	for p.peek().Type != TokenNewline && p.peek().Type != TokenEOF && p.peek().Type != TokenComment {
		// Skip commas
		if p.peek().Type == TokenComma {
			p.advance()
			continue
		}

		operand, err := p.parseOperand()
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: %w", mnemonic.Line, mnemonic.Column, err)
		}
		inst.Operands = append(inst.Operands, operand)
	}

	return inst, nil
}

func (p *Parser) parseOperand() (*Operand, error) {
	tok := p.peek()

	switch tok.Type {
	case TokenRegister:
		p.advance()
		return &Operand{
			Type:  OperandRegister,
			Value: tok.Value,
		}, nil

	case TokenHash:
		// Immediate value: #42
		p.advance() // consume #
		valueTok := p.advance()
		return &Operand{
			Type:  OperandImmediate,
			Value: valueTok.Value,
		}, nil

	case TokenIdentifier:
		// Could be a shift modifier (lsl, asr), label reference, or @PAGE/@PAGEOFF
		p.advance()

		// Check if it's a shift modifier (lsl #N, asr #N, etc.)
		if (tok.Value == "lsl" || tok.Value == "asr" || tok.Value == "lsr") && p.peek().Type == TokenHash {
			shiftType := tok.Value
			p.advance() // consume #
			shiftAmount := p.advance()
			return &Operand{
				Type:      OperandShift,
				Value:     shiftAmount.Value,
				ShiftType: shiftType,
			}, nil
		}

		operand := &Operand{
			Type:  OperandLabel,
			Value: tok.Value,
		}

		// Check for @PAGE or @PAGEOFF
		if p.peek().Type == TokenAt {
			p.advance() // consume @
			modifier := p.advance()
			operand.Value = tok.Value + "@" + modifier.Value
		}

		return operand, nil

	case TokenLBracket:
		// Memory operand: [base, #offset], [base, #offset]!, or [base]
		p.advance() // consume [

		base := p.advance()
		operand := &Operand{
			Type: OperandMemory,
			Base: base.Value,
		}

		// Check for offset inside brackets
		if p.peek().Type == TokenComma {
			p.advance() // consume comma
			if p.peek().Type == TokenHash {
				p.advance() // consume #
			}
			offset := p.advance()
			operand.Offset = offset.Value
		}

		if p.peek().Type == TokenRBracket {
			p.advance() // consume ]
		}

		// Check for pre-indexed writeback: [base, #offset]!
		if p.peek().Type == TokenExclamation {
			p.advance() // consume !
			operand.Writeback = true
		}

		// Check for post-indexed: [base], #offset
		// The comma after ] indicates post-indexed mode
		if p.peek().Type == TokenComma {
			p.advance() // consume comma
			if p.peek().Type == TokenHash {
				p.advance() // consume #
			}
			postOffset := p.advance()
			operand.PostIndexOffset = postOffset.Value
		}

		return operand, nil

	default:
		return nil, fmt.Errorf("unexpected token in operand: %v", tok.Type)
	}
}

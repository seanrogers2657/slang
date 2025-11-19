package parser

import (
	"fmt"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/lexer"
)

// Precedence levels for operators (higher = tighter binding)
type precedence int

const (
	precedenceLowest precedence = iota
	precedenceComparison        // ==, !=, <, >, <=, >=
	precedenceSum               // +, -
	precedenceProduct           // *, /, %
	precedencePrefix            // -x, !x
)

func NewParser(source []lexer.Token) *parser {
	parser := &parser{
		Source: source,
		Index:  0,
	}

	return parser
}

type parser struct {
	Source []lexer.Token
	Index  int

	Errors []error
}

// getPrecedence returns the precedence level for the current token
func (p *parser) getPrecedence(tokenType lexer.TokenType) precedence {
	switch tokenType {
	case lexer.TokenTypeEqual, lexer.TokenTypeNotEqual,
		lexer.TokenTypeLessThan, lexer.TokenTypeGreaterThan,
		lexer.TokenTypeLessThanOrEqual, lexer.TokenTypeGreaterThanOrEqual:
		return precedenceComparison
	case lexer.TokenTypePlus, lexer.TokenTypeMinus:
		return precedenceSum
	case lexer.TokenTypeMultiply, lexer.TokenTypeDivide, lexer.TokenTypeModulo:
		return precedenceProduct
	default:
		return precedenceLowest
	}
}

func (p *parser) PreviousToken() lexer.Token {
	return p.Source[p.Index-1]
}

func (p *parser) CurrentToken() lexer.Token {
	return p.Source[p.Index]
}

func (p *parser) isAtEnd() bool {
	return p.Index >= len(p.Source)
}

func (p *parser) skipNewlines() {
	for !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeNewline {
		p.Index++
	}
}

// advance consumes and returns the current token
func (p *parser) advance() lexer.Token {
	if p.isAtEnd() {
		return lexer.Token{}
	}
	token := p.CurrentToken()
	p.Index++
	return token
}

// peek returns the next token without consuming it
func (p *parser) peek() lexer.Token {
	if p.Index+1 >= len(p.Source) {
		return lexer.Token{}
	}
	return p.Source[p.Index+1]
}

// currentPrecedence returns the precedence of the current token
func (p *parser) currentPrecedence() precedence {
	if p.isAtEnd() {
		return precedenceLowest
	}
	return p.getPrecedence(p.CurrentToken().Type)
}

// getOperatorString converts a token type to its operator string
func (p *parser) getOperatorString(tokenType lexer.TokenType) string {
	switch tokenType {
	case lexer.TokenTypePlus:
		return "+"
	case lexer.TokenTypeMinus:
		return "-"
	case lexer.TokenTypeMultiply:
		return "*"
	case lexer.TokenTypeDivide:
		return "/"
	case lexer.TokenTypeModulo:
		return "%"
	case lexer.TokenTypeEqual:
		return "=="
	case lexer.TokenTypeNotEqual:
		return "!="
	case lexer.TokenTypeLessThan:
		return "<"
	case lexer.TokenTypeGreaterThan:
		return ">"
	case lexer.TokenTypeLessThanOrEqual:
		return "<="
	case lexer.TokenTypeGreaterThanOrEqual:
		return ">="
	default:
		return ""
	}
}

// Top level parsing
func (p *parser) Parse() *ast.Program {
	startPos := ast.Position{Line: 1, Column: 1, Offset: 0}
	if !p.isAtEnd() {
		startPos = p.CurrentToken().Pos
	}

	program := &ast.Program{
		Statements: []ast.Statement{},
		StartPos:   startPos,
	}

	// Skip leading newlines
	p.skipNewlines()

	for !p.isAtEnd() {
		stmt := p.ParseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}

		// After each statement, we expect a newline or end of input
		if !p.isAtEnd() {
			if p.CurrentToken().Type == lexer.TokenTypeNewline {
				p.Index++ // Consume the newline
				p.skipNewlines() // Skip any additional newlines
			} else {
				// Error: expected newline or end of input
				p.Errors = append(p.Errors, fmt.Errorf("expected newline after statement, got %s", p.CurrentToken().Value))
				break
			}
		}
	}

	// Set end position
	if len(program.Statements) > 0 {
		program.EndPos = program.Statements[len(program.Statements)-1].End()
	} else {
		program.EndPos = startPos
	}

	return program
}

func (p *parser) ParseStatement() ast.Statement {
	// Check if it's a print statement
	if p.CurrentToken().Type == lexer.TokenTypePrint {
		return p.ParsePrintStatement()
	}

	// Otherwise, it's an expression statement
	expr := p.parseExpression(precedenceLowest)
	if expr != nil {
		return &ast.ExprStmt{Expr: expr}
	}

	return nil
}

func (p *parser) ParsePrintStatement() ast.Statement {
	// Get position of 'print' keyword
	keywordPos := p.CurrentToken().Pos

	// Consume 'print' token
	p.advance()

	// Parse the expression to print using the Pratt parser
	expr := p.parseExpression(precedenceLowest)
	if expr == nil {
		p.Errors = append(p.Errors, fmt.Errorf("expected expression after 'print'"))
		return nil
	}

	return &ast.PrintStmt{
		Keyword: keywordPos,
		Expr:    expr,
	}
}

func (p *parser) ParseLiteral() ast.Expression {
	//spew.Dump("parsing literal")
	if p.CurrentToken().Type == lexer.TokenTypeInteger {
		token := p.CurrentToken()
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeNumber,
			Value:    token.Value,
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + len(token.Value), Offset: token.Pos.Offset + len(token.Value)},
		}
		p.Index++
		return literal
	}

	if p.CurrentToken().Type == lexer.TokenTypeString {
		token := p.CurrentToken()
		// String length includes the quotes, so add 2 for quote characters
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeString,
			Value:    token.Value,
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + len(token.Value) + 2, Offset: token.Pos.Offset + len(token.Value) + 2},
		}
		p.Index++
		return literal
	}

	return nil
}

// parseExpression implements Pratt parsing with operator precedence
func (p *parser) parseExpression(minPrec precedence) ast.Expression {
	// Parse prefix (primary expression)
	left := p.parsePrimary()
	if left == nil {
		return nil
	}

	// Parse infix (binary operators)
	for !p.isAtEnd() && p.currentPrecedence() > minPrec {
		// Stop at newlines (statement terminators)
		if p.CurrentToken().Type == lexer.TokenTypeNewline {
			break
		}

		tokenType := p.CurrentToken().Type
		op := p.getOperatorString(tokenType)
		if op == "" {
			break
		}

		opPos := p.CurrentToken().Pos
		opPrec := p.currentPrecedence()

		p.advance() // consume operator

		// For left-associative operators, we use opPrec for the right side
		// For right-associative, we would use opPrec - 1
		right := p.parseExpression(opPrec)
		if right == nil {
			p.Errors = append(p.Errors, fmt.Errorf("expected expression after operator '%s'", op))
			return nil
		}

		left = &ast.BinaryExpr{
			Left:     left,
			Op:       op,
			Right:    right,
			LeftPos:  left.Pos(),
			OpPos:    opPos,
			RightPos: right.Pos(),
		}
	}

	return left
}

// parsePrimary parses primary expressions (literals, grouping, etc.)
func (p *parser) parsePrimary() ast.Expression {
	return p.ParseLiteral()
}

// ParseBinaryExpression is kept for backward compatibility during transition
// It now delegates to the new Pratt parser
func (p *parser) ParseBinaryExpression() ast.Expression {
	expr := p.parseExpression(precedenceLowest)
	if expr == nil && len(p.Errors) == 0 {
		// Add error if no expression was parsed and no error was set
		p.Errors = append(p.Errors, fmt.Errorf("unsupported operation: %s", p.CurrentToken().Value))
	}
	return expr
}

package parser

import (
	"fmt"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/lexer"
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
	expr := p.ParseBinaryExpression()
	if expr != nil {
		return &ast.ExprStmt{Expr: expr}
	}

	return nil
}

func (p *parser) ParsePrintStatement() ast.Statement {
	// Get position of 'print' keyword
	keywordPos := p.CurrentToken().Pos

	// Consume 'print' token
	p.Index++

	// Parse the expression to print
	// First try to parse a literal
	left := p.ParseLiteral()
	if left == nil {
		p.Errors = append(p.Errors, fmt.Errorf("expected expression after 'print'"))
		return nil
	}

	// Check if there's an operator (binary expression)
	if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeNewline {
		// This is a binary expression
		tokenType := p.CurrentToken().Type
		var op string

		switch tokenType {
		case lexer.TokenTypePlus:
			op = "+"
		case lexer.TokenTypeMinus:
			op = "-"
		case lexer.TokenTypeMultiply:
			op = "*"
		case lexer.TokenTypeDivide:
			op = "/"
		case lexer.TokenTypeModulo:
			op = "%"
		case lexer.TokenTypeEqual:
			op = "=="
		case lexer.TokenTypeNotEqual:
			op = "!="
		case lexer.TokenTypeLessThan:
			op = "<"
		case lexer.TokenTypeGreaterThan:
			op = ">"
		case lexer.TokenTypeLessThanOrEqual:
			op = "<="
		case lexer.TokenTypeGreaterThanOrEqual:
			op = ">="
		default:
			// Not a binary operator, just a single literal
			return &ast.PrintStmt{
				Keyword: keywordPos,
				Expr:    left,
			}
		}

		// Get operator position
		opPos := p.CurrentToken().Pos

		// Consume operator
		p.Index++

		// Parse right operand
		right := p.ParseLiteral()
		if right == nil {
			p.Errors = append(p.Errors, fmt.Errorf("expected right operand"))
			return nil
		}

		binaryExpr := &ast.BinaryExpr{
			Left:     left,
			Op:       op,
			Right:    right,
			LeftPos:  left.Pos(),
			OpPos:    opPos,
			RightPos: right.Pos(),
		}

		return &ast.PrintStmt{
			Keyword: keywordPos,
			Expr:    binaryExpr,
		}
	}

	// Just a single literal
	return &ast.PrintStmt{
		Keyword: keywordPos,
		Expr:    left,
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

func (p *parser) ParseBinaryExpression() ast.Expression {
	left := p.ParseLiteral()

	tokenType := p.CurrentToken().Type
	var op string

	switch tokenType {
	case lexer.TokenTypePlus:
		op = "+"
	case lexer.TokenTypeMinus:
		op = "-"
	case lexer.TokenTypeMultiply:
		op = "*"
	case lexer.TokenTypeDivide:
		op = "/"
	case lexer.TokenTypeModulo:
		op = "%"
	case lexer.TokenTypeEqual:
		op = "=="
	case lexer.TokenTypeNotEqual:
		op = "!="
	case lexer.TokenTypeLessThan:
		op = "<"
	case lexer.TokenTypeGreaterThan:
		op = ">"
	case lexer.TokenTypeLessThanOrEqual:
		op = "<="
	case lexer.TokenTypeGreaterThanOrEqual:
		op = ">="
	default:
		newError := fmt.Errorf("unsupported operation: %s", p.CurrentToken().Value)
		p.Errors = append(p.Errors, newError)
		return nil
	}

	// Get operator position
	opPos := p.CurrentToken().Pos

	// consume operator
	p.Index++

	right := p.ParseLiteral()
	return &ast.BinaryExpr{
		Left:     left,
		Op:       op,
		Right:    right,
		LeftPos:  left.Pos(),
		OpPos:    opPos,
		RightPos: right.Pos(),
	}
}

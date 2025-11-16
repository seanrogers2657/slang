package parser

import (
	"fmt"

	"github.com/seanrogers2657/slang/frontend/lexer"
)

type LiteralType int

const (
	LiteralTypeNumber LiteralType = iota
	LiteralTypeString
)

type Literal struct {
	Type  LiteralType
	Value string
}

type Expr struct {
	Left  *Literal
	Op    string
	Right *Literal
}

// Statement represents any statement in the program
type Statement interface {
	statementNode()
}

// ExprStmt is a statement that consists of a single expression
type ExprStmt struct {
	Expr *Expr
}

func (e *ExprStmt) statementNode() {}

// PrintStmt is a print statement
type PrintStmt struct {
	Expr *Expr
}

func (p *PrintStmt) statementNode() {}

type Program struct {
	Statements []Statement
}

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
func (p *parser) Parse() *Program {
	program := &Program{
		Statements: []Statement{},
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

	return program
}

func (p *parser) ParseStatement() Statement {
	// Check if it's a print statement
	if p.CurrentToken().Type == lexer.TokenTypePrint {
		return p.ParsePrintStatement()
	}

	// Otherwise, it's an expression statement
	expr := p.ParseBinaryExpression()
	if expr != nil {
		return &ExprStmt{Expr: expr}
	}

	return nil
}

func (p *parser) ParsePrintStatement() Statement {
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
			return &PrintStmt{Expr: &Expr{Left: left, Op: "", Right: nil}}
		}

		// Consume operator
		p.Index++

		// Parse right operand
		right := p.ParseLiteral()
		if right == nil {
			p.Errors = append(p.Errors, fmt.Errorf("expected right operand"))
			return nil
		}

		return &PrintStmt{Expr: &Expr{Left: left, Op: op, Right: right}}
	}

	// Just a single literal
	return &PrintStmt{Expr: &Expr{Left: left, Op: "", Right: nil}}
}

func (p *parser) ParseLiteral() *Literal {
	//spew.Dump("parsing literal")
	if p.CurrentToken().Type == lexer.TokenTypeInteger {
		literal := Literal{
			Type:  LiteralTypeNumber,
			Value: p.CurrentToken().Value,
		}
		p.Index++
		return &literal
	}

	if p.CurrentToken().Type == lexer.TokenTypeString {
		literal := Literal{
			Type:  LiteralTypeString,
			Value: p.CurrentToken().Value,
		}
		p.Index++
		return &literal
	}

	return nil
}

func (p *parser) ParseBinaryExpression() *Expr {
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

	// consume operator
	p.Index++

	right := p.ParseLiteral()
	return &Expr{
		Left:  left,
		Op:    op,
		Right: right,
	}
}

package parser

import (
	"fmt"

	"github.com/seanrogers2657/slang/frontend/lexer"
)

type LiteralType int

const (
	LiteralTypeNumber LiteralType = iota
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

// Top level parsing
// TODO add main function parsing
func (p *parser) Parse() *Expr {
	expr := p.ParseBinaryExpression()
	return expr
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

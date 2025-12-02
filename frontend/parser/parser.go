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
		Declarations: []ast.Declaration{},
		Statements:   []ast.Statement{},
		StartPos:     startPos,
	}

	// Skip leading newlines
	p.skipNewlines()

	// Check if this is a function-based program or legacy statement-based program
	if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeFn {
		// New style: parse function declarations
		for !p.isAtEnd() {
			p.skipNewlines()
			if p.isAtEnd() {
				break
			}

			fnDecl := p.ParseFunctionDecl()
			if fnDecl != nil {
				program.Declarations = append(program.Declarations, fnDecl)
			}

			// Skip newlines after function declaration
			p.skipNewlines()
		}

		// Set end position
		if len(program.Declarations) > 0 {
			program.EndPos = program.Declarations[len(program.Declarations)-1].End()
		} else {
			program.EndPos = startPos
		}
	} else {
		// Legacy style: parse top-level statements
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
	}

	return program
}

func (p *parser) ParseStatement() ast.Statement {
	// Check if it's a print statement
	if p.CurrentToken().Type == lexer.TokenTypePrint {
		return p.ParsePrintStatement()
	}

	// Check if it's a variable declaration
	if p.CurrentToken().Type == lexer.TokenTypeVal {
		return p.ParseVarDecl()
	}

	// Otherwise, it's an expression statement
	expr := p.parseExpression(precedenceLowest)
	if expr != nil {
		return &ast.ExprStmt{Expr: expr}
	}

	return nil
}

// ParseVarDecl parses a variable declaration: val <name> = <expr>
func (p *parser) ParseVarDecl() ast.Statement {
	// Get position of 'val' keyword
	valKeyword := p.CurrentToken().Pos
	p.advance() // consume 'val'

	// Expect identifier
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.Errors = append(p.Errors, fmt.Errorf("expected identifier after 'val', got %s", p.CurrentToken().Value))
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect '='
	if p.CurrentToken().Type != lexer.TokenTypeAssign {
		p.Errors = append(p.Errors, fmt.Errorf("expected '=' after variable name, got %s", p.CurrentToken().Value))
		return nil
	}

	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	// Parse the initializer expression
	initializer := p.parseExpression(precedenceLowest)
	if initializer == nil {
		p.Errors = append(p.Errors, fmt.Errorf("expected expression after '='"))
		return nil
	}

	return &ast.VarDeclStmt{
		ValKeyword:  valKeyword,
		Name:        name,
		NamePos:     namePos,
		Equals:      equalsPos,
		Initializer: initializer,
	}
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

// parsePrimary parses primary expressions (literals, identifiers, grouping, etc.)
func (p *parser) parsePrimary() ast.Expression {
	// Check for identifier
	if p.CurrentToken().Type == lexer.TokenTypeIdentifier {
		token := p.CurrentToken()
		ident := &ast.IdentifierExpr{
			Name:     token.Value,
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + len(token.Value), Offset: token.Pos.Offset + len(token.Value)},
		}
		p.Index++
		return ident
	}

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

// ParseBlockStmt parses a block statement: { <statements> }
func (p *parser) ParseBlockStmt() *ast.BlockStmt {
	// Expect '{'
	if p.CurrentToken().Type != lexer.TokenTypeLBrace {
		p.Errors = append(p.Errors, fmt.Errorf("expected '{', got %s", p.CurrentToken().Value))
		return nil
	}

	leftBrace := p.CurrentToken().Pos
	p.advance() // consume '{'

	// Skip newlines after opening brace
	p.skipNewlines()

	statements := []ast.Statement{}

	// Parse statements until we hit '}'
	for !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRBrace {
		stmt := p.ParseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}

		// After each statement, expect newline or '}'
		if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRBrace {
			if p.CurrentToken().Type == lexer.TokenTypeNewline {
				p.advance() // consume newline
				p.skipNewlines() // skip any additional newlines
			} else {
				p.Errors = append(p.Errors, fmt.Errorf("expected newline or '}' after statement, got %s", p.CurrentToken().Value))
				break
			}
		}
	}

	// Expect '}'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRBrace {
		p.Errors = append(p.Errors, fmt.Errorf("expected '}' to close block"))
		return nil
	}

	rightBrace := p.CurrentToken().Pos
	p.advance() // consume '}'

	return &ast.BlockStmt{
		LeftBrace:  leftBrace,
		Statements: statements,
		RightBrace: rightBrace,
	}
}

// ParseFunctionDecl parses a function declaration: fn <name>() { <body> }
func (p *parser) ParseFunctionDecl() *ast.FunctionDecl {
	// Expect 'fn' keyword
	if p.CurrentToken().Type != lexer.TokenTypeFn {
		p.Errors = append(p.Errors, fmt.Errorf("expected 'fn', got %s", p.CurrentToken().Value))
		return nil
	}

	fnKeyword := p.CurrentToken().Pos
	p.advance() // consume 'fn'

	// Expect identifier (function name)
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.Errors = append(p.Errors, fmt.Errorf("expected function name, got %s", p.CurrentToken().Value))
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect '('
	if p.CurrentToken().Type != lexer.TokenTypeLParen {
		p.Errors = append(p.Errors, fmt.Errorf("expected '(' after function name, got %s", p.CurrentToken().Value))
		return nil
	}

	leftParen := p.CurrentToken().Pos
	p.advance() // consume '('

	// For now, we don't support parameters, so expect ')'
	if p.CurrentToken().Type != lexer.TokenTypeRParen {
		p.Errors = append(p.Errors, fmt.Errorf("expected ')' (parameters not yet supported), got %s", p.CurrentToken().Value))
		return nil
	}

	rightParen := p.CurrentToken().Pos
	p.advance() // consume ')'

	// Skip newlines before body
	p.skipNewlines()

	// Parse function body (block statement)
	body := p.ParseBlockStmt()
	if body == nil {
		return nil
	}

	return &ast.FunctionDecl{
		FnKeyword:  fnKeyword,
		Name:       name,
		NamePos:    namePos,
		LeftParen:  leftParen,
		RightParen: rightParen,
		Body:       body,
	}
}

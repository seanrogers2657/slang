package parser

import (
	"fmt"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/lexer"
)

// Precedence levels for operators (higher = tighter binding)
type precedence int

const (
	precedenceLowest     precedence = iota
	precedenceOr                    // ||
	precedenceAnd                   // &&
	precedenceComparison            // ==, !=, <, >, <=, >=
	precedenceSum                   // +, -
	precedenceProduct               // *, /, %
	precedencePrefix                // -x, !x
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
	case lexer.TokenTypeOr:
		return precedenceOr
	case lexer.TokenTypeAnd:
		return precedenceAnd
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
	case lexer.TokenTypeAnd:
		return "&&"
	case lexer.TokenTypeOr:
		return "||"
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
			} else {
				// If parsing failed, break to avoid infinite loop
				break
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
					p.Index++        // Consume the newline
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
	// Check if it's a return statement
	if p.CurrentToken().Type == lexer.TokenTypeReturn {
		return p.ParseReturnStatement()
	}

	// Check if it's an immutable variable declaration (val)
	if p.CurrentToken().Type == lexer.TokenTypeVal {
		return p.ParseVarDecl(false)
	}

	// Check if it's a mutable variable declaration (var)
	if p.CurrentToken().Type == lexer.TokenTypeVar {
		return p.ParseVarDecl(true)
	}

	// Check for assignment: identifier followed by '='
	if p.CurrentToken().Type == lexer.TokenTypeIdentifier {
		if p.peek().Type == lexer.TokenTypeAssign {
			return p.ParseAssignment()
		}
	}

	// Otherwise, it's an expression statement
	expr := p.parseExpression(precedenceLowest)
	if expr != nil {
		return &ast.ExprStmt{Expr: expr}
	}

	return nil
}

// ParseVarDecl parses a variable declaration: val <name> = <expr> or val <name>: <type> = <expr>
func (p *parser) ParseVarDecl(mutable bool) ast.Statement {
	// Get position of 'val' or 'var' keyword
	keyword := p.CurrentToken().Pos
	keywordName := p.CurrentToken().Value
	p.advance() // consume 'val' or 'var'

	// Expect identifier
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.Errors = append(p.Errors, fmt.Errorf("expected identifier after '%s', got %s", keywordName, p.CurrentToken().Value))
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Check for optional type annotation
	var colonPos ast.Position
	var typeName string
	var typePos ast.Position

	if p.CurrentToken().Type == lexer.TokenTypeColon {
		colonPos = p.CurrentToken().Pos
		p.advance() // consume ':'

		// Expect type identifier
		if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
			p.Errors = append(p.Errors, fmt.Errorf("expected type after ':', got %s", p.CurrentToken().Value))
			return nil
		}

		typeName = p.CurrentToken().Value
		typePos = p.CurrentToken().Pos
		p.advance() // consume type
	}

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
		Keyword:     keyword,
		Mutable:     mutable,
		Name:        name,
		NamePos:     namePos,
		Colon:       colonPos,
		TypeName:    typeName,
		TypePos:     typePos,
		Equals:      equalsPos,
		Initializer: initializer,
	}
}

// ParseAssignment parses a variable assignment: <name> = <expr>
func (p *parser) ParseAssignment() ast.Statement {
	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	value := p.parseExpression(precedenceLowest)
	if value == nil {
		p.Errors = append(p.Errors, fmt.Errorf("expected expression after '='"))
		return nil
	}

	return &ast.AssignStmt{
		Name:    name,
		NamePos: namePos,
		Equals:  equalsPos,
		Value:   value,
	}
}

// ParseReturnStatement parses a return statement: return <expr>
func (p *parser) ParseReturnStatement() ast.Statement {
	// Get position of 'return' keyword
	keywordPos := p.CurrentToken().Pos
	p.advance() // consume 'return'

	// Check if there's a value to return (not newline or closing brace)
	var value ast.Expression
	if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeNewline && p.CurrentToken().Type != lexer.TokenTypeRBrace {
		value = p.parseExpression(precedenceLowest)
	}

	return &ast.ReturnStmt{
		Keyword: keywordPos,
		Value:   value,
	}
}

func (p *parser) ParseLiteral() ast.Expression {
	//spew.Dump("parsing literal")
	if p.CurrentToken().Type == lexer.TokenTypeInteger {
		token := p.CurrentToken()
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeInteger,
			Value:    token.Value,
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + len(token.Value), Offset: token.Pos.Offset + len(token.Value)},
		}
		p.Index++
		return literal
	}

	if p.CurrentToken().Type == lexer.TokenTypeFloat {
		token := p.CurrentToken()
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeFloat,
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

	if p.CurrentToken().Type == lexer.TokenTypeTrue {
		token := p.CurrentToken()
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeBoolean,
			Value:    "true",
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + 4, Offset: token.Pos.Offset + 4},
		}
		p.Index++
		return literal
	}

	if p.CurrentToken().Type == lexer.TokenTypeFalse {
		token := p.CurrentToken()
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeBoolean,
			Value:    "false",
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + 5, Offset: token.Pos.Offset + 5},
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
	// Check for unary NOT operator
	if p.CurrentToken().Type == lexer.TokenTypeNot {
		opPos := p.CurrentToken().Pos
		p.advance() // consume '!'

		// Parse the operand (recursively call parsePrimary for highest precedence)
		operand := p.parsePrimary()
		if operand == nil {
			p.Errors = append(p.Errors, fmt.Errorf("expected expression after '!'"))
			return nil
		}

		return &ast.UnaryExpr{
			Op:         "!",
			Operand:    operand,
			OpPos:      opPos,
			OperandPos: operand.Pos(),
			OperandEnd: operand.End(),
		}
	}

	// Check for identifier (could be variable reference or function call)
	if p.CurrentToken().Type == lexer.TokenTypeIdentifier {
		token := p.CurrentToken()
		p.advance() // consume identifier

		// Check if this is a function call
		if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeLParen {
			return p.parseCallExpr(token.Value, token.Pos)
		}

		// Otherwise it's just an identifier
		return &ast.IdentifierExpr{
			Name:     token.Value,
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + len(token.Value), Offset: token.Pos.Offset + len(token.Value)},
		}
	}

	return p.ParseLiteral()
}

// parseCallExpr parses a function call after the identifier has been consumed
func (p *parser) parseCallExpr(name string, namePos ast.Position) ast.Expression {
	leftParen := p.CurrentToken().Pos
	p.advance() // consume '('

	arguments := []ast.Expression{}

	// Parse arguments
	if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRParen {
		// Parse first argument
		arg := p.parseExpression(precedenceLowest)
		if arg != nil {
			arguments = append(arguments, arg)
		}

		// Parse remaining arguments
		for !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeComma {
			p.advance() // consume ','
			arg := p.parseExpression(precedenceLowest)
			if arg != nil {
				arguments = append(arguments, arg)
			}
		}
	}

	// Expect ')'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRParen {
		p.Errors = append(p.Errors, fmt.Errorf("expected ')' after function arguments, got %s", p.CurrentToken().Value))
		return nil
	}

	rightParen := p.CurrentToken().Pos
	p.advance() // consume ')'

	return &ast.CallExpr{
		Name:       name,
		NamePos:    namePos,
		LeftParen:  leftParen,
		Arguments:  arguments,
		RightParen: rightParen,
	}
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
				p.advance()      // consume newline
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

// ParseFunctionDecl parses a function declaration: fn <name>(params): returnType { <body> }
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

	// Parse parameters
	parameters := p.parseParameterList()

	// Expect ')'
	if p.CurrentToken().Type != lexer.TokenTypeRParen {
		p.Errors = append(p.Errors, fmt.Errorf("expected ')' after parameters, got %s", p.CurrentToken().Value))
		return nil
	}

	rightParen := p.CurrentToken().Pos
	p.advance() // consume ')'

	// Check for optional return type annotation
	var returnType string
	var returnPos ast.Position
	if p.CurrentToken().Type == lexer.TokenTypeColon {
		p.advance() // consume ':'

		// Expect return type identifier
		if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
			p.Errors = append(p.Errors, fmt.Errorf("expected return type, got %s", p.CurrentToken().Value))
			return nil
		}

		returnType = p.CurrentToken().Value
		returnPos = p.CurrentToken().Pos
		p.advance() // consume return type
	} else {
		// No return type specified - default to void
		returnType = "void"
		// returnPos stays as zero value
	}

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
		Parameters: parameters,
		RightParen: rightParen,
		ReturnType: returnType,
		ReturnPos:  returnPos,
		Body:       body,
	}
}

// parseParameterList parses a comma-separated list of parameters: name: type, name: type, ...
func (p *parser) parseParameterList() []ast.Parameter {
	parameters := []ast.Parameter{}

	// Check if there are no parameters
	if p.CurrentToken().Type == lexer.TokenTypeRParen {
		return parameters
	}

	// Parse first parameter
	param := p.parseParameter()
	if param != nil {
		parameters = append(parameters, *param)
	}

	// Parse remaining parameters
	for p.CurrentToken().Type == lexer.TokenTypeComma {
		p.advance() // consume ','
		param := p.parseParameter()
		if param != nil {
			parameters = append(parameters, *param)
		}
	}

	return parameters
}

// parseParameter parses a single parameter: name: type
func (p *parser) parseParameter() *ast.Parameter {
	// Expect identifier (parameter name)
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.Errors = append(p.Errors, fmt.Errorf("expected parameter name, got %s", p.CurrentToken().Value))
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect ':'
	if p.CurrentToken().Type != lexer.TokenTypeColon {
		p.Errors = append(p.Errors, fmt.Errorf("expected ':' after parameter name, got %s", p.CurrentToken().Value))
		return nil
	}

	colonPos := p.CurrentToken().Pos
	p.advance() // consume ':'

	// Expect type identifier
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.Errors = append(p.Errors, fmt.Errorf("expected parameter type, got %s", p.CurrentToken().Value))
		return nil
	}

	typeName := p.CurrentToken().Value
	typePos := p.CurrentToken().Pos
	p.advance() // consume type

	return &ast.Parameter{
		Name:     name,
		NamePos:  namePos,
		Colon:    colonPos,
		TypeName: typeName,
		TypePos:  typePos,
	}
}

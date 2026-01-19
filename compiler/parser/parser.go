package parser

import (
	"fmt"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/lexer"
	"github.com/seanrogers2657/slang/errors"
)

// Precedence levels for operators (higher = tighter binding)
type precedence int

const (
	precedenceLowest     precedence = iota
	precedenceElvis                 // ?:
	precedenceOr                    // ||
	precedenceAnd                   // &&
	precedenceComparison            // ==, !=, <, >, <=, >=
	precedenceSum                   // +, -
	precedenceProduct               // *, /, %
	precedencePrefix                // -x, !x
)

func NewParser(source []lexer.Token) *parser {
	return NewParserWithFilename(source, "")
}

// NewParserWithFilename creates a new parser with a source filename for error reporting
func NewParserWithFilename(source []lexer.Token, filename string) *parser {
	return &parser{
		Source:   source,
		Index:    0,
		Filename: filename,
	}
}

type parser struct {
	Source   []lexer.Token
	Index    int
	Filename string // source filename for error reporting

	Errors []*errors.CompilerError
}

// toErrorPos converts an ast.Position to an errors.Position
func toErrorPos(p ast.Position) errors.Position {
	return errors.Position{Line: p.Line, Column: p.Column, Offset: p.Offset}
}

// addError creates and adds a compiler error at a specific position
func (p *parser) addError(message string, pos ast.Position) *errors.CompilerError {
	err := errors.NewError(message, p.Filename, toErrorPos(pos), "parser")
	err.Tool = errors.ToolSL
	p.Errors = append(p.Errors, err)
	return err
}

// addErrorWithSpan creates and adds a compiler error spanning from start to end position
func (p *parser) addErrorWithSpan(message string, startPos, endPos ast.Position) *errors.CompilerError {
	err := errors.NewErrorWithSpan(message, p.Filename, toErrorPos(startPos), toErrorPos(endPos), "parser")
	err.Tool = errors.ToolSL
	p.Errors = append(p.Errors, err)
	return err
}

// getPrecedence returns the precedence level for the current token
func (p *parser) getPrecedence(tokenType lexer.TokenType) precedence {
	switch tokenType {
	case lexer.TokenTypeElvis:
		return precedenceElvis
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
	if p.Index <= 0 {
		return lexer.Token{}
	}
	return p.Source[p.Index-1]
}

func (p *parser) CurrentToken() lexer.Token {
	if p.isAtEnd() {
		return lexer.Token{}
	}
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

// peekPastNewlinesIs looks ahead past any newlines and returns true if the next non-newline token matches the given type.
// This is used to support multi-line expressions like chained field access.
func (p *parser) peekPastNewlinesIs(tokenType lexer.TokenType) bool {
	i := p.Index
	for i < len(p.Source) && p.Source[i].Type == lexer.TokenTypeNewline {
		i++
	}
	if i >= len(p.Source) {
		return false
	}
	return p.Source[i].Type == tokenType
}

// peekToken returns the token at the given offset from the current position.
// Returns an invalid token if the offset is out of bounds.
func (p *parser) peekToken(offset int) lexer.Token {
	idx := p.Index + offset
	if idx >= len(p.Source) || idx < 0 {
		return lexer.Token{Type: lexer.TokenTypeInvalid}
	}
	return p.Source[idx]
}

// looksLikeFunctionDecl returns true if the current position looks like a function declaration.
// Pattern: identifier = (
func (p *parser) looksLikeFunctionDecl() bool {
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		return false
	}
	if p.peekToken(1).Type != lexer.TokenTypeAssign {
		return false
	}
	return p.peekToken(2).Type == lexer.TokenTypeLParen
}

// looksLikeStructDecl returns true if the current position looks like a struct declaration.
// Pattern: identifier = struct
func (p *parser) looksLikeStructDecl() bool {
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		return false
	}
	if p.peekToken(1).Type != lexer.TokenTypeAssign {
		return false
	}
	return p.peekToken(2).Type == lexer.TokenTypeStruct
}

// skipUntilDecl skips tokens until we find a function or struct declaration or reach end of input.
// This is used for error recovery to continue parsing after a syntax error.
func (p *parser) skipUntilDecl() {
	for !p.isAtEnd() {
		// If we find a declaration (identifier = ( or identifier = struct), stop
		if p.looksLikeFunctionDecl() || p.looksLikeStructDecl() {
			return
		}
		p.Index++
	}
}

// skipToNextStatement skips tokens until we find something that could start a new statement.
// This is used for error recovery within function bodies.
func (p *parser) skipToNextStatement() {
	for !p.isAtEnd() {
		tok := p.CurrentToken().Type
		// Stop at tokens that could start a new statement
		switch tok {
		case lexer.TokenTypeVal, lexer.TokenTypeVar, lexer.TokenTypeIf, lexer.TokenTypeWhen,
			lexer.TokenTypeFor, lexer.TokenTypeWhile, lexer.TokenTypeBreak, lexer.TokenTypeContinue,
			lexer.TokenTypeReturn, lexer.TokenTypeRBrace:
			return
		case lexer.TokenTypeNewline:
			// Skip the newline and check next token
			p.Index++
			continue
		}
		p.Index++
	}
}

// skipStructDeclaration skips an entire struct declaration: struct Name(fields...)
// Used for error recovery when struct is found inside a function.
func (p *parser) skipStructDeclaration() {
	// Skip 'struct' keyword
	if p.CurrentToken().Type == lexer.TokenTypeStruct {
		p.Index++
	}
	// Skip struct name
	if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeIdentifier {
		p.Index++
	}
	// Skip '(' and everything until matching ')'
	if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeLParen {
		p.Index++
		parenDepth := 1
		for !p.isAtEnd() && parenDepth > 0 {
			switch p.CurrentToken().Type {
			case lexer.TokenTypeLParen:
				parenDepth++
			case lexer.TokenTypeRParen:
				parenDepth--
			}
			p.Index++
		}
	}
	// Skip trailing newline if present
	if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeNewline {
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
	case lexer.TokenTypeElvis:
		return "?:"
	default:
		return ""
	}
}

// Top level parsing
func (p *parser) Parse() *ast.Program {
	// Skip leading newlines first to get meaningful start position
	p.skipNewlines()

	startPos := ast.Position{Line: 1, Column: 1, Offset: 0}
	if !p.isAtEnd() {
		startPos = p.CurrentToken().Pos
	}

	program := &ast.Program{
		Declarations: []ast.Declaration{},
		Statements:   []ast.Statement{},
		StartPos:     startPos,
	}

	// Check if this is a declaration-based program or legacy statement-based program
	if !p.isAtEnd() && (p.looksLikeFunctionDecl() || p.looksLikeStructDecl()) {
		// New style: parse declarations (functions and structs)
		for !p.isAtEnd() {
			p.skipNewlines()
			if p.isAtEnd() {
				break
			}

			// Check if current position looks like a function declaration or struct
			if p.looksLikeFunctionDecl() {
				fnDecl := p.ParseFunctionDecl()
				if fnDecl != nil {
					program.Declarations = append(program.Declarations, fnDecl)
				} else {
					// If parsing failed, try to recover by skipping to next declaration
					p.skipUntilDecl()
				}
			} else if p.looksLikeStructDecl() {
				structDecl := p.ParseStructDecl()
				if structDecl != nil {
					program.Declarations = append(program.Declarations, structDecl)
				} else {
					// If parsing failed, try to recover by skipping to next declaration
					p.skipUntilDecl()
				}
			} else {
				// Report error for unexpected token and try to recover
				p.addError(fmt.Sprintf("expected declaration (Name = (...) or Name = struct {...}), got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
				// Skip tokens until we find a declaration or reach end
				p.skipUntilDecl()
				continue
			}

			// Skip newlines after declaration
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
					p.addError(fmt.Sprintf("expected newline after statement, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
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
	// Check for struct declaration inside function (not allowed)
	if p.CurrentToken().Type == lexer.TokenTypeStruct {
		pos := p.CurrentToken().Pos
		p.addError("struct declarations are only allowed at the top level", pos).
			WithHint("move the struct declaration outside of the function")
		// Skip the entire struct declaration for error recovery
		p.skipStructDeclaration()
		return nil
	}

	// Check if it's a return statement
	if p.CurrentToken().Type == lexer.TokenTypeReturn {
		return p.ParseReturnStatement()
	}

	// Check if it's an if statement
	if p.CurrentToken().Type == lexer.TokenTypeIf {
		return p.ParseIfStatement()
	}

	// Check if it's a for statement
	if p.CurrentToken().Type == lexer.TokenTypeFor {
		return p.ParseForStatement()
	}

	// Check if it's a while statement
	if p.CurrentToken().Type == lexer.TokenTypeWhile {
		return p.ParseWhileStatement()
	}

	// Check if it's a break statement
	if p.CurrentToken().Type == lexer.TokenTypeBreak {
		return p.ParseBreakStatement()
	}

	// Check if it's a continue statement
	if p.CurrentToken().Type == lexer.TokenTypeContinue {
		return p.ParseContinueStatement()
	}

	// Check if it's a when statement
	if p.CurrentToken().Type == lexer.TokenTypeWhen {
		return p.ParseWhenStatement()
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

	// Otherwise, parse as expression and check if it's a field assignment
	expr := p.parseExpression(precedenceLowest)
	if expr == nil {
		return nil
	}

	// Check if this is a field assignment (field access followed by '=')
	if fieldAccess, ok := expr.(*ast.FieldAccessExpr); ok {
		if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeAssign {
			return p.parseFieldAssignment(fieldAccess)
		}
	}

	// Check if this is an index assignment (index expression followed by '=')
	if indexExpr, ok := expr.(*ast.IndexExpr); ok {
		if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeAssign {
			return p.parseIndexAssignment(indexExpr)
		}
	}

	// Otherwise it's an expression statement
	return &ast.ExprStmt{Expr: expr}
}

// ParseVarDecl parses a variable declaration: val <name> = <expr> or val <name>: <type> = <expr>
func (p *parser) ParseVarDecl(mutable bool) ast.Statement {
	// Get position of 'val' or 'var' keyword
	keyword := p.CurrentToken().Pos
	keywordName := p.CurrentToken().Value
	p.advance() // consume 'val' or 'var'

	// Expect identifier
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected identifier after '%s', got '%s'", keywordName, p.CurrentToken().Value), p.CurrentToken().Pos)
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

		// Parse type name (may include generic like Array<i64>)
		typeName, typePos = p.parseTypeName()
		if typeName == "" {
			return nil
		}
	}

	// Expect '='
	if p.CurrentToken().Type != lexer.TokenTypeAssign {
		p.addError(fmt.Sprintf("expected '=' after variable name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	// Parse the initializer expression
	initializer := p.parseExpression(precedenceLowest)
	if initializer == nil {
		p.addError("expected expression after '='", equalsPos)
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
		p.addError("expected expression after '='", equalsPos)
		return nil
	}

	return &ast.AssignStmt{
		Name:    name,
		NamePos: namePos,
		Equals:  equalsPos,
		Value:   value,
	}
}

// parseFieldAssignment parses a field assignment: <expr>.<field> = <expr>
func (p *parser) parseFieldAssignment(fieldAccess *ast.FieldAccessExpr) ast.Statement {
	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	value := p.parseExpression(precedenceLowest)
	if value == nil {
		p.addError("expected expression after '='", equalsPos)
		return nil
	}

	return &ast.FieldAssignStmt{
		Object:   fieldAccess.Object,
		Dot:      fieldAccess.Dot,
		Field:    fieldAccess.Field,
		FieldPos: fieldAccess.FieldPos,
		Equals:   equalsPos,
		Value:    value,
	}
}

// parseIndexAssignment parses an index assignment: arr[idx] = value
func (p *parser) parseIndexAssignment(indexExpr *ast.IndexExpr) ast.Statement {
	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	value := p.parseExpression(precedenceLowest)
	if value == nil {
		p.addError("expected expression after '='", equalsPos)
		return nil
	}

	return &ast.IndexAssignStmt{
		Array:        indexExpr.Array,
		LeftBracket:  indexExpr.LeftBracket,
		Index:        indexExpr.Index,
		RightBracket: indexExpr.RightBracket,
		Equals:       equalsPos,
		Value:        value,
	}
}

// parseArrayLiteral parses an array literal: [elem, elem, ...]
func (p *parser) parseArrayLiteral() ast.Expression {
	leftBracket := p.CurrentToken().Pos
	p.advance() // consume '['

	// Skip newlines after '['
	p.skipNewlines()

	elements := []ast.Expression{}

	// Parse elements
	if p.CurrentToken().Type != lexer.TokenTypeRBracket {
		elem := p.parseExpression(precedenceLowest)
		if elem != nil {
			elements = append(elements, elem)
		}

		// Parse remaining elements
		for p.CurrentToken().Type == lexer.TokenTypeComma {
			p.advance() // consume ','
			p.skipNewlines()

			// Handle trailing comma
			if p.CurrentToken().Type == lexer.TokenTypeRBracket {
				break
			}

			elem := p.parseExpression(precedenceLowest)
			if elem != nil {
				elements = append(elements, elem)
			}
		}
	}

	p.skipNewlines()

	// Expect ']'
	if p.CurrentToken().Type != lexer.TokenTypeRBracket {
		p.addError(fmt.Sprintf("expected ']' to close array literal, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	rightBracket := p.CurrentToken().Pos
	p.advance() // consume ']'

	return &ast.ArrayLiteralExpr{
		LeftBracket:  leftBracket,
		Elements:     elements,
		RightBracket: rightBracket,
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

// ParseIfStatement parses an if statement: if <condition> { <body> } [else { <body> } | else if ...]
func (p *parser) ParseIfStatement() ast.Statement {
	// Get position of 'if' keyword
	ifKeyword := p.CurrentToken().Pos
	p.advance() // consume 'if'

	// Parse the condition expression
	condition := p.parseExpression(precedenceLowest)
	if condition == nil {
		p.addError("expected condition after 'if'", ifKeyword)
		return nil
	}

	// Skip newlines before the block
	p.skipNewlines()

	// Parse the then branch (required block)
	thenBranch := p.ParseBlockStmt()
	if thenBranch == nil {
		return nil
	}

	// Check for optional else clause
	var elseKeyword ast.Position
	var elseBranch ast.Statement

	// Skip newlines to check for else
	// But first save position in case there's no else
	savedIndex := p.Index
	p.skipNewlines()

	if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeElse {
		elseKeyword = p.CurrentToken().Pos
		p.advance() // consume 'else'

		// Skip newlines after else
		p.skipNewlines()

		// Check if it's 'else if' or just 'else'
		if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeIf {
			// else if: recursively parse if statement
			elseBranch = p.ParseIfStatement()
		} else {
			// else: parse block
			elseBranch = p.ParseBlockStmt()
		}
	} else {
		// No else clause, restore position
		p.Index = savedIndex
	}

	return &ast.IfStmt{
		IfKeyword:   ifKeyword,
		Condition:   condition,
		ThenBranch:  thenBranch,
		ElseKeyword: elseKeyword,
		ElseBranch:  elseBranch,
	}
}

// ParseForStatement parses a for loop: for [( ]init; cond; update[)] { body }
// Supports both C-style (with parens) and Go-style (without parens)
func (p *parser) ParseForStatement() ast.Statement {
	forKeyword := p.CurrentToken().Pos
	p.advance() // consume 'for'

	// Check for optional opening parenthesis
	hasParens := p.CurrentToken().Type == lexer.TokenTypeLParen
	var leftParen ast.Position
	if hasParens {
		leftParen = p.CurrentToken().Pos
		p.advance() // consume '('
	}

	// Parse initialization (optional)
	var init ast.Statement
	if p.CurrentToken().Type != lexer.TokenTypeSemicolon {
		if p.CurrentToken().Type == lexer.TokenTypeVal {
			init = p.ParseVarDecl(false)
		} else if p.CurrentToken().Type == lexer.TokenTypeVar {
			init = p.ParseVarDecl(true)
		} else if p.CurrentToken().Type == lexer.TokenTypeIdentifier && p.peek().Type == lexer.TokenTypeAssign {
			init = p.ParseAssignment()
		}
	}

	// Expect semicolon after init
	if p.CurrentToken().Type != lexer.TokenTypeSemicolon {
		p.addError(fmt.Sprintf("expected ';' after for-loop initializer, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}
	p.advance() // consume ';'

	// Parse condition (optional - if missing, it's an infinite loop)
	var condition ast.Expression
	if p.CurrentToken().Type != lexer.TokenTypeSemicolon {
		condition = p.parseExpression(precedenceLowest)
	}

	// Expect semicolon after condition
	if p.CurrentToken().Type != lexer.TokenTypeSemicolon {
		p.addError(fmt.Sprintf("expected ';' after for-loop condition, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}
	p.advance() // consume ';'

	// Parse update (optional)
	var update ast.Statement
	// Check what comes next - if it's ) or {, there's no update
	nextIsEnd := (hasParens && p.CurrentToken().Type == lexer.TokenTypeRParen) ||
		(!hasParens && p.CurrentToken().Type == lexer.TokenTypeLBrace)
	if !nextIsEnd && p.CurrentToken().Type == lexer.TokenTypeIdentifier && p.peek().Type == lexer.TokenTypeAssign {
		update = p.ParseAssignment()
	}

	// If has parens, expect closing paren
	var rightParen ast.Position
	if hasParens {
		if p.CurrentToken().Type != lexer.TokenTypeRParen {
			p.addError(fmt.Sprintf("expected ')' after for-loop update, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
			return nil
		}
		rightParen = p.CurrentToken().Pos
		p.advance() // consume ')'
	}

	// Skip newlines before body
	p.skipNewlines()

	// Parse body
	body := p.ParseBlockStmt()
	if body == nil {
		return nil
	}

	return &ast.ForStmt{
		ForKeyword: forKeyword,
		HasParens:  hasParens,
		LeftParen:  leftParen,
		Init:       init,
		Condition:  condition,
		Update:     update,
		RightParen: rightParen,
		Body:       body,
	}
}

// ParseWhileStatement parses a while statement
// Syntax: while condition { ... } or while (condition) { ... }
func (p *parser) ParseWhileStatement() ast.Statement {
	whileKeyword := p.CurrentToken().Pos
	p.advance() // consume 'while'

	// Check for optional opening parenthesis
	hasParens := p.CurrentToken().Type == lexer.TokenTypeLParen
	var leftParen ast.Position
	if hasParens {
		leftParen = p.CurrentToken().Pos
		p.advance() // consume '('
	}

	// Parse condition (required)
	condition := p.parseExpression(precedenceLowest)
	if condition == nil {
		p.addError("expected condition in while statement", p.CurrentToken().Pos)
		return nil
	}

	// If has parens, expect closing paren
	var rightParen ast.Position
	if hasParens {
		if p.CurrentToken().Type != lexer.TokenTypeRParen {
			p.addError(fmt.Sprintf("expected ')' after while condition, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
			return nil
		}
		rightParen = p.CurrentToken().Pos
		p.advance() // consume ')'
	}

	// Skip newlines before body
	p.skipNewlines()

	// Parse body
	body := p.ParseBlockStmt()
	if body == nil {
		return nil
	}

	return &ast.WhileStmt{
		WhileKeyword: whileKeyword,
		HasParens:    hasParens,
		LeftParen:    leftParen,
		Condition:    condition,
		RightParen:   rightParen,
		Body:         body,
	}
}

// ParseBreakStatement parses a break statement
func (p *parser) ParseBreakStatement() ast.Statement {
	keyword := p.CurrentToken().Pos
	p.advance() // consume 'break'
	return &ast.BreakStmt{Keyword: keyword}
}

// ParseContinueStatement parses a continue statement
func (p *parser) ParseContinueStatement() ast.Statement {
	keyword := p.CurrentToken().Pos
	p.advance() // consume 'continue'
	return &ast.ContinueStmt{Keyword: keyword}
}

// parseIfExpression parses an if expression (same as if statement, but in expression context)
// Returns *ast.IfStmt which implements both Statement and Expression interfaces
func (p *parser) parseIfExpression() ast.Expression {
	// Reuse ParseIfStatement since the syntax is identical
	// The *ast.IfStmt type implements both Statement and Expression
	stmt := p.ParseIfStatement()
	if stmt == nil {
		return nil
	}
	// Use comma-ok assertion for safety against future changes
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok {
		return nil
	}
	return ifStmt
}

func (p *parser) ParseLiteral() ast.Expression {
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

	if p.CurrentToken().Type == lexer.TokenTypeNull {
		token := p.CurrentToken()
		literal := &ast.LiteralExpr{
			Kind:     ast.LiteralTypeNull,
			Value:    "null",
			StartPos: token.Pos,
			EndPos:   ast.Position{Line: token.Pos.Line, Column: token.Pos.Column + 4, Offset: token.Pos.Offset + 4},
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

	// Parse postfix and infix operators
	for !p.isAtEnd() {
		// Handle newlines - but check if next non-newline token is '.' or '?.' for chained access
		if p.CurrentToken().Type == lexer.TokenTypeNewline {
			// Look ahead past newlines to see if there's a '.' or '?.' (chained field access)
			if !p.peekPastNewlinesIs(lexer.TokenTypeDot) && !p.peekPastNewlinesIs(lexer.TokenTypeSafeCall) {
				break
			}
			// Skip the newlines and continue to parse the '.' or '?.'
			p.skipNewlines()
		}

		// Handle field access (dot operator) - highest precedence, left-associative
		// This also handles method calls: expr.method(args)
		if p.CurrentToken().Type == lexer.TokenTypeDot {
			dotPos := p.CurrentToken().Pos
			p.advance() // consume '.'

			// Expect field/method name
			if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
				p.addError(fmt.Sprintf("expected field name after '.', got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
				return nil
			}

			memberName := p.CurrentToken().Value
			memberPos := p.CurrentToken().Pos
			p.advance() // consume field/method name

			// Check if this is a method call (identifier followed by '(')
			if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeLParen {
				methodCall := p.parseMethodCall(left, dotPos, memberName, memberPos)
				if methodCall == nil {
					return nil
				}
				left = methodCall
				continue
			}

			// Otherwise it's a field access
			left = &ast.FieldAccessExpr{
				Object:   left,
				Dot:      dotPos,
				Field:    memberName,
				FieldPos: memberPos,
			}
			continue
		}

		// Handle safe call operator (?.) - same precedence as dot, for nullable field access
		if p.CurrentToken().Type == lexer.TokenTypeSafeCall {
			safeCallPos := p.CurrentToken().Pos
			p.advance() // consume '?.'

			// Expect field name
			if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
				p.addError(fmt.Sprintf("expected field name after '?.', got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
				return nil
			}

			fieldName := p.CurrentToken().Value
			fieldPos := p.CurrentToken().Pos
			p.advance() // consume field name

			left = &ast.SafeCallExpr{
				Object:      left,
				SafeCallPos: safeCallPos,
				Field:       fieldName,
				FieldPos:    fieldPos,
			}
			continue
		}

		// Handle index access (arr[idx]) - postfix operator with high precedence
		if p.CurrentToken().Type == lexer.TokenTypeLBracket {
			leftBracket := p.CurrentToken().Pos
			p.advance() // consume '['

			index := p.parseExpression(precedenceLowest)
			if index == nil {
				p.addError("expected index expression", leftBracket)
				return nil
			}

			if p.CurrentToken().Type != lexer.TokenTypeRBracket {
				p.addError(fmt.Sprintf("expected ']' after index, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
				return nil
			}

			rightBracket := p.CurrentToken().Pos
			p.advance() // consume ']'

			left = &ast.IndexExpr{
				Array:        left,
				LeftBracket:  leftBracket,
				Index:        index,
				RightBracket: rightBracket,
			}
			continue
		}

		// Handle binary operators
		if p.currentPrecedence() <= minPrec {
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
			p.addError(fmt.Sprintf("expected expression after operator '%s'", op), opPos)
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
	// Check for array literal
	if p.CurrentToken().Type == lexer.TokenTypeLBracket {
		return p.parseArrayLiteral()
	}

	// Check for anonymous struct literal (e.g., { x: 0, y: 0 })
	// Used when the type is inferred from context
	if p.CurrentToken().Type == lexer.TokenTypeLBrace {
		return p.parseAnonStructLiteral()
	}

	// Check for grouping expression (parenthesized expression)
	if p.CurrentToken().Type == lexer.TokenTypeLParen {
		leftParen := p.CurrentToken().Pos
		p.advance() // consume '('

		// Parse the inner expression with lowest precedence
		expr := p.parseExpression(precedenceLowest)
		if expr == nil {
			p.addError("expected expression after '('", leftParen)
			return nil
		}

		// Expect ')'
		if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRParen {
			p.addError("expected ')' to close grouping expression", p.PreviousToken().Pos)
			return nil
		}

		rightParen := p.CurrentToken().Pos
		p.advance() // consume ')'

		return &ast.GroupingExpr{
			Expr:       expr,
			LeftParen:  leftParen,
			RightParen: rightParen,
		}
	}

	// Check for if expression
	if p.CurrentToken().Type == lexer.TokenTypeIf {
		return p.parseIfExpression()
	}

	// Check for when expression
	if p.CurrentToken().Type == lexer.TokenTypeWhen {
		return p.parseWhenExpression()
	}

	// Check for unary NOT operator
	if p.CurrentToken().Type == lexer.TokenTypeNot {
		opPos := p.CurrentToken().Pos
		p.advance() // consume '!'

		// Parse the operand (recursively call parsePrimary for highest precedence)
		operand := p.parsePrimary()
		if operand == nil {
			p.addError("expected expression after '!'", opPos)
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

		// Check if this is a struct literal (Name { ... })
		// The brace must be on the same line and immediately after the identifier (no space)
		// This prevents "if a == b {" from being parsed as "b { ..."
		// For struct literals with space like "Point { 1, 2 }", the user should use "Point{ 1, 2 }"
		if !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeLBrace {
			bracePos := p.CurrentToken().Pos
			identEnd := token.Pos.Column + len(token.Value)
			// Only parse as struct literal if brace is DIRECTLY after identifier (no space)
			if bracePos.Line == token.Pos.Line && bracePos.Column == identEnd {
				return p.parseStructLiteral(token.Value, token.Pos)
			}
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

	// Skip newlines after '(' for multi-line argument lists
	p.skipNewlines()

	arguments := []ast.Expression{}
	namedArguments := []ast.NamedArgument{}
	hasNamedArgs := false
	hasPositionalArgs := false

	// Parse arguments
	if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRParen {
		// Check if first argument is named (identifier followed by ':')
		if p.isNamedArgument() {
			hasNamedArgs = true
			namedArg := p.parseNamedArgument()
			if namedArg != nil {
				namedArguments = append(namedArguments, *namedArg)
			}
		} else {
			hasPositionalArgs = true
			arg := p.parseExpression(precedenceLowest)
			if arg != nil {
				arguments = append(arguments, arg)
			}
		}

		// Parse remaining arguments
		for !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeComma {
			p.advance() // consume ','

			// Skip newlines after ',' for multi-line argument lists
			p.skipNewlines()

			// Skip trailing comma
			if p.CurrentToken().Type == lexer.TokenTypeRParen {
				break
			}

			if p.isNamedArgument() {
				if hasPositionalArgs {
					p.addError("cannot mix positional and named arguments", p.CurrentToken().Pos)
					return nil
				}
				hasNamedArgs = true
				namedArg := p.parseNamedArgument()
				if namedArg != nil {
					namedArguments = append(namedArguments, *namedArg)
				}
			} else {
				if hasNamedArgs {
					p.addError("cannot mix positional and named arguments", p.CurrentToken().Pos)
					return nil
				}
				hasPositionalArgs = true
				arg := p.parseExpression(precedenceLowest)
				if arg != nil {
					arguments = append(arguments, arg)
				}
			}
		}
	}

	// Skip newlines before ')' for multi-line argument lists
	p.skipNewlines()

	// Expect ')'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRParen {
		p.addError(fmt.Sprintf("expected ')' after function arguments, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	rightParen := p.CurrentToken().Pos
	p.advance() // consume ')'

	return &ast.CallExpr{
		Name:           name,
		NamePos:        namePos,
		LeftParen:      leftParen,
		Arguments:      arguments,
		NamedArguments: namedArguments,
		RightParen:     rightParen,
	}
}

// parseMethodCall parses a method call after the object, dot, and method name have been consumed
// Syntax: object.method(args...)
// Used for patterns like Heap.new(expr), p.copy(), etc.
func (p *parser) parseMethodCall(object ast.Expression, dotPos ast.Position, methodName string, methodPos ast.Position) ast.Expression {
	leftParen := p.CurrentToken().Pos
	p.advance() // consume '('

	// Skip newlines after '(' for multi-line argument lists
	p.skipNewlines()

	arguments := []ast.Expression{}

	// Parse arguments (similar to parseCallExpr but without named arguments for now)
	for !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRParen {
		arg := p.parseExpression(precedenceLowest)
		if arg == nil {
			return nil
		}
		arguments = append(arguments, arg)

		// Skip newlines after argument
		p.skipNewlines()

		// Check for comma (more arguments) or closing paren
		if p.CurrentToken().Type == lexer.TokenTypeComma {
			p.advance() // consume ','
			p.skipNewlines()
		} else if p.CurrentToken().Type != lexer.TokenTypeRParen {
			p.addError(fmt.Sprintf("expected ',' or ')' in method call, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
			return nil
		}
	}

	// Expect ')'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRParen {
		p.addError("expected ')' to close method call", p.PreviousToken().Pos)
		return nil
	}

	rightParen := p.CurrentToken().Pos
	p.advance() // consume ')'

	return &ast.MethodCallExpr{
		Object:     object,
		Dot:        dotPos,
		Method:     methodName,
		MethodPos:  methodPos,
		LeftParen:  leftParen,
		Arguments:  arguments,
		RightParen: rightParen,
	}
}

// parseStructLiteral parses a struct literal after the identifier has been consumed
// Syntax: Name { field1, field2 } or Name { x: field1, y: field2 }
func (p *parser) parseStructLiteral(name string, namePos ast.Position) ast.Expression {
	leftBrace := p.CurrentToken().Pos
	p.advance() // consume '{'

	// Skip newlines after '{' for multi-line field lists
	p.skipNewlines()

	arguments := []ast.Expression{}
	namedArguments := []ast.NamedArgument{}
	hasNamedArgs := false
	hasPositionalArgs := false

	// Parse fields
	if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRBrace {
		// Check if first field is named (identifier followed by ':')
		if p.isNamedArgument() {
			hasNamedArgs = true
			namedArg := p.parseNamedArgument()
			if namedArg != nil {
				namedArguments = append(namedArguments, *namedArg)
			}
		} else {
			hasPositionalArgs = true
			arg := p.parseExpression(precedenceLowest)
			if arg != nil {
				arguments = append(arguments, arg)
			}
		}

		// Parse remaining fields
		for !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeComma {
			p.advance() // consume ','

			// Skip newlines after ',' for multi-line field lists
			p.skipNewlines()

			// Skip trailing comma
			if p.CurrentToken().Type == lexer.TokenTypeRBrace {
				break
			}

			if p.isNamedArgument() {
				if hasPositionalArgs {
					p.addError("cannot mix positional and named arguments", p.CurrentToken().Pos)
					return nil
				}
				hasNamedArgs = true
				namedArg := p.parseNamedArgument()
				if namedArg != nil {
					namedArguments = append(namedArguments, *namedArg)
				}
			} else {
				if hasNamedArgs {
					p.addError("cannot mix positional and named arguments", p.CurrentToken().Pos)
					return nil
				}
				hasPositionalArgs = true
				arg := p.parseExpression(precedenceLowest)
				if arg != nil {
					arguments = append(arguments, arg)
				}
			}
		}
	}

	// Skip newlines before '}' for multi-line field lists
	p.skipNewlines()

	// Expect '}'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRBrace {
		p.addError(fmt.Sprintf("expected '}' after struct fields, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	rightBrace := p.CurrentToken().Pos
	p.advance() // consume '}'

	return &ast.StructLiteral{
		Name:           name,
		NamePos:        namePos,
		LeftBrace:      leftBrace,
		Arguments:      arguments,
		NamedArguments: namedArguments,
		RightBrace:     rightBrace,
	}
}

// parseAnonStructLiteral parses an anonymous struct literal: { field1, field2 } or { x: field1, y: field2 }
// The type is inferred from context (e.g., val p: Point = { x: 0, y: 0 })
func (p *parser) parseAnonStructLiteral() ast.Expression {
	leftBrace := p.CurrentToken().Pos
	p.advance() // consume '{'

	// Skip newlines after '{' for multi-line field lists
	p.skipNewlines()

	arguments := []ast.Expression{}
	namedArguments := []ast.NamedArgument{}
	hasNamedArgs := false
	hasPositionalArgs := false

	// Parse fields
	if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRBrace {
		// Check if first field is named (identifier followed by ':')
		if p.isNamedArgument() {
			hasNamedArgs = true
			namedArg := p.parseNamedArgument()
			if namedArg != nil {
				namedArguments = append(namedArguments, *namedArg)
			}
		} else {
			hasPositionalArgs = true
			arg := p.parseExpression(precedenceLowest)
			if arg != nil {
				arguments = append(arguments, arg)
			}
		}

		// Parse remaining fields
		for !p.isAtEnd() && p.CurrentToken().Type == lexer.TokenTypeComma {
			p.advance() // consume ','

			// Skip newlines after ',' for multi-line field lists
			p.skipNewlines()

			// Skip trailing comma
			if p.CurrentToken().Type == lexer.TokenTypeRBrace {
				break
			}

			if p.isNamedArgument() {
				if hasPositionalArgs {
					p.addError("cannot mix positional and named arguments", p.CurrentToken().Pos)
					return nil
				}
				hasNamedArgs = true
				namedArg := p.parseNamedArgument()
				if namedArg != nil {
					namedArguments = append(namedArguments, *namedArg)
				}
			} else {
				if hasNamedArgs {
					p.addError("cannot mix positional and named arguments", p.CurrentToken().Pos)
					return nil
				}
				hasPositionalArgs = true
				arg := p.parseExpression(precedenceLowest)
				if arg != nil {
					arguments = append(arguments, arg)
				}
			}
		}
	}

	// Skip newlines before '}' for multi-line field lists
	p.skipNewlines()

	// Expect '}'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRBrace {
		p.addError(fmt.Sprintf("expected '}' after struct fields, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	rightBrace := p.CurrentToken().Pos
	p.advance() // consume '}'

	return &ast.AnonStructLiteral{
		LeftBrace:      leftBrace,
		Arguments:      arguments,
		NamedArguments: namedArguments,
		RightBrace:     rightBrace,
	}
}

// isNamedArgument checks if the current position has a named argument (identifier followed by ':')
func (p *parser) isNamedArgument() bool {
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		return false
	}
	// Look ahead to see if next token is ':'
	next := p.peek()
	return next.Type == lexer.TokenTypeColon
}

// parseNamedArgument parses a named argument: name: expr
func (p *parser) parseNamedArgument() *ast.NamedArgument {
	// Current token should be identifier
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected argument name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect ':'
	if p.CurrentToken().Type != lexer.TokenTypeColon {
		p.addError(fmt.Sprintf("expected ':' after argument name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	colonPos := p.CurrentToken().Pos
	p.advance() // consume ':'

	// Parse the value expression
	value := p.parseExpression(precedenceLowest)
	if value == nil {
		p.addError("expected expression after ':'", colonPos)
		return nil
	}

	return &ast.NamedArgument{
		Name:    name,
		NamePos: namePos,
		Colon:   colonPos,
		Value:   value,
	}
}

// ParseBinaryExpression is kept for backward compatibility during transition
// It now delegates to the new Pratt parser
func (p *parser) ParseBinaryExpression() ast.Expression {
	expr := p.parseExpression(precedenceLowest)
	if expr == nil && len(p.Errors) == 0 {
		// Add error if no expression was parsed and no error was set
		p.addError(fmt.Sprintf("unsupported operation: '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
	}
	return expr
}

// ParseBlockStmt parses a block statement: { <statements> }
func (p *parser) ParseBlockStmt() *ast.BlockStmt {
	// Expect '{'
	if p.CurrentToken().Type != lexer.TokenTypeLBrace {
		p.addError(fmt.Sprintf("expected '{', got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
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

			// After each statement, expect newline or '}'
			if !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRBrace {
				if p.CurrentToken().Type == lexer.TokenTypeNewline {
					p.advance()      // consume newline
					p.skipNewlines() // skip any additional newlines
				} else {
					p.addError(fmt.Sprintf("expected newline or '}' after statement, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
					break
				}
			}
		} else {
			// Error recovery: skip to the next statement or end of block
			// This prevents infinite loops when parsing fails
			p.skipToNextStatement()
		}
	}

	// Expect '}'
	if p.isAtEnd() || p.CurrentToken().Type != lexer.TokenTypeRBrace {
		p.addError("expected '}' to close block", leftBrace)
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

// ParseFunctionDecl parses a function declaration: name = (params) -> returnType { body }
// The return type is optional: name = (params) { body } defaults to void
func (p *parser) ParseFunctionDecl() *ast.FunctionDecl {
	// Expect identifier (function name)
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected function name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect '='
	if p.CurrentToken().Type != lexer.TokenTypeAssign {
		p.addError(fmt.Sprintf("expected '=' after function name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	// Expect '('
	if p.CurrentToken().Type != lexer.TokenTypeLParen {
		p.addError(fmt.Sprintf("expected '(' after '=', got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	leftParen := p.CurrentToken().Pos
	p.advance() // consume '('

	// Parse parameters
	parameters := p.parseParameterList()

	// Expect ')'
	if p.CurrentToken().Type != lexer.TokenTypeRParen {
		p.addError(fmt.Sprintf("expected ')' after parameters, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	rightParen := p.CurrentToken().Pos
	p.advance() // consume ')'

	// Check for optional return type: -> returnType
	var arrowPos ast.Position
	var returnType string
	var returnPos ast.Position

	if p.CurrentToken().Type == lexer.TokenTypeArrow {
		arrowPos = p.CurrentToken().Pos
		p.advance() // consume '->'

		// Parse return type (may include generics like Own<Point>, Array<i64>)
		returnType, returnPos = p.parseTypeName()
		if returnType == "" {
			return nil
		}
	} else {
		// No return type specified - default to void
		returnType = "void"
		// arrowPos and returnPos stay as zero values
	}

	// Skip newlines before body
	p.skipNewlines()

	// Parse function body (block statement)
	body := p.ParseBlockStmt()
	if body == nil {
		return nil
	}

	return &ast.FunctionDecl{
		Name:       name,
		NamePos:    namePos,
		EqualsPos:  equalsPos,
		LeftParen:  leftParen,
		Parameters: parameters,
		RightParen: rightParen,
		ArrowPos:   arrowPos,
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

// parseParameter parses a single parameter: [var] name: type
// The 'var' prefix indicates a mutable reference parameter.
func (p *parser) parseParameter() *ast.Parameter {
	// Check for optional 'var' prefix (for mutable reference parameters)
	var mutable bool
	var varPos ast.Position
	if p.CurrentToken().Type == lexer.TokenTypeVar {
		mutable = true
		varPos = p.CurrentToken().Pos
		p.advance() // consume 'var'
	}

	// Expect identifier (parameter name)
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected parameter name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect ':'
	if p.CurrentToken().Type != lexer.TokenTypeColon {
		p.addError(fmt.Sprintf("expected ':' after parameter name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	colonPos := p.CurrentToken().Pos
	p.advance() // consume ':'

	// Parse type name (may include generic like Array<i64>, Own<T>, Ref<T>)
	typeName, typePos := p.parseTypeName()
	if typeName == "" {
		return nil
	}

	return &ast.Parameter{
		Mutable:  mutable,
		VarPos:   varPos,
		Name:     name,
		NamePos:  namePos,
		Colon:    colonPos,
		TypeName: typeName,
		TypePos:  typePos,
	}
}

// parseTypeName parses a type name, including:
// - Symbol pointer types: *T, &T, &&T
// - Generic types: Array<i64>, Own<Point>, Ref<Point>
// - Nullable types: i64?, *Point?
func (p *parser) parseTypeName() (string, ast.Position) {
	typePos := p.CurrentToken().Pos

	// Check for pointer type prefixes: *T, &T, &&T
	switch p.CurrentToken().Type {
	case lexer.TokenTypeMultiply: // *T - owned pointer
		p.advance() // consume '*'
		innerType, _ := p.parseTypeName()
		if innerType == "" {
			return "", typePos
		}
		return "*" + innerType, typePos

	case lexer.TokenTypeAnd: // &&T - mutable borrow
		p.advance() // consume '&&'
		innerType, _ := p.parseTypeName()
		if innerType == "" {
			return "", typePos
		}
		return "&&" + innerType, typePos

	case lexer.TokenTypeAmpersand: // &T - immutable borrow
		p.advance() // consume '&'
		innerType, _ := p.parseTypeName()
		if innerType == "" {
			return "", typePos
		}
		return "&" + innerType, typePos
	}

	// Standard identifier-based type
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected type name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return "", ast.Position{}
	}

	typeName := p.CurrentToken().Value
	p.advance() // consume type identifier

	// Check for generic type: Name<T> (e.g., Array<i64>, Own<Point>, Ref<Point>)
	if p.CurrentToken().Type == lexer.TokenTypeLessThan {
		p.advance() // consume '<'

		// Parse type argument (which may itself be nullable or generic, e.g., Own<Array<i64>>)
		typeArg, _ := p.parseTypeName()
		if typeArg == "" {
			return "", typePos
		}

		if p.CurrentToken().Type != lexer.TokenTypeGreaterThan {
			p.addError(fmt.Sprintf("expected '>' after type argument, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
			return "", typePos
		}
		p.advance() // consume '>'

		// Encode as "Name<typeArg>" for the semantic analyzer
		typeName = fmt.Sprintf("%s<%s>", typeName, typeArg)
	}

	// Check for nullable type: T?
	if p.CurrentToken().Type == lexer.TokenTypeQuestion {
		p.advance() // consume '?'
		typeName = typeName + "?"

		// Check for nested nullable type: T?? (error)
		if p.CurrentToken().Type == lexer.TokenTypeQuestion {
			p.addError("nested nullable types are not allowed", p.CurrentToken().Pos)
			p.advance() // consume the extra '?' to recover
		}
	}

	return typeName, typePos
}

// ParseStructDecl parses a struct declaration: Name = struct { fields }
func (p *parser) ParseStructDecl() *ast.StructDecl {
	// Expect identifier (struct name)
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected struct name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect '='
	if p.CurrentToken().Type != lexer.TokenTypeAssign {
		p.addError(fmt.Sprintf("expected '=' after struct name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	equalsPos := p.CurrentToken().Pos
	p.advance() // consume '='

	// Expect 'struct' keyword
	if p.CurrentToken().Type != lexer.TokenTypeStruct {
		p.addError(fmt.Sprintf("expected 'struct' after '=', got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	structKeyword := p.CurrentToken().Pos
	p.advance() // consume 'struct'

	// Expect '{'
	if p.CurrentToken().Type != lexer.TokenTypeLBrace {
		p.addError(fmt.Sprintf("expected '{' after 'struct', got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	leftBrace := p.CurrentToken().Pos
	p.advance() // consume '{'

	// Skip newlines after opening brace
	p.skipNewlines()

	// Parse fields
	fields := p.parseStructFields()

	// Expect '}'
	if p.CurrentToken().Type != lexer.TokenTypeRBrace {
		p.addError(fmt.Sprintf("expected '}' after struct fields, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	rightBrace := p.CurrentToken().Pos
	p.advance() // consume '}'

	return &ast.StructDecl{
		Name:          name,
		NamePos:       namePos,
		EqualsPos:     equalsPos,
		StructKeyword: structKeyword,
		LeftBrace:     leftBrace,
		Fields:        fields,
		RightBrace:    rightBrace,
	}
}

// parseStructFields parses newline-separated struct fields: name: type
func (p *parser) parseStructFields() []ast.StructField {
	fields := []ast.StructField{}

	// Check if there are no fields
	if p.CurrentToken().Type == lexer.TokenTypeRBrace {
		return fields
	}

	// Parse fields (newline-separated)
	for p.CurrentToken().Type != lexer.TokenTypeRBrace && !p.isAtEnd() {
		field := p.parseStructField()
		if field != nil {
			fields = append(fields, *field)
		}

		// Skip newlines between fields
		p.skipNewlines()
	}

	return fields
}

// parseStructField parses a single struct field: val name: type or var name: type
func (p *parser) parseStructField() *ast.StructField {
	// Expect val or var keyword
	if p.CurrentToken().Type != lexer.TokenTypeVal && p.CurrentToken().Type != lexer.TokenTypeVar {
		p.addError(fmt.Sprintf("expected 'val' or 'var' for struct field, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	mutable := p.CurrentToken().Type == lexer.TokenTypeVar
	keywordPos := p.CurrentToken().Pos
	p.advance() // consume val/var

	// Expect field name
	if p.CurrentToken().Type != lexer.TokenTypeIdentifier {
		p.addError(fmt.Sprintf("expected field name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	name := p.CurrentToken().Value
	namePos := p.CurrentToken().Pos
	p.advance() // consume identifier

	// Expect ':'
	if p.CurrentToken().Type != lexer.TokenTypeColon {
		p.addError(fmt.Sprintf("expected ':' after field name, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}

	colonPos := p.CurrentToken().Pos
	p.advance() // consume ':'

	// Parse type name (may include nullable types like i64? or generics like Array<i64>)
	typeName, typePos := p.parseTypeName()
	if typeName == "" {
		return nil
	}

	// Skip newlines after field
	p.skipNewlines()

	return &ast.StructField{
		Mutable:    mutable,
		KeywordPos: keywordPos,
		Name:       name,
		NamePos:    namePos,
		Colon:      colonPos,
		TypeName:   typeName,
		TypePos:    typePos,
	}
}

// ParseWhenStatement parses a when statement (can also be used as expression)
func (p *parser) ParseWhenStatement() ast.Statement {
	expr := p.parseWhenExpression()
	if expr == nil {
		return nil
	}
	return expr.(*ast.WhenExpr)
}

// parseWhenExpression parses a when expression
// Form: when { condition -> body, ... }
func (p *parser) parseWhenExpression() ast.Expression {
	whenKeyword := p.CurrentToken().Pos
	p.advance() // consume 'when'

	// Skip newlines before brace
	p.skipNewlines()

	// Check for unsupported when (subject) { } syntax
	if p.CurrentToken().Type == lexer.TokenTypeLParen {
		p.addError(
			"when (subject) { } syntax is not supported",
			p.CurrentToken().Pos,
		).WithHint("use 'when { condition -> body }' with boolean conditions instead")
		return nil
	}

	// Expect '{'
	if p.CurrentToken().Type != lexer.TokenTypeLBrace {
		p.addError("expected '{' after when", p.CurrentToken().Pos)
		return nil
	}
	leftBrace := p.CurrentToken().Pos
	p.advance() // consume '{'

	// Parse cases
	cases := p.parseWhenCases()

	// Skip newlines before closing brace
	p.skipNewlines()

	// Expect '}'
	if p.CurrentToken().Type != lexer.TokenTypeRBrace {
		p.addError("expected '}' to close when expression", p.CurrentToken().Pos)
		return nil
	}
	rightBrace := p.CurrentToken().Pos
	p.advance() // consume '}'

	return &ast.WhenExpr{
		WhenKeyword: whenKeyword,
		LeftBrace:   leftBrace,
		Cases:       cases,
		RightBrace:  rightBrace,
	}
}

// parseWhenCases parses the list of cases inside a when expression
func (p *parser) parseWhenCases() []ast.WhenCase {
	cases := []ast.WhenCase{}

	for !p.isAtEnd() && p.CurrentToken().Type != lexer.TokenTypeRBrace {
		// Skip newlines between cases
		p.skipNewlines()

		if p.CurrentToken().Type == lexer.TokenTypeRBrace {
			break
		}

		wcase := p.parseWhenCase()
		if wcase != nil {
			cases = append(cases, *wcase)
		}

		// Skip newlines after case
		p.skipNewlines()
	}

	return cases
}

// parseWhenCase parses a single when case: condition -> body
func (p *parser) parseWhenCase() *ast.WhenCase {
	conditionPos := p.CurrentToken().Pos
	var condition ast.Expression
	isElse := false

	// Check for 'else' keyword
	if p.CurrentToken().Type == lexer.TokenTypeElse {
		isElse = true
		p.advance() // consume 'else'
	} else {
		// Parse condition (boolean expression)
		condition = p.parseExpression(precedenceLowest)
		if condition == nil {
			p.addError("expected condition in when case", conditionPos)
			return nil
		}
	}

	// Expect '->'
	if p.CurrentToken().Type != lexer.TokenTypeArrow {
		p.addError(fmt.Sprintf("expected '->' after when case condition, got '%s'", p.CurrentToken().Value), p.CurrentToken().Pos)
		return nil
	}
	arrow := p.CurrentToken().Pos
	p.advance() // consume '->'

	// Skip newlines after arrow
	p.skipNewlines()

	// Parse body (either a block, assignment statement, or single expression)
	var body ast.Statement
	if p.CurrentToken().Type == lexer.TokenTypeLBrace {
		body = p.ParseBlockStmt()
	} else if p.CurrentToken().Type == lexer.TokenTypeIdentifier && p.peek().Type == lexer.TokenTypeAssign {
		// Assignment statement: name = expr
		body = p.ParseAssignment()
	} else {
		// Single expression - parse it as an expression statement
		expr := p.parseExpression(precedenceLowest)
		if expr == nil {
			p.addError("expected expression after '->'", arrow)
			return nil
		}
		body = &ast.ExprStmt{Expr: expr}
	}

	if body == nil {
		return nil
	}

	return &ast.WhenCase{
		Condition:    condition,
		ConditionPos: conditionPos,
		Arrow:        arrow,
		Body:         body,
		IsElse:       isElse,
	}
}

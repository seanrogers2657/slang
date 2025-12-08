package semantic

import "github.com/seanrogers2657/slang/compiler/ast"

// TypedNode represents a node in the typed AST
type TypedNode interface {
	Pos() ast.Position
	End() ast.Position
	GetType() Type
}

// ============================================================================
// Typed Expressions
// ============================================================================

// TypedExpression represents a typed expression
type TypedExpression interface {
	TypedNode
	typedExprNode()
}

// TypedBinaryExpr represents a typed binary expression
type TypedBinaryExpr struct {
	Type     Type
	Left     TypedExpression
	Op       string
	Right    TypedExpression
	LeftPos  ast.Position
	OpPos    ast.Position
	RightPos ast.Position
}

func (e *TypedBinaryExpr) Pos() ast.Position { return e.LeftPos }
func (e *TypedBinaryExpr) End() ast.Position { return e.RightPos }
func (e *TypedBinaryExpr) GetType() Type     { return e.Type }
func (e *TypedBinaryExpr) typedExprNode()    {}

// TypedLiteralExpr represents a typed literal expression
type TypedLiteralExpr struct {
	Type     Type
	LitType  ast.LiteralType
	Value    string
	StartPos ast.Position
	EndPos   ast.Position
}

func (e *TypedLiteralExpr) Pos() ast.Position { return e.StartPos }
func (e *TypedLiteralExpr) End() ast.Position { return e.EndPos }
func (e *TypedLiteralExpr) GetType() Type     { return e.Type }
func (e *TypedLiteralExpr) typedExprNode()    {}

// TypedIdentifierExpr represents a typed identifier (variable reference)
type TypedIdentifierExpr struct {
	Type     Type
	Name     string
	StartPos ast.Position
	EndPos   ast.Position
}

func (e *TypedIdentifierExpr) Pos() ast.Position { return e.StartPos }
func (e *TypedIdentifierExpr) End() ast.Position { return e.EndPos }
func (e *TypedIdentifierExpr) GetType() Type     { return e.Type }
func (e *TypedIdentifierExpr) typedExprNode()    {}

// TypedCallExpr represents a typed function call expression
type TypedCallExpr struct {
	Type       Type
	Name       string
	NamePos    ast.Position
	LeftParen  ast.Position
	Arguments  []TypedExpression
	RightParen ast.Position
}

func (e *TypedCallExpr) Pos() ast.Position { return e.NamePos }
func (e *TypedCallExpr) End() ast.Position { return e.RightParen }
func (e *TypedCallExpr) GetType() Type     { return e.Type }
func (e *TypedCallExpr) typedExprNode()    {}

// ============================================================================
// Typed Statements
// ============================================================================

// TypedStatement represents a typed statement
type TypedStatement interface {
	TypedNode
	typedStmtNode()
}

// TypedExprStmt represents a typed expression statement
type TypedExprStmt struct {
	Expr TypedExpression
}

func (s *TypedExprStmt) Pos() ast.Position { return s.Expr.Pos() }
func (s *TypedExprStmt) End() ast.Position { return s.Expr.End() }
func (s *TypedExprStmt) GetType() Type     { return TypeVoid }
func (s *TypedExprStmt) typedStmtNode()    {}

// TypedBlockStmt represents a typed block statement
type TypedBlockStmt struct {
	LeftBrace  ast.Position
	Statements []TypedStatement
	RightBrace ast.Position
}

func (s *TypedBlockStmt) Pos() ast.Position { return s.LeftBrace }
func (s *TypedBlockStmt) End() ast.Position { return s.RightBrace }
func (s *TypedBlockStmt) GetType() Type     { return TypeVoid }
func (s *TypedBlockStmt) typedStmtNode()    {}

// TypedVarDeclStmt represents a typed variable declaration
type TypedVarDeclStmt struct {
	Keyword      ast.Position
	Mutable      bool
	Name         string
	NamePos      ast.Position
	Colon        ast.Position // position of ':' (zero if no type annotation)
	TypeName     string       // declared type name (empty if no annotation)
	TypePos      ast.Position // position of type name (zero if no annotation)
	DeclaredType Type         // the declared or inferred type
	Equals       ast.Position
	Initializer  TypedExpression
}

func (s *TypedVarDeclStmt) Pos() ast.Position { return s.Keyword }
func (s *TypedVarDeclStmt) End() ast.Position { return s.Initializer.End() }
func (s *TypedVarDeclStmt) GetType() Type     { return TypeVoid }
func (s *TypedVarDeclStmt) typedStmtNode()    {}

// TypedAssignStmt represents a typed variable assignment
type TypedAssignStmt struct {
	Name    string
	NamePos ast.Position
	Equals  ast.Position
	Value   TypedExpression
	VarType Type // the variable's declared type
}

func (s *TypedAssignStmt) Pos() ast.Position { return s.NamePos }
func (s *TypedAssignStmt) End() ast.Position { return s.Value.End() }
func (s *TypedAssignStmt) GetType() Type     { return TypeVoid }
func (s *TypedAssignStmt) typedStmtNode()    {}

// TypedReturnStmt represents a typed return statement
type TypedReturnStmt struct {
	Keyword ast.Position
	Value   TypedExpression // nil for void returns
}

func (s *TypedReturnStmt) Pos() ast.Position { return s.Keyword }
func (s *TypedReturnStmt) End() ast.Position {
	if s.Value != nil {
		return s.Value.End()
	}
	return ast.Position{Line: s.Keyword.Line, Column: s.Keyword.Column + 6, Offset: s.Keyword.Offset + 6}
}
func (s *TypedReturnStmt) GetType() Type  { return TypeVoid }
func (s *TypedReturnStmt) typedStmtNode() {}

// ============================================================================
// Typed Declarations
// ============================================================================

// TypedDeclaration represents a typed declaration
type TypedDeclaration interface {
	TypedNode
	typedDeclNode()
}

// TypedParameter represents a typed function parameter
type TypedParameter struct {
	Name     string
	NamePos  ast.Position
	Colon    ast.Position
	Type     Type
	TypePos  ast.Position
}

// TypedFunctionDecl represents a typed function declaration
type TypedFunctionDecl struct {
	FnKeyword  ast.Position
	Name       string
	NamePos    ast.Position
	LeftParen  ast.Position
	Parameters []TypedParameter
	RightParen ast.Position
	ReturnType Type
	ReturnPos  ast.Position
	Body       *TypedBlockStmt
}

func (d *TypedFunctionDecl) Pos() ast.Position { return d.FnKeyword }
func (d *TypedFunctionDecl) End() ast.Position { return d.Body.End() }
func (d *TypedFunctionDecl) GetType() Type     { return d.ReturnType }
func (d *TypedFunctionDecl) typedDeclNode()    {}

// ============================================================================
// Typed Program
// ============================================================================

// TypedProgram represents a typed program
type TypedProgram struct {
	Declarations []TypedDeclaration // typed function declarations
	Statements   []TypedStatement   // legacy: typed top-level statements
	StartPos     ast.Position
	EndPos       ast.Position
}

func (p *TypedProgram) Pos() ast.Position { return p.StartPos }
func (p *TypedProgram) End() ast.Position { return p.EndPos }
func (p *TypedProgram) GetType() Type     { return TypeVoid }

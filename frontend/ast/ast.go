package ast

// Position represents a position in the source code
type Position struct {
	Line   int // line number (1-indexed)
	Column int // column number (1-indexed)
	Offset int // byte offset (0-indexed)
}

// Node is the base interface for all AST nodes
type Node interface {
	Pos() Position // starting position
	End() Position // ending position
}

// ============================================================================
// Expressions
// ============================================================================

// Expression represents any expression node
type Expression interface {
	Node
	exprNode() // marker method
}

// BinaryExpr represents a binary operation (e.g., 2 + 3, x == y)
type BinaryExpr struct {
	Left     Expression
	Op       string
	Right    Expression
	LeftPos  Position // position of left operand
	OpPos    Position // position of operator
	RightPos Position // position of right operand
}

func (b *BinaryExpr) Pos() Position { return b.LeftPos }
func (b *BinaryExpr) End() Position { return b.RightPos }
func (b *BinaryExpr) exprNode()     {}

// LiteralExpr represents a literal value (number, string, boolean)
type LiteralExpr struct {
	Kind     LiteralType
	Value    string
	StartPos Position
	EndPos   Position
}

func (l *LiteralExpr) Pos() Position { return l.StartPos }
func (l *LiteralExpr) End() Position { return l.EndPos }
func (l *LiteralExpr) exprNode()     {}

// LiteralType represents the type of a literal
type LiteralType int

const (
	LiteralTypeNumber LiteralType = iota
	LiteralTypeString
	LiteralTypeBoolean
)

// UnaryExpr represents a unary operation (e.g., -5, !true)
type UnaryExpr struct {
	Op          string
	Operand     Expression
	OpPos       Position // position of operator
	OperandPos  Position // position of operand
	OperandEnd  Position // end position of operand
}

func (u *UnaryExpr) Pos() Position { return u.OpPos }
func (u *UnaryExpr) End() Position { return u.OperandEnd }
func (u *UnaryExpr) exprNode()     {}

// GroupingExpr represents a parenthesized expression (e.g., (2 + 3))
type GroupingExpr struct {
	Expr       Expression
	LeftParen  Position // position of '('
	RightParen Position // position of ')'
}

func (g *GroupingExpr) Pos() Position { return g.LeftParen }
func (g *GroupingExpr) End() Position { return g.RightParen }
func (g *GroupingExpr) exprNode()     {}

// IdentifierExpr represents a variable reference (e.g., x, myVar)
type IdentifierExpr struct {
	Name     string
	StartPos Position
	EndPos   Position
}

func (i *IdentifierExpr) Pos() Position { return i.StartPos }
func (i *IdentifierExpr) End() Position { return i.EndPos }
func (i *IdentifierExpr) exprNode()     {}

// CallExpr represents a function call (e.g., add(1, 2))
type CallExpr struct {
	Name       string       // function name
	NamePos    Position     // position of function name
	LeftParen  Position     // position of '('
	Arguments  []Expression // list of argument expressions
	RightParen Position     // position of ')'
}

func (c *CallExpr) Pos() Position { return c.NamePos }
func (c *CallExpr) End() Position { return c.RightParen }
func (c *CallExpr) exprNode()     {}

// ============================================================================
// Statements
// ============================================================================

// Statement represents any statement node
type Statement interface {
	Node
	stmtNode() // marker method
}

// ExprStmt represents an expression statement
type ExprStmt struct {
	Expr Expression
}

func (e *ExprStmt) Pos() Position { return e.Expr.Pos() }
func (e *ExprStmt) End() Position { return e.Expr.End() }
func (e *ExprStmt) stmtNode()     {}

// PrintStmt represents a print statement
type PrintStmt struct {
	Keyword Position   // position of 'print' keyword
	Expr    Expression // expression to print
}

func (p *PrintStmt) Pos() Position { return p.Keyword }
func (p *PrintStmt) End() Position { return p.Expr.End() }
func (p *PrintStmt) stmtNode()     {}

// BlockStmt represents a block of statements enclosed in braces
type BlockStmt struct {
	LeftBrace  Position    // position of '{'
	Statements []Statement // statements in the block
	RightBrace Position    // position of '}'
}

func (b *BlockStmt) Pos() Position { return b.LeftBrace }
func (b *BlockStmt) End() Position { return b.RightBrace }
func (b *BlockStmt) stmtNode()     {}

// VarDeclStmt represents a variable declaration (e.g., val x = 5 or var x = 5)
type VarDeclStmt struct {
	Keyword     Position   // position of 'val' or 'var' keyword
	Mutable     bool       // true for var, false for val
	Name        string     // variable name
	NamePos     Position   // position of variable name
	Equals      Position   // position of '='
	Initializer Expression // initializer expression
}

func (v *VarDeclStmt) Pos() Position { return v.Keyword }
func (v *VarDeclStmt) End() Position { return v.Initializer.End() }
func (v *VarDeclStmt) stmtNode()     {}

// AssignStmt represents a variable assignment (e.g., x = 5)
type AssignStmt struct {
	Name    string     // variable name
	NamePos Position   // position of variable name
	Equals  Position   // position of '='
	Value   Expression // value expression
}

func (a *AssignStmt) Pos() Position { return a.NamePos }
func (a *AssignStmt) End() Position { return a.Value.End() }
func (a *AssignStmt) stmtNode()     {}

// ReturnStmt represents a return statement (e.g., return x + 1)
type ReturnStmt struct {
	Keyword Position   // position of 'return' keyword
	Value   Expression // return value (nil for void return)
}

func (r *ReturnStmt) Pos() Position { return r.Keyword }
func (r *ReturnStmt) End() Position {
	if r.Value != nil {
		return r.Value.End()
	}
	return Position{Line: r.Keyword.Line, Column: r.Keyword.Column + 6, Offset: r.Keyword.Offset + 6}
}
func (r *ReturnStmt) stmtNode() {}

// ============================================================================
// Declarations
// ============================================================================

// Declaration represents any declaration node
type Declaration interface {
	Node
	declNode() // marker method
}

// Parameter represents a function parameter (e.g., x: int)
type Parameter struct {
	Name     string   // parameter name
	NamePos  Position // position of parameter name
	Colon    Position // position of ':'
	TypeName string   // type name (e.g., "int", "void")
	TypePos  Position // position of type name
}

func (p *Parameter) Pos() Position { return p.NamePos }
func (p *Parameter) End() Position {
	return Position{Line: p.TypePos.Line, Column: p.TypePos.Column + len(p.TypeName), Offset: p.TypePos.Offset + len(p.TypeName)}
}

// FunctionDecl represents a function declaration
type FunctionDecl struct {
	FnKeyword  Position    // position of 'fn' keyword
	Name       string      // function name
	NamePos    Position    // position of function name
	LeftParen  Position    // position of '('
	Parameters []Parameter // function parameters
	RightParen Position    // position of ')'
	ReturnType string      // return type (e.g., "int", "void")
	ReturnPos  Position    // position of return type
	Body       *BlockStmt  // function body
}

func (f *FunctionDecl) Pos() Position { return f.FnKeyword }
func (f *FunctionDecl) End() Position { return f.Body.End() }
func (f *FunctionDecl) declNode()     {}

// ============================================================================
// Program
// ============================================================================

// Program represents the root node of the AST
type Program struct {
	Declarations []Declaration // top-level declarations (functions, etc.)
	Statements   []Statement   // legacy: top-level statements (will be deprecated)
	StartPos     Position
	EndPos       Position
}

func (p *Program) Pos() Position { return p.StartPos }
func (p *Program) End() Position { return p.EndPos }

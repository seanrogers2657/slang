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

// ============================================================================
// Program
// ============================================================================

// Program represents the root node of the AST
type Program struct {
	Statements []Statement
	StartPos   Position
	EndPos     Position
}

func (p *Program) Pos() Position { return p.StartPos }
func (p *Program) End() Position { return p.EndPos }

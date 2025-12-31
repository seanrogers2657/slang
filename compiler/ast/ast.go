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
	LiteralTypeInteger LiteralType = iota
	LiteralTypeFloat
	LiteralTypeString
	LiteralTypeBoolean
)

// UnaryExpr represents a unary operation (e.g., -5, !true)
type UnaryExpr struct {
	Op         string
	Operand    Expression
	OpPos      Position // position of operator
	OperandPos Position // position of operand
	OperandEnd Position // end position of operand
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

// NamedArgument represents a named argument in a call (e.g., x: 10)
type NamedArgument struct {
	Name     string     // argument name
	NamePos  Position   // position of name
	Colon    Position   // position of ':'
	Value    Expression // argument value
}

// CallExpr represents a function call (e.g., add(1, 2)) or struct construction
type CallExpr struct {
	Name           string          // function/struct name
	NamePos        Position        // position of function/struct name
	LeftParen      Position        // position of '('
	Arguments      []Expression    // list of positional argument expressions
	NamedArguments []NamedArgument // list of named arguments (e.g., x: 10, y: 20)
	RightParen     Position        // position of ')'
}

func (c *CallExpr) Pos() Position { return c.NamePos }
func (c *CallExpr) End() Position { return c.RightParen }
func (c *CallExpr) exprNode()     {}

// HasNamedArguments returns true if the call uses named arguments
func (c *CallExpr) HasNamedArguments() bool { return len(c.NamedArguments) > 0 }

// StructLiteral represents a struct instantiation with braces (e.g., Point { 10, 20 } or Point { x: 10, y: 20 })
type StructLiteral struct {
	Name           string          // struct name
	NamePos        Position        // position of struct name
	LeftBrace      Position        // position of '{'
	Arguments      []Expression    // list of positional argument expressions
	NamedArguments []NamedArgument // list of named arguments (e.g., x: 10, y: 20)
	RightBrace     Position        // position of '}'
}

func (s *StructLiteral) Pos() Position { return s.NamePos }
func (s *StructLiteral) End() Position { return s.RightBrace }
func (s *StructLiteral) exprNode()     {}

// HasNamedArguments returns true if the struct literal uses named arguments
func (s *StructLiteral) HasNamedArguments() bool { return len(s.NamedArguments) > 0 }

// AnonStructLiteral represents an anonymous struct literal without a type name (e.g., { x: 0, y: 0 })
// Used when the type is inferred from context (e.g., val p: Point = { x: 0, y: 0 })
type AnonStructLiteral struct {
	LeftBrace      Position        // position of '{'
	Arguments      []Expression    // list of positional argument expressions
	NamedArguments []NamedArgument // list of named arguments (e.g., x: 10, y: 20)
	RightBrace     Position        // position of '}'
}

func (a *AnonStructLiteral) Pos() Position { return a.LeftBrace }
func (a *AnonStructLiteral) End() Position { return a.RightBrace }
func (a *AnonStructLiteral) exprNode()     {}

// HasNamedArguments returns true if the anonymous struct literal uses named arguments
func (a *AnonStructLiteral) HasNamedArguments() bool { return len(a.NamedArguments) > 0 }

// FieldAccessExpr represents field access (e.g., p.x, rect.topLeft.x)
type FieldAccessExpr struct {
	Object   Expression // the struct expression
	Dot      Position   // position of '.'
	Field    string     // field name
	FieldPos Position   // position of field name
}

func (f *FieldAccessExpr) Pos() Position { return f.Object.Pos() }
func (f *FieldAccessExpr) End() Position {
	return Position{Line: f.FieldPos.Line, Column: f.FieldPos.Column + len(f.Field), Offset: f.FieldPos.Offset + len(f.Field)}
}
func (f *FieldAccessExpr) exprNode() {}

// ArrayLiteralExpr represents an array literal (e.g., [1, 2, 3])
type ArrayLiteralExpr struct {
	LeftBracket  Position     // position of '['
	Elements     []Expression // element expressions
	RightBracket Position     // position of ']'
}

func (a *ArrayLiteralExpr) Pos() Position { return a.LeftBracket }
func (a *ArrayLiteralExpr) End() Position { return a.RightBracket }
func (a *ArrayLiteralExpr) exprNode()     {}

// IndexExpr represents an array index access (e.g., arr[0])
type IndexExpr struct {
	Array        Expression // the array expression
	LeftBracket  Position   // position of '['
	Index        Expression // the index expression
	RightBracket Position   // position of ']'
}

func (i *IndexExpr) Pos() Position { return i.Array.Pos() }
func (i *IndexExpr) End() Position { return i.RightBracket }
func (i *IndexExpr) exprNode()     {}

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

// BlockStmt represents a block of statements enclosed in braces
type BlockStmt struct {
	LeftBrace  Position    // position of '{'
	Statements []Statement // statements in the block
	RightBrace Position    // position of '}'
}

func (b *BlockStmt) Pos() Position { return b.LeftBrace }
func (b *BlockStmt) End() Position { return b.RightBrace }
func (b *BlockStmt) stmtNode()     {}

// VarDeclStmt represents a variable declaration (e.g., val x = 5 or val x: i32 = 5)
type VarDeclStmt struct {
	Keyword     Position   // position of 'val' or 'var' keyword
	Mutable     bool       // true for var, false for val
	Name        string     // variable name
	NamePos     Position   // position of variable name
	Colon       Position   // position of ':' (zero value if no type annotation)
	TypeName    string     // type name (empty string if no type annotation)
	TypePos     Position   // position of type name (zero value if no type annotation)
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

// FieldAssignStmt represents a field assignment (e.g., p.y = 25)
type FieldAssignStmt struct {
	Object   Expression // the struct expression (could be nested: rect.topLeft)
	Dot      Position   // position of '.'
	Field    string     // field name
	FieldPos Position   // position of field name
	Equals   Position   // position of '='
	Value    Expression // value expression
}

func (f *FieldAssignStmt) Pos() Position { return f.Object.Pos() }
func (f *FieldAssignStmt) End() Position { return f.Value.End() }
func (f *FieldAssignStmt) stmtNode()     {}

// IndexAssignStmt represents an array index assignment (e.g., arr[0] = 5)
type IndexAssignStmt struct {
	Array        Expression // the array expression
	LeftBracket  Position   // position of '['
	Index        Expression // the index expression
	RightBracket Position   // position of ']'
	Equals       Position   // position of '='
	Value        Expression // the value being assigned
}

func (i *IndexAssignStmt) Pos() Position { return i.Array.Pos() }
func (i *IndexAssignStmt) End() Position { return i.Value.End() }
func (i *IndexAssignStmt) stmtNode()     {}

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

// IfStmt represents an if statement with optional else/else-if branches
type IfStmt struct {
	IfKeyword   Position   // position of 'if'
	Condition   Expression // boolean condition
	ThenBranch  *BlockStmt // required block for true case
	ElseKeyword Position   // position of 'else' (zero if none)
	ElseBranch  Statement  // optional: either *BlockStmt or *IfStmt for else-if
}

func (i *IfStmt) Pos() Position { return i.IfKeyword }
func (i *IfStmt) End() Position {
	if i.ElseBranch != nil {
		return i.ElseBranch.End()
	}
	return i.ThenBranch.End()
}
func (i *IfStmt) stmtNode() {}
func (i *IfStmt) exprNode() {} // IfStmt can also be used as an expression

// ForStmt represents a for-loop statement
// Supports both with and without parentheses:
//   for (var i = 0; i < 10; i = i + 1) { ... }
//   for var i = 0; i < 10; i = i + 1 { ... }
type ForStmt struct {
	ForKeyword Position   // position of 'for'
	HasParens  bool       // true if loop uses parentheses syntax
	LeftParen  Position   // position of '(' if HasParens
	Init       Statement  // initialization (VarDeclStmt or AssignStmt), may be nil
	Condition  Expression // loop condition, may be nil (infinite loop)
	Update     Statement  // update statement (AssignStmt), may be nil
	RightParen Position   // position of ')' if HasParens
	Body       *BlockStmt // loop body
}

func (f *ForStmt) Pos() Position { return f.ForKeyword }
func (f *ForStmt) End() Position { return f.Body.End() }
func (f *ForStmt) stmtNode()     {}

// WhileStmt represents a while-loop statement
// Supports both with and without parentheses:
//
//	while (i < 10) { ... }
//	while i < 10 { ... }
type WhileStmt struct {
	WhileKeyword Position   // position of 'while'
	HasParens    bool       // true if loop uses parentheses syntax
	LeftParen    Position   // position of '(' if HasParens
	Condition    Expression // loop condition (required)
	RightParen   Position   // position of ')' if HasParens
	Body         *BlockStmt // loop body
}

func (w *WhileStmt) Pos() Position { return w.WhileKeyword }
func (w *WhileStmt) End() Position { return w.Body.End() }
func (w *WhileStmt) stmtNode()     {}

// BreakStmt represents a break statement
type BreakStmt struct {
	Keyword Position // position of 'break'
}

func (b *BreakStmt) Pos() Position { return b.Keyword }
func (b *BreakStmt) End() Position {
	return Position{Line: b.Keyword.Line, Column: b.Keyword.Column + 5, Offset: b.Keyword.Offset + 5}
}
func (b *BreakStmt) stmtNode() {}

// ContinueStmt represents a continue statement
type ContinueStmt struct {
	Keyword Position // position of 'continue'
}

func (c *ContinueStmt) Pos() Position { return c.Keyword }
func (c *ContinueStmt) End() Position {
	return Position{Line: c.Keyword.Line, Column: c.Keyword.Column + 8, Offset: c.Keyword.Offset + 8}
}
func (c *ContinueStmt) stmtNode() {}

// WhenCase represents a single case in a when expression
// Condition is nil for else case, otherwise it's a boolean expression
type WhenCase struct {
	Condition    Expression // nil for else case; boolean condition otherwise
	ConditionPos Position   // position of condition start (or 'else' keyword)
	Arrow        Position   // position of '->'
	Body         Statement  // can be BlockStmt or ExprStmt
	IsElse       bool       // true if this is the else case
}

func (w *WhenCase) Pos() Position { return w.ConditionPos }
func (w *WhenCase) End() Position { return w.Body.End() }

// WhenExpr represents a when expression/statement
// Form: when { cond1 -> body1, cond2 -> body2, else -> body3 }
type WhenExpr struct {
	WhenKeyword Position   // position of 'when'
	LeftBrace   Position   // position of '{'
	Cases       []WhenCase // list of cases
	RightBrace  Position   // position of '}'
}

func (w *WhenExpr) Pos() Position { return w.WhenKeyword }
func (w *WhenExpr) End() Position { return w.RightBrace }
func (w *WhenExpr) exprNode()     {} // WhenExpr can be used as an expression
func (w *WhenExpr) stmtNode()     {} // WhenExpr can also be used as a statement

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
// Syntax: name = (params) -> returnType { body }
// Return type is optional (defaults to void): name = (params) { body }
type FunctionDecl struct {
	Name       string      // function name
	NamePos    Position    // position of function name
	EqualsPos  Position    // position of '='
	LeftParen  Position    // position of '('
	Parameters []Parameter // function parameters
	RightParen Position    // position of ')'
	ArrowPos   Position    // position of '->' (zero if no return type specified)
	ReturnType string      // return type (e.g., "int", "void", empty = void)
	ReturnPos  Position    // position of return type (zero if no return type)
	Body       *BlockStmt  // function body
}

func (f *FunctionDecl) Pos() Position { return f.NamePos }
func (f *FunctionDecl) End() Position { return f.Body.End() }
func (f *FunctionDecl) declNode()     {}

// StructField represents a field in a struct definition (e.g., val x: i64, var y: i64)
type StructField struct {
	Mutable    bool     // true for var, false for val
	KeywordPos Position // position of 'val' or 'var' keyword
	Name       string   // field name
	NamePos    Position // position of field name
	Colon      Position // position of ':'
	TypeName   string   // type name (e.g., "i64", "Point")
	TypePos    Position // position of type name
}

func (f *StructField) Pos() Position { return f.KeywordPos }
func (f *StructField) End() Position {
	return Position{Line: f.TypePos.Line, Column: f.TypePos.Column + len(f.TypeName), Offset: f.TypePos.Offset + len(f.TypeName)}
}

// StructDecl represents a struct declaration
// Syntax: Name = struct { fields }
type StructDecl struct {
	Name          string        // struct name
	NamePos       Position      // position of struct name
	EqualsPos     Position      // position of '='
	StructKeyword Position      // position of 'struct' keyword
	LeftBrace     Position      // position of '{'
	Fields        []StructField // list of fields
	RightBrace    Position      // position of '}'
}

func (s *StructDecl) Pos() Position { return s.NamePos }
func (s *StructDecl) End() Position { return s.RightBrace }
func (s *StructDecl) declNode()     {}

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

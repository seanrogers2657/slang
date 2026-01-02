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

// TypedUnaryExpr represents a typed unary expression (e.g., !x)
type TypedUnaryExpr struct {
	Type       Type
	Op         string // "!"
	Operand    TypedExpression
	OpPos      ast.Position
	OperandEnd ast.Position
}

func (e *TypedUnaryExpr) Pos() ast.Position { return e.OpPos }
func (e *TypedUnaryExpr) End() ast.Position { return e.OperandEnd }
func (e *TypedUnaryExpr) GetType() Type     { return e.Type }
func (e *TypedUnaryExpr) typedExprNode()    {}

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

// TypedFieldAccessExpr represents a typed field access expression (e.g., p.x)
type TypedFieldAccessExpr struct {
	Type     Type            // the type of the field being accessed
	Object   TypedExpression // the struct expression
	Dot      ast.Position    // position of '.'
	Field    string          // field name
	FieldPos ast.Position    // position of field name
	Mutable  bool            // whether the field is mutable (for assignment checking)
}

func (e *TypedFieldAccessExpr) Pos() ast.Position { return e.Object.Pos() }
func (e *TypedFieldAccessExpr) End() ast.Position {
	return ast.Position{Line: e.FieldPos.Line, Column: e.FieldPos.Column + len(e.Field), Offset: e.FieldPos.Offset + len(e.Field)}
}
func (e *TypedFieldAccessExpr) GetType() Type  { return e.Type }
func (e *TypedFieldAccessExpr) typedExprNode() {}

// TypedSafeCallExpr represents a typed safe call expression (e.g., person?.address)
// Result is always nullable: if object is null, returns null; otherwise returns field value
type TypedSafeCallExpr struct {
	Type        Type            // the result type (always nullable)
	Object      TypedExpression // the nullable struct expression
	SafeCallPos ast.Position    // position of '?.'
	Field       string          // field name
	FieldPos    ast.Position    // position of field name
	FieldOffset int             // byte offset of field in struct (for codegen)
	InnerType   Type            // the unwrapped inner type of the nullable object
}

func (e *TypedSafeCallExpr) Pos() ast.Position { return e.Object.Pos() }
func (e *TypedSafeCallExpr) End() ast.Position {
	return ast.Position{Line: e.FieldPos.Line, Column: e.FieldPos.Column + len(e.Field), Offset: e.FieldPos.Offset + len(e.Field)}
}
func (e *TypedSafeCallExpr) GetType() Type  { return e.Type }
func (e *TypedSafeCallExpr) typedExprNode() {}

// TypedStructLiteralExpr represents a typed struct literal (construction)
type TypedStructLiteralExpr struct {
	Type       StructType        // the struct type being constructed
	TypePos    ast.Position      // position of struct name
	LeftBrace  ast.Position      // position of '{'
	Args       []TypedExpression // typed argument expressions (in field order)
	RightBrace ast.Position      // position of '}'
}

func (e *TypedStructLiteralExpr) Pos() ast.Position { return e.TypePos }
func (e *TypedStructLiteralExpr) End() ast.Position { return e.RightBrace }
func (e *TypedStructLiteralExpr) GetType() Type     { return e.Type }
func (e *TypedStructLiteralExpr) typedExprNode()    {}

// TypedArrayLiteralExpr represents a typed array literal (e.g., [1, 2, 3])
type TypedArrayLiteralExpr struct {
	Type         ArrayType         // the array type (with size and element type)
	LeftBracket  ast.Position      // position of '['
	Elements     []TypedExpression // typed element expressions
	RightBracket ast.Position      // position of ']'
}

func (e *TypedArrayLiteralExpr) Pos() ast.Position { return e.LeftBracket }
func (e *TypedArrayLiteralExpr) End() ast.Position { return e.RightBracket }
func (e *TypedArrayLiteralExpr) GetType() Type     { return e.Type }
func (e *TypedArrayLiteralExpr) typedExprNode()    {}

// TypedIndexExpr represents a typed array index access (e.g., arr[0])
type TypedIndexExpr struct {
	Type         Type            // the element type
	Array        TypedExpression // typed array expression
	LeftBracket  ast.Position    // position of '['
	Index        TypedExpression // typed index expression (must be integer)
	RightBracket ast.Position    // position of ']'
	ArraySize    int             // array size for runtime bounds checking
}

func (e *TypedIndexExpr) Pos() ast.Position { return e.Array.Pos() }
func (e *TypedIndexExpr) End() ast.Position { return e.RightBracket }
func (e *TypedIndexExpr) GetType() Type     { return e.Type }
func (e *TypedIndexExpr) typedExprNode()    {}

// TypedLenExpr represents a len() call on an array
type TypedLenExpr struct {
	Type       Type            // always TypeI64
	Array      TypedExpression // the array expression
	ArraySize  int             // known at compile time for fixed-size arrays
	NamePos    ast.Position    // position of 'len'
	LeftParen  ast.Position    // position of '('
	RightParen ast.Position    // position of ')'
}

func (e *TypedLenExpr) Pos() ast.Position { return e.NamePos }
func (e *TypedLenExpr) End() ast.Position { return e.RightParen }
func (e *TypedLenExpr) GetType() Type     { return e.Type }
func (e *TypedLenExpr) typedExprNode()    {}

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

// TypedFieldAssignStmt represents a typed field assignment (e.g., p.y = 25)
type TypedFieldAssignStmt struct {
	Object   TypedExpression // the struct expression (could be nested: rect.topLeft)
	Dot      ast.Position    // position of '.'
	Field    string          // field name
	FieldPos ast.Position    // position of field name
	Equals   ast.Position    // position of '='
	Value    TypedExpression // value expression
}

func (s *TypedFieldAssignStmt) Pos() ast.Position { return s.Object.Pos() }
func (s *TypedFieldAssignStmt) End() ast.Position { return s.Value.End() }
func (s *TypedFieldAssignStmt) GetType() Type     { return TypeVoid }
func (s *TypedFieldAssignStmt) typedStmtNode()    {}

// TypedIndexAssignStmt represents a typed array index assignment (e.g., arr[0] = 5)
type TypedIndexAssignStmt struct {
	Array        TypedExpression // the array expression
	LeftBracket  ast.Position    // position of '['
	Index        TypedExpression // typed index expression
	RightBracket ast.Position    // position of ']'
	Equals       ast.Position    // position of '='
	Value        TypedExpression // the value being assigned
	ArraySize    int             // array size for runtime bounds checking
}

func (s *TypedIndexAssignStmt) Pos() ast.Position { return s.Array.Pos() }
func (s *TypedIndexAssignStmt) End() ast.Position { return s.Value.End() }
func (s *TypedIndexAssignStmt) GetType() Type     { return TypeVoid }
func (s *TypedIndexAssignStmt) typedStmtNode()    {}

// TypedReturnStmt represents a typed return statement
type TypedReturnStmt struct {
	Keyword      ast.Position
	Value        TypedExpression // nil for void returns
	ExpectedType Type            // the function's declared return type (for nullable handling)
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

// TypedIfStmt represents a typed if statement/expression
type TypedIfStmt struct {
	IfKeyword   ast.Position
	Condition   TypedExpression
	ThenBranch  *TypedBlockStmt
	ElseKeyword ast.Position
	ElseBranch  TypedStatement // nil, *TypedBlockStmt, or *TypedIfStmt (else-if)
	ResultType  Type           // non-nil when used as expression
}

func (s *TypedIfStmt) Pos() ast.Position { return s.IfKeyword }
func (s *TypedIfStmt) End() ast.Position {
	if s.ElseBranch != nil {
		return s.ElseBranch.End()
	}
	return s.ThenBranch.End()
}
func (s *TypedIfStmt) GetType() Type {
	if s.ResultType != nil {
		return s.ResultType
	}
	return TypeVoid
}
func (s *TypedIfStmt) typedStmtNode() {}
func (s *TypedIfStmt) typedExprNode() {} // TypedIfStmt can also be used as an expression

// TypedForStmt represents a typed for-loop statement
type TypedForStmt struct {
	ForKeyword ast.Position
	Init       TypedStatement  // nil if no init
	Condition  TypedExpression // nil if no condition (infinite loop)
	Update     TypedStatement  // nil if no update
	Body       *TypedBlockStmt
}

func (s *TypedForStmt) Pos() ast.Position { return s.ForKeyword }
func (s *TypedForStmt) End() ast.Position { return s.Body.End() }
func (s *TypedForStmt) GetType() Type     { return TypeVoid }
func (s *TypedForStmt) typedStmtNode()    {}

// TypedWhileStmt represents a typed while-loop statement
type TypedWhileStmt struct {
	WhileKeyword ast.Position
	Condition    TypedExpression // required for while loops
	Body         *TypedBlockStmt
}

func (s *TypedWhileStmt) Pos() ast.Position { return s.WhileKeyword }
func (s *TypedWhileStmt) End() ast.Position { return s.Body.End() }
func (s *TypedWhileStmt) GetType() Type     { return TypeVoid }
func (s *TypedWhileStmt) typedStmtNode()    {}

// TypedBreakStmt represents a typed break statement
type TypedBreakStmt struct {
	Keyword ast.Position
}

func (s *TypedBreakStmt) Pos() ast.Position { return s.Keyword }
func (s *TypedBreakStmt) End() ast.Position {
	return ast.Position{Line: s.Keyword.Line, Column: s.Keyword.Column + 5, Offset: s.Keyword.Offset + 5}
}
func (s *TypedBreakStmt) GetType() Type  { return TypeVoid }
func (s *TypedBreakStmt) typedStmtNode() {}

// TypedContinueStmt represents a typed continue statement
type TypedContinueStmt struct {
	Keyword ast.Position
}

func (s *TypedContinueStmt) Pos() ast.Position { return s.Keyword }
func (s *TypedContinueStmt) End() ast.Position {
	return ast.Position{Line: s.Keyword.Line, Column: s.Keyword.Column + 8, Offset: s.Keyword.Offset + 8}
}
func (s *TypedContinueStmt) GetType() Type  { return TypeVoid }
func (s *TypedContinueStmt) typedStmtNode() {}

// TypedWhenCase represents a type-checked when case
type TypedWhenCase struct {
	Condition    TypedExpression // typed condition (nil for else)
	ConditionPos ast.Position
	Arrow        ast.Position
	Body         TypedStatement // *TypedBlockStmt or *TypedExprStmt
	IsElse       bool
}

// TypedWhenExpr represents a type-checked when expression/statement
type TypedWhenExpr struct {
	WhenKeyword ast.Position
	Cases       []TypedWhenCase
	RightBrace  ast.Position
	ResultType  Type // non-nil when used as expression
}

func (s *TypedWhenExpr) Pos() ast.Position { return s.WhenKeyword }
func (s *TypedWhenExpr) End() ast.Position { return s.RightBrace }
func (s *TypedWhenExpr) GetType() Type {
	if s.ResultType != nil {
		return s.ResultType
	}
	return TypeVoid
}
func (s *TypedWhenExpr) typedStmtNode() {}
func (s *TypedWhenExpr) typedExprNode() {} // TypedWhenExpr can also be used as an expression

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
	Name    string
	NamePos ast.Position
	Colon   ast.Position
	Type    Type
	TypePos ast.Position
}

// TypedFunctionDecl represents a typed function declaration
// Syntax: name = (params) -> returnType { body }
type TypedFunctionDecl struct {
	Name       string
	NamePos    ast.Position
	EqualsPos  ast.Position
	LeftParen  ast.Position
	Parameters []TypedParameter
	RightParen ast.Position
	ArrowPos   ast.Position // position of '->' (zero if no return type specified)
	ReturnType Type
	ReturnPos  ast.Position
	Body       *TypedBlockStmt
}

func (d *TypedFunctionDecl) Pos() ast.Position { return d.NamePos }
func (d *TypedFunctionDecl) End() ast.Position { return d.Body.End() }
func (d *TypedFunctionDecl) GetType() Type     { return d.ReturnType }
func (d *TypedFunctionDecl) typedDeclNode()    {}

// TypedStructDecl represents a typed struct declaration
// Syntax: Name = struct { fields }
type TypedStructDecl struct {
	Name          string
	NamePos       ast.Position
	EqualsPos     ast.Position
	StructKeyword ast.Position
	LeftBrace     ast.Position
	StructType    StructType // the full struct type with field info
	RightBrace    ast.Position
}

func (d *TypedStructDecl) Pos() ast.Position { return d.NamePos }
func (d *TypedStructDecl) End() ast.Position { return d.RightBrace }
func (d *TypedStructDecl) GetType() Type     { return d.StructType }
func (d *TypedStructDecl) typedDeclNode()    {}

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

package semantic

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
)

// -----------------------------------------------------------------------------
// AST Builder Helpers
// -----------------------------------------------------------------------------

func pos(line, col int) ast.Position {
	return ast.Position{Line: line, Column: col}
}

func intLit(value string) *ast.LiteralExpr {
	return &ast.LiteralExpr{
		Kind:     ast.LiteralTypeInteger,
		Value:    value,
		StartPos: pos(1, 1),
		EndPos:   pos(1, len(value)),
	}
}

func strLit(value string) *ast.LiteralExpr {
	return &ast.LiteralExpr{
		Kind:     ast.LiteralTypeString,
		Value:    value,
		StartPos: pos(1, 1),
		EndPos:   pos(1, len(value)),
	}
}

func ident(name string) *ast.IdentifierExpr {
	return &ast.IdentifierExpr{
		Name:     name,
		StartPos: pos(1, 1),
		EndPos:   pos(1, len(name)),
	}
}

func binExpr(left ast.Expression, op string, right ast.Expression) *ast.BinaryExpr {
	return &ast.BinaryExpr{
		Left:     left,
		Op:       op,
		Right:    right,
		LeftPos:  pos(1, 1),
		OpPos:    pos(1, 3),
		RightPos: pos(1, 5),
	}
}

func varDecl(name string, mutable bool, init ast.Expression) *ast.VarDeclStmt {
	return &ast.VarDeclStmt{
		Keyword:     pos(1, 1),
		Mutable:     mutable,
		Name:        name,
		NamePos:     pos(1, 5),
		Equals:      pos(1, 7),
		Initializer: init,
	}
}

func typedVarDecl(name string, typeName string, mutable bool, init ast.Expression) *ast.VarDeclStmt {
	return &ast.VarDeclStmt{
		Keyword:     pos(1, 1),
		Mutable:     mutable,
		Name:        name,
		NamePos:     pos(1, 5),
		Colon:       pos(1, 7),
		TypeName:    typeName,
		TypePos:     pos(1, 9),
		Equals:      pos(1, 12),
		Initializer: init,
	}
}

func floatLit(value string) *ast.LiteralExpr {
	return &ast.LiteralExpr{
		Kind:     ast.LiteralTypeFloat,
		Value:    value,
		StartPos: pos(1, 1),
		EndPos:   pos(1, len(value)),
	}
}

func boolLit(value string) *ast.LiteralExpr {
	return &ast.LiteralExpr{
		Kind:     ast.LiteralTypeBoolean,
		Value:    value,
		StartPos: pos(1, 1),
		EndPos:   pos(1, len(value)),
	}
}

func nullLit() *ast.LiteralExpr {
	return &ast.LiteralExpr{
		Kind:     ast.LiteralTypeNull,
		Value:    "null",
		StartPos: pos(1, 1),
		EndPos:   pos(1, 4),
	}
}

func safeCallExpr(object ast.Expression, field string) *ast.SafeCallExpr {
	return &ast.SafeCallExpr{
		Object:      object,
		SafeCallPos: pos(1, 2),
		Field:       field,
		FieldPos:    pos(1, 4),
	}
}

func unaryExpr(op string, operand ast.Expression) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		Op:         op,
		Operand:    operand,
		OpPos:      pos(1, 1),
		OperandPos: pos(1, 2),
		OperandEnd: pos(1, 6),
	}
}

func assignStmt(name string, value ast.Expression) *ast.AssignStmt {
	return &ast.AssignStmt{
		Name:    name,
		NamePos: pos(1, 1),
		Equals:  pos(1, 3),
		Value:   value,
	}
}

func exprStmt(expr ast.Expression) *ast.ExprStmt {
	return &ast.ExprStmt{Expr: expr}
}

func program(stmts ...ast.Statement) *ast.Program {
	return &ast.Program{
		Statements: stmts,
		StartPos:   pos(1, 1),
		EndPos:     pos(1, 1),
	}
}

// -----------------------------------------------------------------------------
// Test Fixture
// -----------------------------------------------------------------------------

type analyzerTest struct {
	t        *testing.T
	analyzer *Analyzer
}

func newTest(t *testing.T) *analyzerTest {
	t.Helper()
	return &analyzerTest{
		t:        t,
		analyzer: NewAnalyzer("test.sl"),
	}
}

func (at *analyzerTest) withScope() *analyzerTest {
	at.analyzer.enterScope()
	return at
}

func (at *analyzerTest) declare(name string, typ Type, mutable bool) *analyzerTest {
	at.analyzer.currentScope.declare(name, typ, mutable)
	// Also track ownership for move-only types
	if IsMoveOnly(typ) {
		at.analyzer.ownershipScope.declare(name, typ)
	}
	return at
}

func (at *analyzerTest) expectNoErrors() {
	at.t.Helper()
	if len(at.analyzer.errors) > 0 {
		at.t.Errorf("expected no errors, got %d: %s", len(at.analyzer.errors), at.analyzer.errors[0].Message)
	}
}

func (at *analyzerTest) expectErrors(n int) {
	at.t.Helper()
	if len(at.analyzer.errors) != n {
		at.t.Errorf("expected %d error(s), got %d", n, len(at.analyzer.errors))
	}
}

func (at *analyzerTest) expectErrorContaining(substr string) {
	at.t.Helper()
	for _, err := range at.analyzer.errors {
		if strings.Contains(err.Message, substr) {
			return
		}
	}
	if len(at.analyzer.errors) == 0 {
		at.t.Errorf("expected error containing %q, got no errors", substr)
	} else {
		at.t.Errorf("expected error containing %q, got: %s", substr, at.analyzer.errors[0].Message)
	}
}

func (at *analyzerTest) expectType(result TypedExpression, expected Type) {
	at.t.Helper()
	if !result.GetType().Equals(expected) {
		at.t.Errorf("expected type %s, got %s", expected, result.GetType())
	}
}

func (at *analyzerTest) expectStage(stage string) {
	at.t.Helper()
	if len(at.analyzer.errors) == 0 {
		at.t.Errorf("expected error with stage %q, got no errors", stage)
		return
	}
	if at.analyzer.errors[0].Stage != stage {
		at.t.Errorf("expected stage %q, got %q", stage, at.analyzer.errors[0].Stage)
	}
}

func groupExpr(inner ast.Expression) *ast.GroupingExpr {
	return &ast.GroupingExpr{
		Expr:       inner,
		LeftParen:  pos(1, 1),
		RightParen: pos(1, 10),
	}
}

func param(name string, typeName string) ast.Parameter {
	return ast.Parameter{
		Name:     name,
		NamePos:  pos(1, 1),
		Colon:    pos(1, 2),
		TypeName: typeName,
		TypePos:  pos(1, 4),
	}
}

func varParam(name string, typeName string) ast.Parameter {
	return ast.Parameter{
		Mutable:  true,
		VarPos:   pos(1, 1),
		Name:     name,
		NamePos:  pos(1, 5),
		Colon:    pos(1, 6),
		TypeName: typeName,
		TypePos:  pos(1, 8),
	}
}

func funcDecl(name string, returnType string, params []ast.Parameter, stmts ...ast.Statement) *ast.FunctionDecl {
	return &ast.FunctionDecl{
		Name:       name,
		NamePos:    pos(1, 1),
		EqualsPos:  pos(1, 3),
		LeftParen:  pos(1, 5),
		Parameters: params,
		RightParen: pos(1, 6),
		ArrowPos:   pos(1, 8),
		ReturnType: returnType,
		ReturnPos:  pos(1, 11),
		Body: &ast.BlockStmt{
			LeftBrace:  pos(1, 15),
			Statements: stmts,
			RightBrace: pos(1, 25),
		},
	}
}

func programWithFuncs(funcs ...*ast.FunctionDecl) *ast.Program {
	decls := make([]ast.Declaration, len(funcs))
	for i, f := range funcs {
		decls[i] = f
	}
	return &ast.Program{
		Declarations: decls,
		StartPos:     pos(1, 1),
		EndPos:       pos(1, 1),
	}
}

func callExpr(name string, args ...ast.Expression) *ast.CallExpr {
	return &ast.CallExpr{
		Name:       name,
		NamePos:    pos(1, 1),
		LeftParen:  pos(1, 2),
		Arguments:  args,
		RightParen: pos(1, 10),
	}
}

func returnStmt(value ast.Expression) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Keyword: pos(1, 1),
		Value:   value,
	}
}

func returnStmtAST(value ast.Expression) ast.Statement {
	return &ast.ReturnStmt{
		Keyword: pos(1, 1),
		Value:   value,
	}
}

func ifStmtWithElse(cond ast.Expression, thenStmts []ast.Statement, elseStmts []ast.Statement) ast.Statement {
	return &ast.IfStmt{
		IfKeyword: pos(1, 1),
		Condition: cond,
		ThenBranch: &ast.BlockStmt{
			LeftBrace:  pos(1, 5),
			Statements: thenStmts,
			RightBrace: pos(1, 10),
		},
		ElseKeyword: pos(1, 12),
		ElseBranch: &ast.BlockStmt{
			LeftBrace:  pos(1, 17),
			Statements: elseStmts,
			RightBrace: pos(1, 22),
		},
	}
}

func ifStmtNoElse(cond ast.Expression, thenStmts []ast.Statement) ast.Statement {
	return &ast.IfStmt{
		IfKeyword: pos(1, 1),
		Condition: cond,
		ThenBranch: &ast.BlockStmt{
			LeftBrace:  pos(1, 5),
			Statements: thenStmts,
			RightBrace: pos(1, 10),
		},
	}
}

func whenCase(cond ast.Expression, body ast.Statement, isElse bool) ast.WhenCase {
	return ast.WhenCase{
		Condition:    cond,
		ConditionPos: pos(1, 1),
		Arrow:        pos(1, 5),
		Body:         body,
		IsElse:       isElse,
	}
}

func whenExpr(cases ...ast.WhenCase) *ast.WhenExpr {
	return &ast.WhenExpr{
		WhenKeyword: pos(1, 1),
		LeftBrace:   pos(1, 6),
		Cases:       cases,
		RightBrace:  pos(1, 20),
	}
}

func whileStmt(cond ast.Expression, stmts ...ast.Statement) *ast.WhileStmt {
	return &ast.WhileStmt{
		WhileKeyword: pos(1, 1),
		HasParens:    false,
		Condition:    cond,
		Body: &ast.BlockStmt{
			LeftBrace:  pos(1, 10),
			Statements: stmts,
			RightBrace: pos(1, 20),
		},
	}
}

func forStmt(init ast.Statement, cond ast.Expression, update ast.Statement, stmts ...ast.Statement) *ast.ForStmt {
	return &ast.ForStmt{
		ForKeyword: pos(1, 1),
		HasParens:  false,
		Init:       init,
		Condition:  cond,
		Update:     update,
		Body: &ast.BlockStmt{
			LeftBrace:  pos(1, 10),
			Statements: stmts,
			RightBrace: pos(1, 20),
		},
	}
}

func breakStmt() *ast.BreakStmt {
	return &ast.BreakStmt{Keyword: pos(1, 1)}
}

func continueStmt() *ast.ContinueStmt {
	return &ast.ContinueStmt{Keyword: pos(1, 1)}
}

func methodCallExpr(object ast.Expression, method string, args ...ast.Expression) *ast.MethodCallExpr {
	return &ast.MethodCallExpr{
		Object:     object,
		Dot:        pos(1, 5),
		Method:     method,
		MethodPos:  pos(1, 6),
		LeftParen:  pos(1, 10),
		Arguments:  args,
		RightParen: pos(1, 15),
	}
}

func structDecl(name string, fields ...ast.StructField) *ast.StructDecl {
	return &ast.StructDecl{
		Name:          name,
		NamePos:       pos(1, 1),
		EqualsPos:     pos(1, 3),
		StructKeyword: pos(1, 5),
		LeftBrace:     pos(1, 12),
		Fields:        fields,
		RightBrace:    pos(1, 30),
	}
}

func structField(name string, typeName string, mutable bool) ast.StructField {
	return ast.StructField{
		Mutable:    mutable,
		KeywordPos: pos(1, 1),
		Name:       name,
		NamePos:    pos(1, 5),
		Colon:      pos(1, 6),
		TypeName:   typeName,
		TypePos:    pos(1, 8),
	}
}

func structLiteral(name string, args ...ast.Expression) *ast.StructLiteral {
	return &ast.StructLiteral{
		Name:       name,
		NamePos:    pos(1, 1),
		LeftBrace:  pos(1, 6),
		Arguments:  args,
		RightBrace: pos(1, 15),
	}
}

func fieldAccessExpr(object ast.Expression, field string) *ast.FieldAccessExpr {
	return &ast.FieldAccessExpr{
		Object:   object,
		Dot:      pos(1, 3),
		Field:    field,
		FieldPos: pos(1, 4),
	}
}

func fieldAssignStmt(object ast.Expression, field string, value ast.Expression) *ast.FieldAssignStmt {
	return &ast.FieldAssignStmt{
		Object:   object,
		Dot:      pos(1, 3),
		Field:    field,
		FieldPos: pos(1, 4),
		Equals:   pos(1, 6),
		Value:    value,
	}
}

// withStruct registers a struct type with the analyzer
func (at *analyzerTest) withStruct(name string, fields ...StructFieldInfo) *analyzerTest {
	at.analyzer.TypeRegistry.RegisterStruct(name, StructType{
		Name:   name,
		Fields: fields,
	})
	return at
}

// programWithDecls creates a program with mixed declarations
func programWithDecls(decls ...ast.Declaration) *ast.Program {
	return &ast.Program{
		Declarations: decls,
		StartPos:     pos(1, 1),
		EndPos:       pos(1, 1),
	}
}

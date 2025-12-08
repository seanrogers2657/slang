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

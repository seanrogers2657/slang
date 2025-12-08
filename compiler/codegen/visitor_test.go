package codegen

import (
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

func TestNewProgramInfo(t *testing.T) {
	info := NewProgramInfo()

	if info.FloatLiterals == nil {
		t.Error("FloatLiterals should be initialized")
	}
	if info.StringLiterals == nil {
		t.Error("StringLiterals should be initialized")
	}
	if info.HasPrint {
		t.Error("HasPrint should be false initially")
	}
}

func TestCollectFromTypedFunction(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedExprStmt{
					Expr: &semantic.TypedLiteralExpr{
						Value:   "3.14",
						LitType: ast.LiteralTypeFloat,
						Type:    semantic.TypeFloat64,
					},
				},
				&semantic.TypedExprStmt{
					Expr: &semantic.TypedCallExpr{
						Name: "print",
						Arguments: []semantic.TypedExpression{
							&semantic.TypedLiteralExpr{
								Value:   "hello",
								LitType: ast.LiteralTypeString,
								Type:    semantic.TypeString,
							},
						},
					},
				},
			},
		},
	}

	info.CollectFromTypedFunction(fn)

	if len(info.FloatLiterals) != 1 {
		t.Errorf("expected 1 float literal, got %d", len(info.FloatLiterals))
	}
	if len(info.StringLiterals) != 1 {
		t.Errorf("expected 1 string literal, got %d", len(info.StringLiterals))
	}
	if !info.HasPrint {
		t.Error("expected HasPrint to be true")
	}
}

func TestCollectFromTypedFunction_VarDecl(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedVarDeclStmt{
					Name: "x",
					Initializer: &semantic.TypedLiteralExpr{
						Value:   "2.71",
						LitType: ast.LiteralTypeFloat,
						Type:    semantic.TypeFloat64,
					},
				},
			},
		},
	}

	info.CollectFromTypedFunction(fn)

	if len(info.FloatLiterals) != 1 {
		t.Errorf("expected 1 float literal, got %d", len(info.FloatLiterals))
	}
}

func TestCollectFromTypedFunction_AssignStmt(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedAssignStmt{
					Name: "x",
					Value: &semantic.TypedLiteralExpr{
						Value:   "test string",
						LitType: ast.LiteralTypeString,
						Type:    semantic.TypeString,
					},
				},
			},
		},
	}

	info.CollectFromTypedFunction(fn)

	if len(info.StringLiterals) != 1 {
		t.Errorf("expected 1 string literal, got %d", len(info.StringLiterals))
	}
}

func TestCollectFromTypedFunction_ReturnStmt(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: &semantic.TypedLiteralExpr{
						Value:   "1.5",
						LitType: ast.LiteralTypeFloat,
						Type:    semantic.TypeFloat32,
					},
				},
			},
		},
	}

	info.CollectFromTypedFunction(fn)

	if len(info.FloatLiterals) != 1 {
		t.Errorf("expected 1 float literal, got %d", len(info.FloatLiterals))
	}
}

func TestCollectFromTypedFunction_BinaryExpr(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedExprStmt{
					Expr: &semantic.TypedBinaryExpr{
						Left: &semantic.TypedLiteralExpr{
							Value:   "1.0",
							LitType: ast.LiteralTypeFloat,
							Type:    semantic.TypeFloat64,
						},
						Op: "+",
						Right: &semantic.TypedLiteralExpr{
							Value:   "2.0",
							LitType: ast.LiteralTypeFloat,
							Type:    semantic.TypeFloat64,
						},
						Type: semantic.TypeFloat64,
					},
				},
			},
		},
	}

	info.CollectFromTypedFunction(fn)

	if len(info.FloatLiterals) != 2 {
		t.Errorf("expected 2 float literals, got %d", len(info.FloatLiterals))
	}
}

func TestCollectFromTypedFunction_NestedCallExpr(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedExprStmt{
					Expr: &semantic.TypedCallExpr{
						Name: "foo",
						Arguments: []semantic.TypedExpression{
							&semantic.TypedLiteralExpr{
								Value:   "nested",
								LitType: ast.LiteralTypeString,
								Type:    semantic.TypeString,
							},
						},
					},
				},
			},
		},
	}

	info.CollectFromTypedFunction(fn)

	if len(info.StringLiterals) != 1 {
		t.Errorf("expected 1 string literal, got %d", len(info.StringLiterals))
	}
	if info.HasPrint {
		t.Error("expected HasPrint to be false for non-print call")
	}
}

func TestCollectFromASTFunction(t *testing.T) {
	info := NewProgramInfo()

	fn := &ast.FunctionDecl{
		Name: "main",
		Body: &ast.BlockStmt{
			Statements: []ast.Statement{
				&ast.ExprStmt{
					Expr: &ast.CallExpr{
						Name: "print",
						Arguments: []ast.Expression{
							&ast.LiteralExpr{
								Value: "hello world",
								Kind:  ast.LiteralTypeString,
							},
						},
					},
				},
			},
		},
	}

	stringMap := info.CollectFromASTFunction(fn)

	if len(stringMap) != 1 {
		t.Errorf("expected 1 string in map, got %d", len(stringMap))
	}
	if len(info.StringLiterals) != 1 {
		t.Errorf("expected 1 string literal, got %d", len(info.StringLiterals))
	}
	if !info.HasPrint {
		t.Error("expected HasPrint to be true")
	}
}

func TestCollectFromASTFunction_VarDecl(t *testing.T) {
	info := NewProgramInfo()

	fn := &ast.FunctionDecl{
		Name: "main",
		Body: &ast.BlockStmt{
			Statements: []ast.Statement{
				&ast.VarDeclStmt{
					Name: "msg",
					Initializer: &ast.LiteralExpr{
						Value: "test",
						Kind:  ast.LiteralTypeString,
					},
				},
			},
		},
	}

	stringMap := info.CollectFromASTFunction(fn)

	if len(stringMap) != 1 {
		t.Errorf("expected 1 string in map, got %d", len(stringMap))
	}
}

func TestCollectFromASTFunction_AssignStmt(t *testing.T) {
	info := NewProgramInfo()

	fn := &ast.FunctionDecl{
		Name: "main",
		Body: &ast.BlockStmt{
			Statements: []ast.Statement{
				&ast.AssignStmt{
					Name: "x",
					Value: &ast.LiteralExpr{
						Value: "assigned",
						Kind:  ast.LiteralTypeString,
					},
				},
			},
		},
	}

	stringMap := info.CollectFromASTFunction(fn)

	if len(stringMap) != 1 {
		t.Errorf("expected 1 string in map, got %d", len(stringMap))
	}
}

func TestCollectFromASTFunction_ReturnStmt(t *testing.T) {
	info := NewProgramInfo()

	fn := &ast.FunctionDecl{
		Name: "main",
		Body: &ast.BlockStmt{
			Statements: []ast.Statement{
				&ast.ReturnStmt{
					Value: &ast.LiteralExpr{
						Value: "returned",
						Kind:  ast.LiteralTypeString,
					},
				},
			},
		},
	}

	stringMap := info.CollectFromASTFunction(fn)

	if len(stringMap) != 1 {
		t.Errorf("expected 1 string in map, got %d", len(stringMap))
	}
}

func TestCollectFromASTFunction_BinaryExpr(t *testing.T) {
	info := NewProgramInfo()

	fn := &ast.FunctionDecl{
		Name: "main",
		Body: &ast.BlockStmt{
			Statements: []ast.Statement{
				&ast.ExprStmt{
					Expr: &ast.BinaryExpr{
						Left: &ast.LiteralExpr{
							Value: "left",
							Kind:  ast.LiteralTypeString,
						},
						Op: "+",
						Right: &ast.LiteralExpr{
							Value: "right",
							Kind:  ast.LiteralTypeString,
						},
					},
				},
			},
		},
	}

	stringMap := info.CollectFromASTFunction(fn)

	if len(stringMap) != 2 {
		t.Errorf("expected 2 strings in map, got %d", len(stringMap))
	}
}

func TestCollectFromASTFunction_DuplicateStrings(t *testing.T) {
	info := NewProgramInfo()

	literal := &ast.LiteralExpr{
		Value: "same",
		Kind:  ast.LiteralTypeString,
	}

	fn := &ast.FunctionDecl{
		Name: "main",
		Body: &ast.BlockStmt{
			Statements: []ast.Statement{
				&ast.ExprStmt{Expr: literal},
				&ast.ExprStmt{Expr: literal}, // same pointer
			},
		},
	}

	stringMap := info.CollectFromASTFunction(fn)

	// Should only have one entry since it's the same pointer
	if len(stringMap) != 1 {
		t.Errorf("expected 1 string in map (deduped), got %d", len(stringMap))
	}
}

func TestCountVariables(t *testing.T) {
	stmts := []ast.Statement{
		&ast.VarDeclStmt{Name: "x"},
		&ast.ExprStmt{Expr: &ast.LiteralExpr{Value: "1", Kind: ast.LiteralTypeInteger}},
		&ast.VarDeclStmt{Name: "y"},
		&ast.VarDeclStmt{Name: "z"},
	}

	count := CountVariables(stmts)
	if count != 3 {
		t.Errorf("expected 3 variables, got %d", count)
	}
}

func TestCountVariables_Empty(t *testing.T) {
	stmts := []ast.Statement{}
	count := CountVariables(stmts)
	if count != 0 {
		t.Errorf("expected 0 variables, got %d", count)
	}
}

func TestCountTypedVariables(t *testing.T) {
	stmts := []semantic.TypedStatement{
		&semantic.TypedVarDeclStmt{Name: "a"},
		&semantic.TypedExprStmt{},
		&semantic.TypedVarDeclStmt{Name: "b"},
	}

	count := CountTypedVariables(stmts)
	if count != 2 {
		t.Errorf("expected 2 variables, got %d", count)
	}
}

func TestFindStringLiteral(t *testing.T) {
	info := NewProgramInfo()
	info.StringLiterals = []LiteralInfo{
		{Value: "hello", Label: "str_0", Length: 5},
		{Value: "world", Label: "str_1", Length: 5},
	}

	lit, found := info.FindStringLiteral("world")
	if !found {
		t.Error("expected to find string literal")
	}
	if lit.Label != "str_1" {
		t.Errorf("expected label str_1, got %s", lit.Label)
	}

	_, found = info.FindStringLiteral("notfound")
	if found {
		t.Error("expected not to find string literal")
	}
}

func TestFindFloatLiteral(t *testing.T) {
	info := NewProgramInfo()
	info.FloatLiterals = map[string]LiteralInfo{
		"float_0": {Value: "3.14", IsF64: true},
		"float_1": {Value: "2.71", IsF64: false},
	}

	label, lit, found := info.FindFloatLiteral("3.14")
	if !found {
		t.Error("expected to find float literal")
	}
	if label != "float_0" {
		t.Errorf("expected label float_0, got %s", label)
	}
	if !lit.IsF64 {
		t.Error("expected IsF64 to be true")
	}

	_, _, found = info.FindFloatLiteral("1.0")
	if found {
		t.Error("expected not to find float literal")
	}
}

func TestCollectFromTypedFunction_NilReturnValue(t *testing.T) {
	info := NewProgramInfo()

	fn := &semantic.TypedFunctionDecl{
		Name: "main",
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: nil, // void return
				},
			},
		},
	}

	// Should not panic
	info.CollectFromTypedFunction(fn)
}

package codegen

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/frontend/ast"
)

// makeMainFunction creates a main function declaration wrapping the given statements
func makeMainFunction(stmts ...ast.Statement) *ast.FunctionDecl {
	return &ast.FunctionDecl{
		Name:       "main",
		ReturnType: "void",
		Body:       &ast.BlockStmt{Statements: stmts},
	}
}

func TestAsGeneratorInterface(t *testing.T) {
	tests := []struct {
		name            string
		expr            *ast.BinaryExpr
		expectedContent []string
		expectError     bool
	}{
		{
			name: "successful generation with addition",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedContent: []string{
				".global _start",
				"_start:",
				"bl _main",
				"mov x16, #1",
				"svc #0",
				"_main:",
				"mov x0, #2",
				"mov x1, #5",
				"add x2, x0, x1",
				"mov x0, #0",
				"ret",
			},
			expectError: false,
		},
		{
			name: "successful generation with subtraction",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
				Op:    "-",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
			},
			expectedContent: []string{
				".global _start",
				"_main:",
				"mov x0, #10",
				"mov x1, #3",
				"sub x2, x0, x1",
				"ret",
			},
			expectError: false,
		},
		{
			name: "error on unsupported operation",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
				Op:    "^",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wrap the expression in a function declaration
			program := &ast.Program{
				Declarations: []ast.Declaration{
					makeMainFunction(&ast.ExprStmt{Expr: tt.expr}),
				},
			}
			generator := NewAsGenerator(program, nil)
			output, err := generator.Generate()

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify expected content is present
			for _, expected := range tt.expectedContent {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestGenerateProgramMultipleStatements(t *testing.T) {
	tests := []struct {
		name            string
		statements      []ast.Statement
		expectedContent []string
	}{
		{
			name: "two statements",
			statements: []ast.Statement{
				&ast.ExprStmt{Expr: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
					Op:    "+",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
				}},
				&ast.ExprStmt{Expr: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
					Op:    "-",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
				}},
			},
			expectedContent: []string{
				".global _start",
				"_main:",
				"mov x0, #2",
				"mov x1, #5",
				"add x2, x0, x1",
				"mov x0, #10",
				"mov x1, #3",
				"sub x2, x0, x1",
				"ret",
			},
		},
		{
			name: "three statements with different operations",
			statements: []ast.Statement{
				&ast.ExprStmt{Expr: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "4"},
					Op:    "*",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
				}},
				&ast.ExprStmt{Expr: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
					Op:    "/",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
				}},
				&ast.ExprStmt{Expr: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
					Op:    "==",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
				}},
			},
			expectedContent: []string{
				".global _start",
				"_main:",
				"mov x0, #4",
				"mov x1, #3",
				"mul x2, x0, x1",
				"mov x0, #10",
				"mov x1, #2",
				"sdiv x2, x0, x1",
				"mov x0, #5",
				"mov x1, #5",
				"cmp x0, x1",
				"cset x2, eq",
				"ret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &ast.Program{
				Declarations: []ast.Declaration{
					makeMainFunction(tt.statements...),
				},
			}
			output, err := GenerateProgram(program, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify expected content is present
			for _, expected := range tt.expectedContent {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestGenerateVarDecl(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *ast.VarDeclStmt
		expected []string
	}{
		{
			name: "simple variable declaration",
			stmt: &ast.VarDeclStmt{
				Name:    "x",
				Mutable: false,
				Initializer: &ast.LiteralExpr{
					Kind:  ast.LiteralTypeInteger,
					Value: "42",
				},
			},
			expected: []string{
				"mov x2, #42",
				"str x2, [x29, #-16]",
			},
		},
		{
			name: "variable with expression initializer",
			stmt: &ast.VarDeclStmt{
				Name:    "result",
				Mutable: true,
				Initializer: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
					Op:    "+",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
				},
			},
			expected: []string{
				"mov x0, #10",
				"mov x1, #5",
				"add x2, x0, x1",
				"str x2, [x29, #-16]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewBaseContext(nil)
			output, err := GenerateVarDecl(tt.stmt, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expectedLine := range tt.expected {
				if !strings.Contains(output, expectedLine) {
					t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", expectedLine, output)
				}
			}
		})
	}
}

func TestGenerateAssignStmt(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(ctx *BaseContext)
		stmt     *ast.AssignStmt
		expected []string
	}{
		{
			name: "simple assignment",
			setup: func(ctx *BaseContext) {
				// Pre-declare variable to simulate it being declared earlier
				ctx.DeclareVariable("x", nil)
			},
			stmt: &ast.AssignStmt{
				Name: "x",
				Value: &ast.LiteralExpr{
					Kind:  ast.LiteralTypeInteger,
					Value: "100",
				},
			},
			expected: []string{
				"mov x2, #100",
				"str x2, [x29, #-16]",
			},
		},
		{
			name: "assignment with expression",
			setup: func(ctx *BaseContext) {
				ctx.DeclareVariable("counter", nil)
			},
			stmt: &ast.AssignStmt{
				Name: "counter",
				Value: &ast.BinaryExpr{
					Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "20"},
					Op:    "*",
					Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
				},
			},
			expected: []string{
				"mov x0, #20",
				"mov x1, #3",
				"mul x2, x0, x1",
				"str x2, [x29, #-16]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewBaseContext(nil)
			tt.setup(ctx)

			output, err := GenerateAssignStmt(tt.stmt, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expectedLine := range tt.expected {
				if !strings.Contains(output, expectedLine) {
					t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", expectedLine, output)
				}
			}
		})
	}
}

func TestGenerateAssignStmtUndefinedVariable(t *testing.T) {
	ctx := NewBaseContext(nil)
	// Don't declare any variable

	stmt := &ast.AssignStmt{
		Name: "undeclared",
		Value: &ast.LiteralExpr{
			Kind:  ast.LiteralTypeInteger,
			Value: "10",
		},
	}

	_, err := GenerateAssignStmt(stmt, ctx)
	if err == nil {
		t.Error("expected error for undefined variable, got none")
	}

	if !strings.Contains(err.Error(), "undefined variable") {
		t.Errorf("expected 'undefined variable' error, got: %v", err)
	}
}

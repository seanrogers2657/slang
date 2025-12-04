package codegen

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/frontend/ast"
)

func TestGenerateExprAddition(t *testing.T) {
	tests := []struct {
		name           string
		expr           *ast.BinaryExpr
		expectedOutput []string
	}{
		{
			name: "simple addition",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #2",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
			},
		},
		{
			name: "addition with larger numbers",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "100"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "200"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #100",
				"    mov x1, #200",
				"    add x2, x0, x1",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
			},
		},
		{
			name: "addition with zero",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "0"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #0",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := GenerateExpr(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) != len(tt.expectedOutput) {
				t.Fatalf("expected %d lines, got %d", len(tt.expectedOutput), len(lines))
			}

			for i, line := range lines {
				if line != tt.expectedOutput[i] {
					t.Errorf("line %d: expected %q, got %q", i, tt.expectedOutput[i], line)
				}
			}
		})
	}
}

func TestGenerateExprAllOperations(t *testing.T) {
	tests := []struct {
		name              string
		expr              *ast.BinaryExpr
		expectedOperation []string
	}{
		{
			name: "subtraction",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
				Op:    "-",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
			},
			expectedOperation: []string{"    sub x2, x0, x1"},
		},
		{
			name: "multiplication",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "4"},
				Op:    "*",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "7"},
			},
			expectedOperation: []string{"    mul x2, x0, x1"},
		},
		{
			name: "division",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "20"},
				Op:    "/",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "4"},
			},
			expectedOperation: []string{"    sdiv x2, x0, x1"},
		},
		{
			name: "modulo",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
				Op:    "%",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
			},
			expectedOperation: []string{
				"    sdiv x3, x0, x1",
				"    msub x2, x3, x1, x0",
			},
		},
		{
			name: "equality",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
				Op:    "==",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, eq",
			},
		},
		{
			name: "not equal",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
				Op:    "!=",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, ne",
			},
		},
		{
			name: "less than",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
				Op:    "<",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, lt",
			},
		},
		{
			name: "greater than",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "7"},
				Op:    ">",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, gt",
			},
		},
		{
			name: "less than or equal",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "3"},
				Op:    "<=",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, le",
			},
		},
		{
			name: "greater than or equal",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "7"},
				Op:    ">=",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, ge",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := GenerateExpr(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the output contains all expected operation instructions
			for _, expectedLine := range tt.expectedOperation {
				if !strings.Contains(output, expectedLine) {
					t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", expectedLine, output)
				}
			}

			// Verify basic structure is present
			if !strings.Contains(output, ".global _start") {
				t.Error("output should contain .global _start")
			}
			leftLit, ok := tt.expr.Left.(*ast.LiteralExpr)
			if ok && !strings.Contains(output, "mov x0, #"+leftLit.Value) {
				t.Error("output should load left operand")
			}
			rightLit, ok := tt.expr.Right.(*ast.LiteralExpr)
			if ok && !strings.Contains(output, "mov x1, #"+rightLit.Value) {
				t.Error("output should load right operand")
			}
		})
	}
}

func TestGenerateExprUnsupportedOperation(t *testing.T) {
	expr := &ast.BinaryExpr{
		Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
		Op:    "^",
		Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
	}

	output, err := GenerateExpr(expr)
	if err == nil {
		t.Fatal("expected error for unsupported operator, got none")
	}

	// Check error contains the unsupported operation
	if !strings.Contains(err.Error(), "unsupported") || !strings.Contains(err.Error(), "^") {
		t.Errorf("expected error about unsupported operation ^, got %q", err.Error())
	}

	if output != "" {
		t.Error("expected empty output on error")
	}
}

func TestAsGeneratorInterface(t *testing.T) {
	tests := []struct {
		name           string
		expr           *ast.BinaryExpr
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "successful generation with addition",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    b main",
				"",
				"main:",
				"    mov x0, #2",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
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
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    b main",
				"",
				"main:",
				"    mov x0, #10",
				"    mov x1, #3",
				"    sub x2, x0, x1",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
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
			// Wrap the expression in a Program
			program := &ast.Program{
				Statements: []ast.Statement{&ast.ExprStmt{Expr: tt.expr}},
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

			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) != len(tt.expectedOutput) {
				t.Fatalf("expected %d lines, got %d", len(tt.expectedOutput), len(lines))
			}

			for i, line := range lines {
				if line != tt.expectedOutput[i] {
					t.Errorf("line %d: expected %q, got %q", i, tt.expectedOutput[i], line)
				}
			}
		})
	}
}

func TestGenerateExprStructure(t *testing.T) {
	expr := &ast.BinaryExpr{
		Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "2"},
		Op:    "+",
		Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
	}

	output, err := GenerateExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("contains global directive", func(t *testing.T) {
		if !strings.Contains(output, ".global _start") {
			t.Error("output should contain .global _start directive")
		}
	})

	t.Run("contains align directive", func(t *testing.T) {
		if !strings.Contains(output, ".align 4") {
			t.Error("output should contain .align 4 directive")
		}
	})

	t.Run("contains start label", func(t *testing.T) {
		if !strings.Contains(output, "_start:") {
			t.Error("output should contain _start: label")
		}
	})

	t.Run("contains operand loads", func(t *testing.T) {
		if !strings.Contains(output, "mov x0, #2") {
			t.Error("output should load left operand into x0")
		}
		if !strings.Contains(output, "mov x1, #5") {
			t.Error("output should load right operand into x1")
		}
	})

	t.Run("contains add instruction", func(t *testing.T) {
		if !strings.Contains(output, "add x2, x0, x1") {
			t.Error("output should contain add instruction")
		}
	})

	t.Run("contains exit syscall", func(t *testing.T) {
		if !strings.Contains(output, "mov x0, #0") {
			t.Error("output should contain exit code setup")
		}
		if !strings.Contains(output, "mov x16, #1") {
			t.Error("output should contain syscall number")
		}
		if !strings.Contains(output, "svc #0") {
			t.Error("output should contain syscall instruction")
		}
	})
}

func TestGenerateProgramMultipleStatements(t *testing.T) {
	tests := []struct {
		name     string
		program  *ast.Program
		expected []string
	}{
		{
			name: "two statements",
			program: &ast.Program{
				Statements: []ast.Statement{
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
			},
			expected: []string{
				".global _start",
				".align 4",
				"_start:",
				"    b main",
				"",
				"main:",
				"    mov x0, #2",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #10",
				"    mov x1, #3",
				"    sub x2, x0, x1",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
			},
		},
		{
			name: "three statements with different operations",
			program: &ast.Program{
				Statements: []ast.Statement{
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
			},
			expected: []string{
				".global _start",
				".align 4",
				"_start:",
				"    b main",
				"",
				"main:",
				"    mov x0, #4",
				"    mov x1, #3",
				"    mul x2, x0, x1",
				"    mov x0, #10",
				"    mov x1, #2",
				"    sdiv x2, x0, x1",
				"    mov x0, #5",
				"    mov x1, #5",
				"    cmp x0, x1",
				"    cset x2, eq",
				"    mov x0, #0",
				"    mov x16, #1",
				"    svc #0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := GenerateProgram(tt.program, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) != len(tt.expected) {
				t.Fatalf("expected %d lines, got %d\nExpected:\n%v\nGot:\n%v",
					len(tt.expected), len(lines), tt.expected, lines)
			}

			for i, line := range lines {
				if line != tt.expected[i] {
					t.Errorf("line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestGenerateExprWithStrings(t *testing.T) {
	tests := []struct {
		name     string
		expr     *ast.BinaryExpr
		expected []string
	}{
		{
			name: "string on left side",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: "hello"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "5"},
			},
			expected: []string{
				".data",
				".align 3",
				"str_left:",
				`    .asciz "hello"`,
				".text",
				".global _start",
				".align 4",
				"_start:",
				"    adr x0, str_left",
				"    mov x1, #5",
				"    add x2, x0, x1",
			},
		},
		{
			name: "string on right side",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "10"},
				Op:    "-",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: "world"},
			},
			expected: []string{
				".data",
				".align 3",
				"str_right:",
				`    .asciz "world"`,
				".text",
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #10",
				"    adr x1, str_right",
				"    sub x2, x0, x1",
			},
		},
		{
			name: "strings on both sides",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: "hello"},
				Op:    "==",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: "world"},
			},
			expected: []string{
				".data",
				".align 3",
				"str_left:",
				`    .asciz "hello"`,
				"str_right:",
				`    .asciz "world"`,
				".text",
				".global _start",
				".align 4",
				"_start:",
				"    adr x0, str_left",
				"    adr x1, str_right",
				"    cmp x0, x1",
				"    cset x2, eq",
			},
		},
		{
			name: "empty string",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: ""},
				Op:    "!=",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: "test"},
			},
			expected: []string{
				".data",
				".align 3",
				"str_left:",
				`    .asciz ""`,
				"str_right:",
				`    .asciz "test"`,
				".text",
				".global _start",
				".align 4",
				"_start:",
				"    adr x0, str_left",
				"    adr x1, str_right",
				"    cmp x0, x1",
				"    cset x2, ne",
			},
		},
		{
			name: "string with escape sequences",
			expr: &ast.BinaryExpr{
				Left:  &ast.LiteralExpr{Kind: ast.LiteralTypeString, Value: "hello\nworld"},
				Op:    "+",
				Right: &ast.LiteralExpr{Kind: ast.LiteralTypeInteger, Value: "1"},
			},
			expected: []string{
				".data",
				".align 3",
				"str_left:",
				`    .asciz "hello\nworld"`,
				".text",
				".global _start",
				".align 4",
				"_start:",
				"    adr x0, str_left",
				"    mov x1, #1",
				"    add x2, x0, x1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := GenerateExpr(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify all expected lines are present in order
			for _, expectedLine := range tt.expected {
				if !strings.Contains(output, expectedLine) {
					t.Errorf("expected output to contain %q, but it didn't.\nFull output:\n%s", expectedLine, output)
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

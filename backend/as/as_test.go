package as

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/frontend/parser"
)

func TestGenerateExprAddition(t *testing.T) {
	tests := []struct {
		name           string
		expr           *parser.Expr
		expectedOutput []string
	}{
		{
			name: "simple addition",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
				Op:    "+",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #2",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #1",
				"    mov x16, #0",
				"    svc #0",
			},
		},
		{
			name: "addition with larger numbers",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "100"},
				Op:    "+",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "200"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #100",
				"    mov x1, #200",
				"    add x2, x0, x1",
				"    mov x0, #1",
				"    mov x16, #0",
				"    svc #0",
			},
		},
		{
			name: "addition with zero",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "0"},
				Op:    "+",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #0",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #1",
				"    mov x16, #0",
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
		expr              *parser.Expr
		expectedOperation []string
	}{
		{
			name: "subtraction",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "10"},
				Op:    "-",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
			},
			expectedOperation: []string{"    sub x2, x0, x1"},
		},
		{
			name: "multiplication",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "4"},
				Op:    "*",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "7"},
			},
			expectedOperation: []string{"    mul x2, x0, x1"},
		},
		{
			name: "division",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "20"},
				Op:    "/",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "4"},
			},
			expectedOperation: []string{"    sdiv x2, x0, x1"},
		},
		{
			name: "modulo",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "10"},
				Op:    "%",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
			},
			expectedOperation: []string{
				"    sdiv x3, x0, x1",
				"    msub x2, x3, x1, x0",
			},
		},
		{
			name: "equality",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
				Op:    "==",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, eq",
			},
		},
		{
			name: "not equal",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
				Op:    "!=",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, ne",
			},
		},
		{
			name: "less than",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
				Op:    "<",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, lt",
			},
		},
		{
			name: "greater than",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "7"},
				Op:    ">",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, gt",
			},
		},
		{
			name: "less than or equal",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
				Op:    "<=",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
			},
			expectedOperation: []string{
				"    cmp x0, x1",
				"    cset x2, le",
			},
		},
		{
			name: "greater than or equal",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "7"},
				Op:    ">=",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
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
			if !strings.Contains(output, "mov x0, #"+tt.expr.Left.Value) {
				t.Error("output should load left operand")
			}
			if !strings.Contains(output, "mov x1, #"+tt.expr.Right.Value) {
				t.Error("output should load right operand")
			}
		})
	}
}

func TestGenerateExprUnsupportedOperation(t *testing.T) {
	expr := &parser.Expr{
		Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
		Op:    "^",
		Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
	}

	output, err := GenerateExpr(expr)
	if err == nil {
		t.Fatal("expected error for unsupported operator, got none")
	}

	expectedError := "unsupported operation ^ when generating code"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}

	if output != "" {
		t.Error("expected empty output on error")
	}
}

func TestAsGeneratorInterface(t *testing.T) {
	tests := []struct {
		name           string
		expr           *parser.Expr
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "successful generation with addition",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
				Op:    "+",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #2",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #1",
				"    mov x16, #0",
				"    svc #0",
			},
			expectError: false,
		},
		{
			name: "successful generation with subtraction",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "10"},
				Op:    "-",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
			},
			expectedOutput: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #10",
				"    mov x1, #3",
				"    sub x2, x0, x1",
				"    mov x0, #1",
				"    mov x16, #0",
				"    svc #0",
			},
			expectError: false,
		},
		{
			name: "error on unsupported operation",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
				Op:    "^",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewAsGenerator(tt.expr)
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
	expr := &parser.Expr{
		Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
		Op:    "+",
		Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
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
		if !strings.Contains(output, "mov x0, #1") {
			t.Error("output should contain exit code setup")
		}
		if !strings.Contains(output, "mov x16, #0") {
			t.Error("output should contain syscall number")
		}
		if !strings.Contains(output, "svc #0") {
			t.Error("output should contain syscall instruction")
		}
	})
}

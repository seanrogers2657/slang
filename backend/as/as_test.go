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
			// Wrap the expression in a Program
			program := &parser.Program{
				Statements: []*parser.Expr{tt.expr},
			}
			generator := NewAsGenerator(program)
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

func TestGenerateProgramMultipleStatements(t *testing.T) {
	tests := []struct {
		name     string
		program  *parser.Program
		expected []string
	}{
		{
			name: "two statements",
			program: &parser.Program{
				Statements: []*parser.Expr{
					{
						Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
						Op:    "+",
						Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
					},
					{
						Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "10"},
						Op:    "-",
						Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
					},
				},
			},
			expected: []string{
				".global _start",
				".align 4",
				"_start:",
				"    mov x0, #2",
				"    mov x1, #5",
				"    add x2, x0, x1",
				"    mov x0, #10",
				"    mov x1, #3",
				"    sub x2, x0, x1",
				"    mov x0, #1",
				"    mov x16, #0",
				"    svc #0",
			},
		},
		{
			name: "three statements with different operations",
			program: &parser.Program{
				Statements: []*parser.Expr{
					{
						Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "4"},
						Op:    "*",
						Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "3"},
					},
					{
						Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "10"},
						Op:    "/",
						Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "2"},
					},
					{
						Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
						Op:    "==",
						Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
					},
				},
			},
			expected: []string{
				".global _start",
				".align 4",
				"_start:",
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
				"    mov x0, #1",
				"    mov x16, #0",
				"    svc #0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := GenerateProgram(tt.program)
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
		expr     *parser.Expr
		expected []string
	}{
		{
			name: "string on left side",
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeString, Value: "hello"},
				Op:    "+",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "5"},
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
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeNumber, Value: "10"},
				Op:    "-",
				Right: &parser.Literal{Type: parser.LiteralTypeString, Value: "world"},
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
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeString, Value: "hello"},
				Op:    "==",
				Right: &parser.Literal{Type: parser.LiteralTypeString, Value: "world"},
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
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeString, Value: ""},
				Op:    "!=",
				Right: &parser.Literal{Type: parser.LiteralTypeString, Value: "test"},
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
			expr: &parser.Expr{
				Left:  &parser.Literal{Type: parser.LiteralTypeString, Value: "hello\nworld"},
				Op:    "+",
				Right: &parser.Literal{Type: parser.LiteralTypeNumber, Value: "1"},
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

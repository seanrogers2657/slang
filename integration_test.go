package main

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/backend/as"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
)

// TestEndToEndCompilation tests the entire compilation pipeline
// from source code to assembly output
func TestEndToEndCompilation(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedOutput []string
		expectError    bool
		errorStage     string
	}{
		{
			name:   "simple addition",
			source: "2 + 5",
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
			name:   "addition with larger numbers",
			source: "100 + 200",
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
			expectError: false,
		},
		{
			name:   "addition with extra whitespace",
			source: "  2   +   5  ",
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
			name:   "subtraction",
			source: "10 - 3",
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
			name:        "unsupported operator",
			source:      "5 ^ 2",
			expectError: true,
			errorStage:  "lexer",
		},
		{
			name:        "invalid character",
			source:      "2 @ 5",
			expectError: true,
			errorStage:  "lexer",
		},
		{
			name:        "single equals",
			source:      "5 = 5",
			expectError: true,
			errorStage:  "lexer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Lexer stage
			l := lexer.NewLexer([]byte(tt.source))
			l.Parse()

			if len(l.Errors) > 0 {
				if !tt.expectError || tt.errorStage != "lexer" {
					t.Fatalf("lexer error: %v", l.Errors)
				}
				return
			}

			// Parser stage
			p := parser.NewParser(l.Tokens)
			program := p.Parse()

			if len(p.Errors) > 0 {
				if !tt.expectError || tt.errorStage != "parser" {
					t.Fatalf("parser error: %v", p.Errors)
				}
				return
			}

			if program == nil || len(program.Statements) == 0 {
				t.Fatal("parser returned nil or empty program")
			}

			// Code generation stage
			generator := as.NewAsGenerator(program)
			output, err := generator.Generate()

			if err != nil {
				if !tt.expectError || tt.errorStage != "codegen" {
					t.Fatalf("codegen error: %v", err)
				}
				return
			}

			if tt.expectError {
				t.Fatal("expected error but compilation succeeded")
			}

			// Verify assembly output
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

// TestCompilationPipelineStages tests each stage independently
func TestCompilationPipelineStages(t *testing.T) {
	source := "2 + 5"

	t.Run("lexer produces correct tokens", func(t *testing.T) {
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		if len(l.Errors) > 0 {
			t.Fatalf("lexer errors: %v", l.Errors)
		}

		expectedTokens := 3 // number, operator, number
		if len(l.Tokens) != expectedTokens {
			t.Errorf("expected %d tokens, got %d", expectedTokens, len(l.Tokens))
		}
	})

	t.Run("parser produces correct AST", func(t *testing.T) {
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := parser.NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("parser errors: %v", p.Errors)
		}

		if program == nil || len(program.Statements) == 0 {
			t.Fatal("parser returned nil or empty program")
		}

		expr := program.Statements[0]

		if expr.Op != "+" {
			t.Errorf("expected operator '+', got %q", expr.Op)
		}
		if expr.Left.Value != "2" {
			t.Errorf("expected left value '2', got %q", expr.Left.Value)
		}
		if expr.Right.Value != "5" {
			t.Errorf("expected right value '5', got %q", expr.Right.Value)
		}
	})

	t.Run("code generator produces valid assembly", func(t *testing.T) {
		l := lexer.NewLexer([]byte(source))
		l.Parse()

		p := parser.NewParser(l.Tokens)
		program := p.Parse()

		generator := as.NewAsGenerator(program)
		output, err := generator.Generate()

		if err != nil {
			t.Fatalf("codegen error: %v", err)
		}

		if !strings.Contains(output, ".global _start") {
			t.Error("assembly should contain .global _start")
		}
		if !strings.Contains(output, "_start:") {
			t.Error("assembly should contain _start: label")
		}
		if !strings.Contains(output, "add x2, x0, x1") {
			t.Error("assembly should contain add instruction")
		}
	})
}

// TestExampleFile tests the example file in the repository
func TestExampleFile(t *testing.T) {
	source := "2 + 5"

	l := lexer.NewLexer([]byte(source))
	l.Parse()

	if len(l.Errors) > 0 {
		t.Fatalf("lexer errors: %v", l.Errors)
	}

	p := parser.NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		t.Fatalf("parser errors: %v", p.Errors)
	}

	generator := as.NewAsGenerator(program)
	output, err := generator.Generate()

	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// Verify the output matches expected ARM64 assembly structure
	expectedComponents := []string{
		".global _start",
		".align 4",
		"_start:",
		"mov x0, #2",
		"mov x1, #5",
		"add x2, x0, x1",
		"svc #0",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(output, component) {
			t.Errorf("expected assembly to contain %q", component)
		}
	}
}

// TestRegressions tests for specific bugs or edge cases
func TestRegressions(t *testing.T) {
	t.Run("newline at end of source", func(t *testing.T) {
		source := "2 + 5\n"

		l := lexer.NewLexer([]byte(source))
		l.Parse()

		if len(l.Errors) > 0 {
			t.Fatalf("lexer errors: %v", l.Errors)
		}

		p := parser.NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("parser errors: %v", p.Errors)
		}

		if program == nil || len(program.Statements) == 0 {
			t.Fatal("parser returned nil or empty program")
		}
	})

	t.Run("no whitespace", func(t *testing.T) {
		source := "2+5"

		l := lexer.NewLexer([]byte(source))
		l.Parse()

		if len(l.Errors) > 0 {
			t.Fatalf("lexer errors: %v", l.Errors)
		}

		p := parser.NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("parser errors: %v", p.Errors)
		}

		generator := as.NewAsGenerator(program)
		_, err := generator.Generate()

		if err != nil {
			t.Fatalf("codegen error: %v", err)
		}
	})

	t.Run("large numbers", func(t *testing.T) {
		source := "999999 + 888888"

		l := lexer.NewLexer([]byte(source))
		l.Parse()

		if len(l.Errors) > 0 {
			t.Fatalf("lexer errors: %v", l.Errors)
		}

		p := parser.NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			t.Fatalf("parser errors: %v", p.Errors)
		}

		if program == nil || len(program.Statements) == 0 {
			t.Fatal("parser returned nil or empty program")
		}

		expr := program.Statements[0]

		if expr.Left.Value != "999999" {
			t.Errorf("expected left value '999999', got %q", expr.Left.Value)
		}
		if expr.Right.Value != "888888" {
			t.Errorf("expected right value '888888', got %q", expr.Right.Value)
		}
	})
}

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/seanrogers2657/slang/backend/as"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
	"github.com/urfave/cli/v2"
)

type testResult struct {
	name    string
	passed  bool
	message string
}

func main() {
	app := &cli.App{
		Name:  "it",
		Usage: "Run integration tests for the Slang compiler",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show verbose output",
			},
		},
		Action: func(c *cli.Context) error {
			verbose := c.Bool("verbose")
			results := []testResult{}

			// Run all test suites
			results = append(results, runEndToEndTests(verbose)...)
			results = append(results, runPipelineStageTests(verbose)...)
			results = append(results, runExampleFileTest(verbose)...)
			results = append(results, runRegressionTests(verbose)...)

			// Print summary
			passed := 0
			failed := 0
			for _, result := range results {
				if result.passed {
					passed++
					if verbose {
						fmt.Printf("✓ %s\n", result.name)
					}
				} else {
					failed++
					fmt.Printf("✗ %s: %s\n", result.name, result.message)
				}
			}

			fmt.Printf("\nTotal: %d, Passed: %d, Failed: %d\n", len(results), passed, failed)

			if failed > 0 {
				os.Exit(1)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runEndToEndTests(verbose bool) []testResult {
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

	results := []testResult{}

	for _, tt := range tests {
		// Lexer stage
		l := lexer.NewLexer([]byte(tt.source))
		l.Parse()

		if len(l.Errors) > 0 {
			if !tt.expectError || tt.errorStage != "lexer" {
				results = append(results, testResult{
					name:    "end-to-end/" + tt.name,
					passed:  false,
					message: fmt.Sprintf("lexer error: %v", l.Errors),
				})
				continue
			}
			results = append(results, testResult{
				name:   "end-to-end/" + tt.name,
				passed: true,
			})
			continue
		}

		// Parser stage
		p := parser.NewParser(l.Tokens)
		program := p.Parse()

		if len(p.Errors) > 0 {
			if !tt.expectError || tt.errorStage != "parser" {
				results = append(results, testResult{
					name:    "end-to-end/" + tt.name,
					passed:  false,
					message: fmt.Sprintf("parser error: %v", p.Errors),
				})
				continue
			}
			results = append(results, testResult{
				name:   "end-to-end/" + tt.name,
				passed: true,
			})
			continue
		}

		if program == nil || len(program.Statements) == 0 {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: "parser returned nil or empty program",
			})
			continue
		}

		// Code generation stage
		generator := as.NewAsGenerator(program)
		output, err := generator.Generate()

		if err != nil {
			if !tt.expectError || tt.errorStage != "codegen" {
				results = append(results, testResult{
					name:    "end-to-end/" + tt.name,
					passed:  false,
					message: fmt.Sprintf("codegen error: %v", err),
				})
				continue
			}
			results = append(results, testResult{
				name:   "end-to-end/" + tt.name,
				passed: true,
			})
			continue
		}

		if tt.expectError {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: "expected error but compilation succeeded",
			})
			continue
		}

		// Verify assembly output
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != len(tt.expectedOutput) {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("expected %d lines, got %d", len(tt.expectedOutput), len(lines)),
			})
			continue
		}

		allMatch := true
		var mismatch string
		for i, line := range lines {
			if line != tt.expectedOutput[i] {
				allMatch = false
				mismatch = fmt.Sprintf("line %d: expected %q, got %q", i, tt.expectedOutput[i], line)
				break
			}
		}

		if !allMatch {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: mismatch,
			})
		} else {
			results = append(results, testResult{
				name:   "end-to-end/" + tt.name,
				passed: true,
			})
		}
	}

	return results
}

func runPipelineStageTests(verbose bool) []testResult {
	source := "2 + 5"
	results := []testResult{}

	// Test lexer stage
	l := lexer.NewLexer([]byte(source))
	l.Parse()

	if len(l.Errors) > 0 {
		results = append(results, testResult{
			name:    "pipeline/lexer",
			passed:  false,
			message: fmt.Sprintf("lexer errors: %v", l.Errors),
		})
	} else {
		expectedTokens := 3 // number, operator, number
		if len(l.Tokens) != expectedTokens {
			results = append(results, testResult{
				name:    "pipeline/lexer",
				passed:  false,
				message: fmt.Sprintf("expected %d tokens, got %d", expectedTokens, len(l.Tokens)),
			})
		} else {
			results = append(results, testResult{
				name:   "pipeline/lexer",
				passed: true,
			})
		}
	}

	// Test parser stage
	l2 := lexer.NewLexer([]byte(source))
	l2.Parse()

	p := parser.NewParser(l2.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		results = append(results, testResult{
			name:    "pipeline/parser",
			passed:  false,
			message: fmt.Sprintf("parser errors: %v", p.Errors),
		})
	} else if program == nil || len(program.Statements) == 0 {
		results = append(results, testResult{
			name:    "pipeline/parser",
			passed:  false,
			message: "parser returned nil or empty program",
		})
	} else {
		stmt := program.Statements[0]
		exprStmt, ok := stmt.(*parser.ExprStmt)
		if !ok {
			results = append(results, testResult{
				name:    "pipeline/parser",
				passed:  false,
				message: "expected ExprStmt, got different statement type",
			})
		} else {
			expr := exprStmt.Expr
			if expr.Op != "+" {
				results = append(results, testResult{
					name:    "pipeline/parser",
					passed:  false,
					message: fmt.Sprintf("expected operator '+', got %q", expr.Op),
				})
			} else if expr.Left.Value != "2" {
				results = append(results, testResult{
					name:    "pipeline/parser",
					passed:  false,
					message: fmt.Sprintf("expected left value '2', got %q", expr.Left.Value),
				})
			} else if expr.Right.Value != "5" {
				results = append(results, testResult{
					name:    "pipeline/parser",
					passed:  false,
					message: fmt.Sprintf("expected right value '5', got %q", expr.Right.Value),
				})
			} else {
				results = append(results, testResult{
					name:   "pipeline/parser",
					passed: true,
				})
			}
		}
	}

	// Test code generator stage
	l3 := lexer.NewLexer([]byte(source))
	l3.Parse()

	p3 := parser.NewParser(l3.Tokens)
	program3 := p3.Parse()

	generator := as.NewAsGenerator(program3)
	output, err := generator.Generate()

	if err != nil {
		results = append(results, testResult{
			name:    "pipeline/codegen",
			passed:  false,
			message: fmt.Sprintf("codegen error: %v", err),
		})
	} else {
		if !strings.Contains(output, ".global _start") {
			results = append(results, testResult{
				name:    "pipeline/codegen",
				passed:  false,
				message: "assembly should contain .global _start",
			})
		} else if !strings.Contains(output, "_start:") {
			results = append(results, testResult{
				name:    "pipeline/codegen",
				passed:  false,
				message: "assembly should contain _start: label",
			})
		} else if !strings.Contains(output, "add x2, x0, x1") {
			results = append(results, testResult{
				name:    "pipeline/codegen",
				passed:  false,
				message: "assembly should contain add instruction",
			})
		} else {
			results = append(results, testResult{
				name:   "pipeline/codegen",
				passed: true,
			})
		}
	}

	return results
}

func runExampleFileTest(verbose bool) []testResult {
	source := "2 + 5"
	results := []testResult{}

	l := lexer.NewLexer([]byte(source))
	l.Parse()

	if len(l.Errors) > 0 {
		return []testResult{{
			name:    "example-file",
			passed:  false,
			message: fmt.Sprintf("lexer errors: %v", l.Errors),
		}}
	}

	p := parser.NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		return []testResult{{
			name:    "example-file",
			passed:  false,
			message: fmt.Sprintf("parser errors: %v", p.Errors),
		}}
	}

	generator := as.NewAsGenerator(program)
	output, err := generator.Generate()

	if err != nil {
		return []testResult{{
			name:    "example-file",
			passed:  false,
			message: fmt.Sprintf("codegen error: %v", err),
		}}
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
			return []testResult{{
				name:    "example-file",
				passed:  false,
				message: fmt.Sprintf("expected assembly to contain %q", component),
			}}
		}
	}

	results = append(results, testResult{
		name:   "example-file",
		passed: true,
	})

	return results
}

func runRegressionTests(verbose bool) []testResult {
	results := []testResult{}

	// Test newline at end of source
	source1 := "2 + 5\n"
	l1 := lexer.NewLexer([]byte(source1))
	l1.Parse()

	if len(l1.Errors) > 0 {
		results = append(results, testResult{
			name:    "regression/newline-at-end",
			passed:  false,
			message: fmt.Sprintf("lexer errors: %v", l1.Errors),
		})
	} else {
		p1 := parser.NewParser(l1.Tokens)
		program1 := p1.Parse()

		if len(p1.Errors) > 0 {
			results = append(results, testResult{
				name:    "regression/newline-at-end",
				passed:  false,
				message: fmt.Sprintf("parser errors: %v", p1.Errors),
			})
		} else if program1 == nil || len(program1.Statements) == 0 {
			results = append(results, testResult{
				name:    "regression/newline-at-end",
				passed:  false,
				message: "parser returned nil or empty program",
			})
		} else {
			results = append(results, testResult{
				name:   "regression/newline-at-end",
				passed: true,
			})
		}
	}

	// Test no whitespace
	source2 := "2+5"
	l2 := lexer.NewLexer([]byte(source2))
	l2.Parse()

	if len(l2.Errors) > 0 {
		results = append(results, testResult{
			name:    "regression/no-whitespace",
			passed:  false,
			message: fmt.Sprintf("lexer errors: %v", l2.Errors),
		})
	} else {
		p2 := parser.NewParser(l2.Tokens)
		program2 := p2.Parse()

		if len(p2.Errors) > 0 {
			results = append(results, testResult{
				name:    "regression/no-whitespace",
				passed:  false,
				message: fmt.Sprintf("parser errors: %v", p2.Errors),
			})
		} else {
			generator2 := as.NewAsGenerator(program2)
			_, err := generator2.Generate()

			if err != nil {
				results = append(results, testResult{
					name:    "regression/no-whitespace",
					passed:  false,
					message: fmt.Sprintf("codegen error: %v", err),
				})
			} else {
				results = append(results, testResult{
					name:   "regression/no-whitespace",
					passed: true,
				})
			}
		}
	}

	// Test large numbers
	source3 := "999999 + 888888"
	l3 := lexer.NewLexer([]byte(source3))
	l3.Parse()

	if len(l3.Errors) > 0 {
		results = append(results, testResult{
			name:    "regression/large-numbers",
			passed:  false,
			message: fmt.Sprintf("lexer errors: %v", l3.Errors),
		})
	} else {
		p3 := parser.NewParser(l3.Tokens)
		program3 := p3.Parse()

		if len(p3.Errors) > 0 {
			results = append(results, testResult{
				name:    "regression/large-numbers",
				passed:  false,
				message: fmt.Sprintf("parser errors: %v", p3.Errors),
			})
		} else if program3 == nil || len(program3.Statements) == 0 {
			results = append(results, testResult{
				name:    "regression/large-numbers",
				passed:  false,
				message: "parser returned nil or empty program",
			})
		} else {
			stmt := program3.Statements[0]
			exprStmt, ok := stmt.(*parser.ExprStmt)
			if !ok {
				results = append(results, testResult{
					name:    "regression/large-numbers",
					passed:  false,
					message: "expected ExprStmt, got different statement type",
				})
			} else {
				expr := exprStmt.Expr
				if expr.Left.Value != "999999" {
					results = append(results, testResult{
						name:    "regression/large-numbers",
						passed:  false,
						message: fmt.Sprintf("expected left value '999999', got %q", expr.Left.Value),
					})
				} else if expr.Right.Value != "888888" {
					results = append(results, testResult{
						name:    "regression/large-numbers",
						passed:  false,
						message: fmt.Sprintf("expected right value '888888', got %q", expr.Right.Value),
					})
				} else {
					results = append(results, testResult{
						name:   "regression/large-numbers",
						passed: true,
					})
				}
			}
		}
	}

	return results
}

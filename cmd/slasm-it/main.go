package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/urfave/cli/v2"
)

type testResult struct {
	name    string
	passed  bool
	message string
}

func main() {
	app := &cli.App{
		Name:  "slasm-it",
		Usage: "Run integration tests for the native assembler",
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
			results = append(results, runLexerIntegrationTests(verbose)...)
			results = append(results, runParserIntegrationTests(verbose)...)
			results = append(results, runSymbolTableIntegrationTests(verbose)...)
			results = append(results, runLayoutIntegrationTests(verbose)...)
			results = append(results, runEndToEndTests(verbose)...)

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

func runLexerIntegrationTests(verbose bool) []testResult {
	tests := []struct {
		name     string
		input    string
		validate func([]slasm.Token) error
	}{
		{
			name:  "tokenize simple instruction",
			input: "mov x0, #1",
			validate: func(tokens []slasm.Token) error {
				if len(tokens) < 5 {
					return fmt.Errorf("expected at least 5 tokens, got %d", len(tokens))
				}
				if tokens[0].Value != "mov" {
					return fmt.Errorf("expected 'mov', got %q", tokens[0].Value)
				}
				return nil
			},
		},
		{
			name: "tokenize complete program",
			input: `.global _start
.align 4
_start:
    mov x0, #1
    svc #0`,
			validate: func(tokens []slasm.Token) error {
				// Just check we got tokens without error
				if len(tokens) == 0 {
					return fmt.Errorf("expected tokens but got none")
				}
				return nil
			},
		},
		{
			name:  "tokenize data section",
			input: `.data
buffer: .space 32
newline: .byte 10`,
			validate: func(tokens []slasm.Token) error {
				foundSpace := false
				foundByte := false
				for _, tok := range tokens {
					if tok.Value == ".space" {
						foundSpace = true
					}
					if tok.Value == ".byte" {
						foundByte = true
					}
				}
				if !foundSpace {
					return fmt.Errorf("expected to find .space directive")
				}
				if !foundByte {
					return fmt.Errorf("expected to find .byte directive")
				}
				return nil
			},
		},
		{
			name:  "tokenize with comments",
			input: "mov x0, #1  // Load 1 into x0",
			validate: func(tokens []slasm.Token) error {
				foundComment := false
				for _, tok := range tokens {
					if tok.Type == slasm.TokenComment {
						foundComment = true
					}
				}
				if !foundComment {
					return fmt.Errorf("expected to find comment token")
				}
				return nil
			},
		},
		{
			name:  "tokenize page offset relocations",
			input: "adrp x0, buffer@PAGE\nadd x0, x0, buffer@PAGEOFF",
			validate: func(tokens []slasm.Token) error {
				foundPage := false
				foundPageoff := false
				for _, tok := range tokens {
					if tok.Value == "PAGE" {
						foundPage = true
					}
					if tok.Value == "PAGEOFF" {
						foundPageoff = true
					}
				}
				if !foundPage {
					return fmt.Errorf("expected to find PAGE")
				}
				if !foundPageoff {
					return fmt.Errorf("expected to find PAGEOFF")
				}
				return nil
			},
		},
	}

	results := []testResult{}
	for _, tt := range tests {
		lexer := slasm.NewLexer(tt.input)
		tokens, err := lexer.Tokenize()

		if err != nil {
			results = append(results, testResult{
				name:    "lexer-integration/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("tokenization error: %v", err),
			})
			continue
		}

		if err := tt.validate(tokens); err != nil {
			results = append(results, testResult{
				name:    "lexer-integration/" + tt.name,
				passed:  false,
				message: err.Error(),
			})
		} else {
			results = append(results, testResult{
				name:   "lexer-integration/" + tt.name,
				passed: true,
			})
		}
	}

	return results
}

func runParserIntegrationTests(verbose bool) []testResult {
	tests := []struct {
		name     string
		input    string
		validate func(*slasm.Program) error
	}{
		{
			name:  "parse simple instruction",
			input: "mov x0, #1",
			validate: func(program *slasm.Program) error {
				if program == nil {
					return fmt.Errorf("program is nil")
				}
				if len(program.Sections) == 0 {
					return fmt.Errorf("expected sections")
				}
				return nil
			},
		},
		{
			name: "parse with labels",
			input: `_start:
    mov x0, #1
main:
    ret`,
			validate: func(program *slasm.Program) error {
				if program == nil {
					return fmt.Errorf("program is nil")
				}
				// Should have parsed labels and instructions
				return nil
			},
		},
		{
			name: "parse data section",
			input: `.data
buffer: .space 32`,
			validate: func(program *slasm.Program) error {
				if program == nil {
					return fmt.Errorf("program is nil")
				}
				if len(program.Sections) == 0 {
					return fmt.Errorf("expected at least one section")
				}
				return nil
			},
		},
	}

	results := []testResult{}
	for _, tt := range tests {
		lexer := slasm.NewLexer(tt.input)
		tokens, err := lexer.Tokenize()
		if err != nil {
			results = append(results, testResult{
				name:    "parser-integration/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("lexer error: %v", err),
			})
			continue
		}

		parser := slasm.NewParser(tokens)
		program, err := parser.Parse()

		if err != nil {
			results = append(results, testResult{
				name:    "parser-integration/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("parse error: %v", err),
			})
			continue
		}

		if err := tt.validate(program); err != nil {
			results = append(results, testResult{
				name:    "parser-integration/" + tt.name,
				passed:  false,
				message: err.Error(),
			})
		} else {
			results = append(results, testResult{
				name:   "parser-integration/" + tt.name,
				passed: true,
			})
		}
	}

	return results
}

func runSymbolTableIntegrationTests(verbose bool) []testResult {
	results := []testResult{}

	// Test creating and using a symbol table
	st := slasm.NewSymbolTable()

	// Define multiple symbols
	symbols := map[string]uint64{
		"_start": 0x0,
		"main":   0x100,
		"buffer": 0x200,
	}

	for name, addr := range symbols {
		if err := st.Define(name, addr, slasm.SectionText, 0, 0); err != nil {
			results = append(results, testResult{
				name:    "symbol-table-integration/define-symbols",
				passed:  false,
				message: fmt.Sprintf("failed to define %q: %v", name, err),
			})
			return results
		}
	}

	// Lookup all symbols
	for name, expectedAddr := range symbols {
		sym, exists := st.Lookup(name)
		if !exists {
			results = append(results, testResult{
				name:    "symbol-table-integration/lookup-symbols",
				passed:  false,
				message: fmt.Sprintf("symbol %q not found", name),
			})
			return results
		}
		if sym.Address != expectedAddr {
			results = append(results, testResult{
				name:    "symbol-table-integration/verify-addresses",
				passed:  false,
				message: fmt.Sprintf("symbol %q: expected 0x%x, got 0x%x", name, expectedAddr, sym.Address),
			})
			return results
		}
	}

	results = append(results, testResult{
		name:   "symbol-table-integration/complete-workflow",
		passed: true,
	})

	return results
}

func runLayoutIntegrationTests(verbose bool) []testResult {
	results := []testResult{}

	// Create a simple program for layout testing
	program := &slasm.Program{
		Sections: []*slasm.Section{
			{
				Type: slasm.SectionText,
				Items: []slasm.Item{
					&slasm.Label{Name: "_start"},
					&slasm.Instruction{
						Mnemonic: "mov",
						Operands: []*slasm.Operand{
							{Type: slasm.OperandRegister, Value: "x0"},
							{Type: slasm.OperandImmediate, Value: "1"},
						},
					},
					&slasm.Label{Name: "main"},
					&slasm.Instruction{
						Mnemonic: "ret",
					},
				},
			},
		},
	}

	layout := slasm.NewLayout(program)
	if err := layout.Calculate(); err != nil {
		results = append(results, testResult{
			name:    "layout-integration/calculate",
			passed:  false,
			message: fmt.Sprintf("layout calculation failed: %v", err),
		})
		return results
	}

	st := layout.GetSymbolTable()

	// Verify _start at address 0
	startSym, exists := st.Lookup("_start")
	if !exists {
		results = append(results, testResult{
			name:    "layout-integration/start-symbol",
			passed:  false,
			message: "_start symbol not found",
		})
		return results
	}
	if startSym.Address != 0 {
		results = append(results, testResult{
			name:    "layout-integration/start-address",
			passed:  false,
			message: fmt.Sprintf("expected _start at 0, got 0x%x", startSym.Address),
		})
		return results
	}

	// Verify main at address 4 (after mov instruction)
	mainSym, exists := st.Lookup("main")
	if !exists {
		results = append(results, testResult{
			name:    "layout-integration/main-symbol",
			passed:  false,
			message: "main symbol not found",
		})
		return results
	}
	if mainSym.Address != 4 {
		results = append(results, testResult{
			name:    "layout-integration/main-address",
			passed:  false,
			message: fmt.Sprintf("expected main at 4, got 0x%x", mainSym.Address),
		})
		return results
	}

	results = append(results, testResult{
		name:   "layout-integration/complete-workflow",
		passed: true,
	})

	return results
}

func runEndToEndTests(verbose bool) []testResult {
	tests := []struct {
		name     string
		assembly string
		validate func(*slasm.Program, *slasm.SymbolTable) error
	}{
		{
			name: "minimal program",
			assembly: `.global _start
.align 4
_start:
    mov x0, #1
    mov x16, #1
    svc #0`,
			validate: func(program *slasm.Program, st *slasm.SymbolTable) error {
				// Check that _start symbol exists
				sym, exists := st.Lookup("_start")
				if !exists {
					return fmt.Errorf("_start symbol not found")
				}
				if !sym.Global {
					return fmt.Errorf("_start should be marked as global")
				}
				return nil
			},
		},
		{
			name: "program with data section",
			assembly: `.data
.align 3
buffer: .space 32
newline: .byte 10

.text
.global _start
_start:
    mov x0, #0`,
			validate: func(program *slasm.Program, st *slasm.SymbolTable) error {
				// Check for buffer and newline symbols
				buffer, exists := st.Lookup("buffer")
				if !exists {
					return fmt.Errorf("buffer symbol not found")
				}
				if buffer.Section != slasm.SectionData {
					return fmt.Errorf("buffer should be in data section")
				}

				newline, exists := st.Lookup("newline")
				if !exists {
					return fmt.Errorf("newline symbol not found")
				}
				if newline.Address != 32 {
					return fmt.Errorf("newline should be at offset 32, got %d", newline.Address)
				}

				return nil
			},
		},
		{
			name: "program with multiple labels",
			assembly: `_start:
    b main

main:
    mov x0, #0
    bl helper
    ret

helper:
    ret`,
			validate: func(program *slasm.Program, st *slasm.SymbolTable) error {
				symbols := []string{"_start", "main", "helper"}
				for _, name := range symbols {
					if _, exists := st.Lookup(name); !exists {
						return fmt.Errorf("symbol %q not found", name)
					}
				}
				return nil
			},
		},
	}

	results := []testResult{}
	for _, tt := range tests {
		// Tokenize
		lexer := slasm.NewLexer(tt.assembly)
		tokens, err := lexer.Tokenize()
		if err != nil {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("lexer error: %v", err),
			})
			continue
		}

		// Parse
		parser := slasm.NewParser(tokens)
		program, err := parser.Parse()
		if err != nil {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("parser error: %v", err),
			})
			continue
		}

		// Layout
		layout := slasm.NewLayout(program)
		if err := layout.Calculate(); err != nil {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: fmt.Sprintf("layout error: %v", err),
			})
			continue
		}

		// Validate
		st := layout.GetSymbolTable()
		if err := tt.validate(program, st); err != nil {
			results = append(results, testResult{
				name:    "end-to-end/" + tt.name,
				passed:  false,
				message: err.Error(),
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

// Placeholder function to satisfy interface checks
func _ () {
	// This function helps ensure our test code compiles even when
	// the assembler implementation is incomplete
	_ = strings.Join([]string{}, "")
}

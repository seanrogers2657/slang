package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
	"github.com/seanrogers2657/slang/assembler"
	nativeasm "github.com/seanrogers2657/slang/assembler/asm"
	"github.com/seanrogers2657/slang/assembler/system"
	"github.com/seanrogers2657/slang/backend/as"
	"github.com/seanrogers2657/slang/frontend/errors"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
	"github.com/seanrogers2657/slang/frontend/semantic"
	"github.com/seanrogers2657/slang/internal/timing"
	"github.com/urfave/cli/v2"
)

// compileSource performs the full compilation pipeline with error checking
func compileSource(filename string, timer *timing.Timer) (string, error) {
	// Read source file
	if timer != nil {
		timer.Start("Read Source")
	}
	source, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	// Read source lines for error reporting
	sourceLines, err := errors.ReadSourceLines(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}
	if timer != nil {
		timer.End()
	}

	var allErrors []*errors.CompilerError

	// Lexical analysis
	if timer != nil {
		timer.Start("Lexer")
	}
	lex := lexer.NewLexer(source)
	lex.Parse()
	if timer != nil {
		timer.End()
	}

	// Convert lexer errors to CompilerErrors
	for _, lexErr := range lex.Errors {
		compilerErr := errors.NewError(lexErr.Error(), filename, lex.Tokens[0].Pos, "lexer")
		allErrors = append(allErrors, compilerErr)
	}

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Parsing
	if timer != nil {
		timer.Start("Parser")
	}
	pars := parser.NewParser(lex.Tokens)
	ast := pars.Parse()
	if timer != nil {
		timer.End()
	}

	// Convert parser errors to CompilerErrors
	for _, parseErr := range pars.Errors {
		compilerErr := errors.NewError(parseErr.Error(), filename, ast.StartPos, "parser")
		allErrors = append(allErrors, compilerErr)
	}

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Semantic analysis
	if timer != nil {
		timer.Start("Semantic Analysis")
	}
	analyzer := semantic.NewAnalyzer(filename)
	semanticErrors, _ := analyzer.Analyze(ast)
	allErrors = append(allErrors, semanticErrors...)
	if timer != nil {
		timer.End()
	}

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Code generation
	if timer != nil {
		timer.Start("Code Generation")
	}
	codeGenerator := as.NewAsGenerator(ast)
	assemblyOutput, err := codeGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("code generation failed: %w", err)
	}
	if timer != nil {
		timer.End()
	}

	return assemblyOutput, nil
}

// buildExecutable performs the full build pipeline: compile, assemble, and link
// If timer is provided, stages will be timed
// assemblerType specifies which assembler to use: "system" or "native"
func buildExecutable(filename string, assemblerType string, timer *timing.Timer) error {
	// Compile the source
	assemblyOutput, err := compileSource(filename, timer)
	if err != nil {
		return err
	}

	// Create assembler based on type
	var asm assembler.Assembler
	switch assemblerType {
	case "native":
		asm = nativeasm.New()
	case "system":
		asm = system.New()
	default:
		return fmt.Errorf("unknown assembler type: %s (use 'system' or 'native')", assemblerType)
	}

	// Build the executable
	if timer != nil {
		timer.Start("Assemble & Link")
	}

	opts := assembler.BuildOptions{
		AssemblyPath:      "build/output.s",
		ObjectPath:        "build/output.o",
		OutputPath:        "build/output",
		KeepIntermediates: true, // Keep .s and .o files for inspection
	}

	if err := asm.Build(assemblyOutput, opts); err != nil {
		return err
	}

	if timer != nil {
		timer.End()
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:  "sl",
		Usage: "Compile a Slang program",
		Commands: []*cli.Command{
			{
				Name:      "build",
				Usage:     "Build a Slang source file",
				ArgsUsage: "<source-file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "assembler",
						Aliases: []string{"a"},
						Value:   "system",
						Usage:   "Assembler to use: 'system' (default, uses macOS as/ld) or 'native' (custom implementation)",
					},
				},
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					assemblerType := c.String("assembler")
					timer := timing.NewTimer()

					// Build the executable with timing
					if err := buildExecutable(file, assemblerType, timer); err != nil {
						return err
					}

					spew.Dump("compilation done")
					fmt.Println(timer.Summary())
					return nil
				},
			},
			{
				Name:      "run",
				Usage:     "Compile and execute a Slang source file",
				ArgsUsage: "<source-file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "assembler",
						Aliases: []string{"a"},
						Value:   "system",
						Usage:   "Assembler to use: 'system' (default, uses macOS as/ld) or 'native' (custom implementation)",
					},
				},
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					assemblerType := c.String("assembler")
					timer := timing.NewTimer()

					// Build the executable with timing
					if err := buildExecutable(file, assemblerType, timer); err != nil {
						return err
					}

					// Show compilation summary before execution
					fmt.Println(timer.Summary())

					// Execute the compiled binary
					cmd := exec.Command("build/output")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							return fmt.Errorf("program exited with code %d", exitErr.ExitCode())
						}
						return fmt.Errorf("execution failed: %w", err)
					}

					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			return cli.ShowAppHelp(c)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

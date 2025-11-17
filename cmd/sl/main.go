package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
	"github.com/seanrogers2657/slang/backend/as"
	"github.com/seanrogers2657/slang/frontend/errors"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
	"github.com/seanrogers2657/slang/frontend/semantic"
	"github.com/urfave/cli/v2"
)

// compileSource performs the full compilation pipeline with error checking
func compileSource(filename string) (string, error) {
	// Read source file
	source, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	// Read source lines for error reporting
	sourceLines, err := errors.ReadSourceLines(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	var allErrors []*errors.CompilerError

	// Lexical analysis
	lex := lexer.NewLexer(source)
	lex.Parse()

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
	pars := parser.NewParser(lex.Tokens)
	ast := pars.Parse()

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
	analyzer := semantic.NewAnalyzer(filename)
	semanticErrors, _ := analyzer.Analyze(ast)
	allErrors = append(allErrors, semanticErrors...)

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Code generation
	codeGenerator := as.NewAsGenerator(ast)
	assemblyOutput, err := codeGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("code generation failed: %w", err)
	}

	return assemblyOutput, nil
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
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					// Compile the source
					assemblyOutput, err := compileSource(file)
					if err != nil {
						return err
					}

					// Write assembly output
					err = os.WriteFile("build/output.s", []byte(assemblyOutput), fs.ModePerm)
					if err != nil {
						return fmt.Errorf("failed to write assembly: %w", err)
					}

					// Assemble
					cmd := exec.Command("as", "-arch", "arm64", "build/output.s", "-o", "build/output.o")
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("assembly failed: %w", err)
					}

					// Link
					sdkPath := "/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX15.5.sdk"
					cmd = exec.Command(
						"ld",
						"-o",
						"build/output",
						"build/output.o",
						"-lSystem",
						"-syslibroot",
						sdkPath,
						"-e",
						"_start",
						"-arch",
						"arm64",
					)
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("linking failed: %w", err)
					}

					spew.Dump("compilation done")
					return nil
				},
			},
			{
				Name:      "run",
				Usage:     "Compile and execute a Slang source file",
				ArgsUsage: "<source-file>",
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					// Compile the source
					assemblyOutput, err := compileSource(file)
					if err != nil {
						return err
					}

					// Write assembly output
					err = os.WriteFile("build/output.s", []byte(assemblyOutput), fs.ModePerm)
					if err != nil {
						return fmt.Errorf("failed to write assembly: %w", err)
					}

					// Assemble
					cmd := exec.Command("as", "-arch", "arm64", "build/output.s", "-o", "build/output.o")
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("assembly failed: %w", err)
					}

					// Link
					sdkPath := "/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX15.5.sdk"
					cmd = exec.Command(
						"ld",
						"-o",
						"build/output",
						"build/output.o",
						"-lSystem",
						"-syslibroot",
						sdkPath,
						"-e",
						"_start",
						"-arch",
						"arm64",
					)
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("linking failed: %w", err)
					}

					// Execute the compiled binary
					cmd = exec.Command("build/output")
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

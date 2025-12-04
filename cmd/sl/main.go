package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/seanrogers2657/slang/assembler"
	nativeasm "github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/seanrogers2657/slang/assembler/system"
	"github.com/seanrogers2657/slang/backend/codegen"
	"github.com/seanrogers2657/slang/errors"
	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
	"github.com/seanrogers2657/slang/frontend/semantic"
	"github.com/seanrogers2657/slang/internal/timing"
	"github.com/urfave/cli/v2"
)

// Global error handler for sl
var errorHandler = errors.NewHandler(errors.ToolSL)

const sectionWidth = 66

// printSection prints a section header with box drawing characters
func printSection(title string) {
	// Top border
	fmt.Println("╔" + strings.Repeat("═", sectionWidth) + "╗")

	// Title line - center the title
	padding := (sectionWidth - len(title)) / 2
	leftPad := strings.Repeat(" ", padding)
	rightPad := strings.Repeat(" ", sectionWidth-padding-len(title))
	fmt.Println("║" + leftPad + title + rightPad + "║")

	// Bottom border
	fmt.Println("╚" + strings.Repeat("═", sectionWidth) + "╝")
	fmt.Println()
}

// printDivider prints a horizontal divider line
func printDivider() {
	fmt.Println(strings.Repeat("─", sectionWidth+2))
}

// formatTokens returns a formatted table of tokens
func formatTokens(tokens []lexer.Token) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Tokens (%d):\n", len(tokens)))

	// Header
	sb.WriteString(fmt.Sprintf("  %-4s %-14s %-20s %s\n", "#", "Type", "Value", "Position"))
	sb.WriteString("  " + strings.Repeat("-", 60) + "\n")

	// Token rows
	for i, tok := range tokens {
		// Escape special characters for display
		value := tok.Value
		if tok.Type == lexer.TokenTypeNewline {
			value = "\\n"
		} else if tok.Type == lexer.TokenTypeString {
			value = fmt.Sprintf("%q", value)
		}

		// Truncate long values
		if len(value) > 18 {
			value = value[:15] + "..."
		}

		sb.WriteString(fmt.Sprintf("  %-4d %-14s %-20s %d:%d\n",
			i+1,
			tok.Type.String(),
			value,
			tok.Pos.Line,
			tok.Pos.Column,
		))
	}

	return sb.String()
}

// formatAST returns a formatted tree representation of the AST
func formatAST(program *ast.Program) string {
	var sb strings.Builder
	sb.WriteString("AST:\n")
	sb.WriteString("Program\n")

	// Handle function-based programs
	if len(program.Declarations) > 0 {
		for i, decl := range program.Declarations {
			isLast := i == len(program.Declarations)-1
			formatDeclaration(&sb, decl, "", isLast)
		}
	}

	// Handle legacy statement-based programs
	if len(program.Statements) > 0 {
		for i, stmt := range program.Statements {
			isLast := i == len(program.Statements)-1
			formatStatement(&sb, stmt, "", isLast)
		}
	}

	return sb.String()
}

func formatDeclaration(sb *strings.Builder, decl ast.Declaration, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	switch d := decl.(type) {
	case *ast.FunctionDecl:
		sb.WriteString(prefix + connector + fmt.Sprintf("FunctionDecl: %s\n", d.Name))
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
		formatStatement(sb, d.Body, childPrefix, true)
	}
}

func formatStatement(sb *strings.Builder, stmt ast.Statement, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		sb.WriteString(prefix + connector + "BlockStmt\n")
		for i, innerStmt := range s.Statements {
			formatStatement(sb, innerStmt, childPrefix, i == len(s.Statements)-1)
		}
	case *ast.ExprStmt:
		sb.WriteString(prefix + connector + "ExprStmt\n")
		formatExpression(sb, s.Expr, childPrefix, true)
	}
}

func formatExpression(sb *strings.Builder, expr ast.Expression, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		sb.WriteString(prefix + connector + fmt.Sprintf("BinaryExpr: %s\n", e.Op))
		formatExpression(sb, e.Left, childPrefix, false)
		formatExpression(sb, e.Right, childPrefix, true)
	case *ast.LiteralExpr:
		kind := "number"
		if e.Kind == ast.LiteralTypeString {
			kind = "string"
		} else if e.Kind == ast.LiteralTypeBoolean {
			kind = "boolean"
		}
		sb.WriteString(prefix + connector + fmt.Sprintf("LiteralExpr: %s (%s)\n", e.Value, kind))
	}
}

// formatTypedAST returns a formatted tree representation of the typed AST
func formatTypedAST(program *semantic.TypedProgram) string {
	var sb strings.Builder
	sb.WriteString("Typed AST:\n")
	sb.WriteString("Program\n")

	// Handle function-based programs
	if len(program.Declarations) > 0 {
		for i, decl := range program.Declarations {
			isLast := i == len(program.Declarations)-1
			formatTypedDeclaration(&sb, decl, "", isLast)
		}
	}

	// Handle legacy statement-based programs
	if len(program.Statements) > 0 {
		for i, stmt := range program.Statements {
			isLast := i == len(program.Statements)-1
			formatTypedStatement(&sb, stmt, "", isLast)
		}
	}

	return sb.String()
}

func formatTypedDeclaration(sb *strings.Builder, decl semantic.TypedDeclaration, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	switch d := decl.(type) {
	case *semantic.TypedFunctionDecl:
		sb.WriteString(prefix + connector + fmt.Sprintf("FunctionDecl: %s\n", d.Name))
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
		formatTypedStatement(sb, d.Body, childPrefix, true)
	}
}

func formatTypedStatement(sb *strings.Builder, stmt semantic.TypedStatement, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	switch s := stmt.(type) {
	case *semantic.TypedBlockStmt:
		sb.WriteString(prefix + connector + "BlockStmt\n")
		for i, innerStmt := range s.Statements {
			formatTypedStatement(sb, innerStmt, childPrefix, i == len(s.Statements)-1)
		}
	case *semantic.TypedExprStmt:
		sb.WriteString(prefix + connector + "ExprStmt\n")
		formatTypedExpression(sb, s.Expr, childPrefix, true)
	}
}

func formatTypedExpression(sb *strings.Builder, expr semantic.TypedExpression, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	switch e := expr.(type) {
	case *semantic.TypedBinaryExpr:
		sb.WriteString(prefix + connector + fmt.Sprintf("BinaryExpr: %s -> %s\n", e.Op, e.Type.String()))
		formatTypedExpression(sb, e.Left, childPrefix, false)
		formatTypedExpression(sb, e.Right, childPrefix, true)
	case *semantic.TypedLiteralExpr:
		sb.WriteString(prefix + connector + fmt.Sprintf("LiteralExpr: %s -> %s\n", e.Value, e.Type.String()))
	}
}

// toErrorPos converts an ast.Position to an errors.Position
func toErrorPos(line, column, offset int) errors.Position {
	return errors.Position{Line: line, Column: column, Offset: offset}
}

// compileSource performs the full compilation pipeline with error checking
// If verbose is true, debug output is printed for each stage
func compileSource(filename string, verbose bool, timer *timing.Timer) (string, error) {
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

	// Verbose: print source input
	if verbose {
		printSection("SOURCE INPUT")
		fmt.Printf("File: %s\n", filename)
		printDivider()
		fmt.Println(string(source))
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
		pos := errors.Position{Line: 1, Column: 1}
		if len(lex.Tokens) > 0 {
			pos = toErrorPos(lex.Tokens[0].Pos.Line, lex.Tokens[0].Pos.Column, lex.Tokens[0].Pos.Offset)
		}
		compilerErr := errors.NewError(lexErr.Error(), filename, pos, "lexer").WithTool(errors.ToolSL)
		allErrors = append(allErrors, compilerErr)
	}

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Verbose: print lexer output
	if verbose {
		printSection("LEXER OUTPUT")
		fmt.Println(formatTokens(lex.Tokens))
	}

	// Parsing
	if timer != nil {
		timer.Start("Parser")
	}
	pars := parser.NewParser(lex.Tokens)
	parsedAST := pars.Parse()
	if timer != nil {
		timer.End()
	}

	// Convert parser errors to CompilerErrors
	for _, parseErr := range pars.Errors {
		pos := toErrorPos(parsedAST.StartPos.Line, parsedAST.StartPos.Column, parsedAST.StartPos.Offset)
		compilerErr := errors.NewError(parseErr.Error(), filename, pos, "parser").WithTool(errors.ToolSL)
		allErrors = append(allErrors, compilerErr)
	}

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Verbose: print parser output
	if verbose {
		printSection("PARSER OUTPUT")
		fmt.Println(formatAST(parsedAST))
	}

	// Semantic analysis
	if timer != nil {
		timer.Start("Semantic Analysis")
	}
	analyzer := semantic.NewAnalyzer(filename)
	semanticErrors, typedAST := analyzer.Analyze(parsedAST)
	allErrors = append(allErrors, semanticErrors...)
	if timer != nil {
		timer.End()
	}

	if len(allErrors) > 0 {
		fmt.Println(errors.FormatErrors(allErrors, sourceLines))
		return "", fmt.Errorf("compilation failed")
	}

	// Verbose: print semantic analysis output
	if verbose {
		printSection("SEMANTIC ANALYSIS OUTPUT")
		fmt.Println(formatTypedAST(typedAST))
	}

	// Code generation
	if timer != nil {
		timer.Start("Code Generation")
	}
	// Use typed code generator for type-aware code generation
	typedCodeGenerator := codegen.NewTypedCodeGenerator(typedAST, sourceLines)
	assemblyOutput, err := typedCodeGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("code generation failed: %w", err)
	}
	if timer != nil {
		timer.End()
	}

	// Verbose: print code generation output
	if verbose {
		printSection("CODE GENERATION OUTPUT")
		fmt.Printf("Generated Assembly (%d bytes):\n", len(assemblyOutput))
		printDivider()
		fmt.Println(assemblyOutput)
	}

	return assemblyOutput, nil
}

// buildExecutable performs the full build pipeline: compile, assemble, and link
// If timer is provided, stages will be timed
// assemblerType specifies which assembler to use: "system" or "native"
// verbose enables debug output for all compilation stages and the native assembler
func buildExecutable(filename string, assemblerType string, verbose bool, timer *timing.Timer) error {
	// Compile the source
	assemblyOutput, err := compileSource(filename, verbose, timer)
	if err != nil {
		return err
	}

	// Create assembler based on type
	var asm assembler.Assembler
	switch assemblerType {
	case "native":
		nasm := nativeasm.New()
		nasm.Logger.SetEnabled(verbose)
		asm = nasm
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
		return fmt.Errorf("[assemble] %w", err)
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
						Value:   "native",
						Usage:   "Assembler to use: 'native' (default, uses slasm) or 'system' (uses macOS as/ld)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose debug output for all compilation stages",
					},
				},
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					assemblerType := c.String("assembler")
					verbose := c.Bool("verbose")
					timer := timing.NewTimer()

					// Build the executable with timing
					if err := buildExecutable(file, assemblerType, verbose, timer); err != nil {
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
						Value:   "native",
						Usage:   "Assembler to use: 'native' (default, uses slasm) or 'system' (uses macOS as/ld)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose debug output for all compilation stages",
					},
				},
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					assemblerType := c.String("assembler")
					verbose := c.Bool("verbose")
					timer := timing.NewTimer()

					// Build the executable with timing
					if err := buildExecutable(file, assemblerType, verbose, timer); err != nil {
						return err
					}

					// Show compilation summary before execution
					fmt.Println(timer.Summary())

					// Execute the compiled binary
					cmd := exec.Command("build/output")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						if verbose {
							if exitErr, ok := err.(*exec.ExitError); ok {
								return fmt.Errorf("program exited with code %d", exitErr.ExitCode())
							}
							return fmt.Errorf("execution failed: %w", err)
						}
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
		// Wrap the error with sl tool identification and display
		compilerErr := errorHandler.Wrap(err, "")
		errorHandler.Handle([]*errors.CompilerError{compilerErr})
		os.Exit(1)
	}
}

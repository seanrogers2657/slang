package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/seanrogers2657/slang/assembler"
	nativeasm "github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/seanrogers2657/slang/assembler/system"
	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/ir"
	"github.com/seanrogers2657/slang/compiler/ir/backend"
	"github.com/seanrogers2657/slang/compiler/ir/backend/arm64"
	"github.com/seanrogers2657/slang/compiler/lexer"
	"github.com/seanrogers2657/slang/compiler/parser"
	"github.com/seanrogers2657/slang/compiler/semantic"
	"github.com/seanrogers2657/slang/errors"
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

// printIRStats prints statistics about the IR program
func printIRStats(prog *ir.Program) {
	fmt.Println("IR Statistics:")

	// Count totals
	totalBlocks := 0
	totalValues := 0
	opCounts := make(map[ir.Op]int)

	for _, fn := range prog.Functions {
		totalBlocks += len(fn.Blocks)
		for _, block := range fn.Blocks {
			totalValues += len(block.Values)
			for _, val := range block.Values {
				opCounts[val.Op]++
			}
		}
	}

	fmt.Printf("  • Functions: %d\n", len(prog.Functions))
	fmt.Printf("  • Structs: %d\n", len(prog.Structs))
	fmt.Printf("  • Globals: %d\n", len(prog.Globals))
	fmt.Printf("  • Strings: %d\n", len(prog.Strings))
	fmt.Printf("  • Blocks: %d\n", totalBlocks)
	fmt.Printf("  • Values: %d\n", totalValues)

	// Show operation breakdown if there are values
	if len(opCounts) > 0 {
		// Sort operations by count (descending) for consistent output
		type opCount struct {
			op    ir.Op
			count int
		}
		var sorted []opCount
		for op, count := range opCounts {
			sorted = append(sorted, opCount{op, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].count != sorted[j].count {
				return sorted[i].count > sorted[j].count
			}
			return sorted[i].op.String() < sorted[j].op.String()
		})

		fmt.Println("\n  Operation counts:")
		for _, oc := range sorted {
			fmt.Printf("    %-12s %d\n", oc.op.String()+":", oc.count)
		}
	}
	fmt.Println()
}

// compileSourceWithIR performs compilation using the IR pipeline
// This is the new compilation path that uses SSA-form IR and the ARM64 backend
func compileSourceWithIR(filename string, verbose bool, timer *timing.Timer) (string, error) {
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
	lex := lexer.NewLexerWithFilename(source, filename)
	lex.Parse()
	if timer != nil {
		timer.End()
	}

	allErrors = append(allErrors, lex.Errors...)
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

	allErrors = append(allErrors, pars.Errors...)
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

	// IR Generation
	if timer != nil {
		timer.Start("IR Generation")
	}
	irProg, err := ir.Generate(typedAST)
	if err != nil {
		return "", fmt.Errorf("IR generation failed: %w", err)
	}
	if timer != nil {
		timer.End()
	}

	// Verbose: print IR
	if verbose {
		printSection("IR OUTPUT")
		fmt.Println(ir.String(irProg))

		// Print IR statistics
		printIRStats(irProg)
	}

	// Validate IR
	irErrors := ir.Validate(irProg)

	// Verbose: print validation results
	if verbose {
		printSection("IR VALIDATION")
		if len(irErrors) == 0 {
			fmt.Println("✓ IR validation passed")
			fmt.Printf("  • %d function(s) validated\n", len(irProg.Functions))
		} else {
			fmt.Printf("✗ IR validation failed with %d error(s):\n", len(irErrors))
			for _, e := range irErrors {
				fmt.Printf("  • %s\n", e.Error())
			}
		}
		fmt.Println()
	}

	if len(irErrors) > 0 {
		if !verbose {
			// If not verbose, still print the errors
			for _, e := range irErrors {
				fmt.Println(e.Error())
			}
		}
		return "", fmt.Errorf("IR validation failed")
	}

	// ARM64 Code Generation
	if timer != nil {
		timer.Start("ARM64 Backend")
	}
	config := &backend.Config{
		Filename:    filename,
		SourceLines: sourceLines,
	}
	arm64Backend := arm64.New(config)
	assemblyOutput, err := arm64Backend.Generate(irProg)
	if err != nil {
		return "", fmt.Errorf("ARM64 code generation failed: %w", err)
	}
	if timer != nil {
		timer.End()
	}

	// Verbose: print assembly output
	if verbose {
		printSection("ARM64 ASSEMBLY OUTPUT")
		fmt.Printf("Generated Assembly (%d bytes):\n", len(assemblyOutput))
		printDivider()
		fmt.Println(assemblyOutput)
	}

	return assemblyOutput, nil
}

// buildExecutable performs the full build pipeline: compile, assemble, and link
// If timer is provided, stages will be timed
// buildExecutable compiles a source file to an executable.
// assemblerType specifies which assembler to use: "system" or "native"
// verbose enables debug output for all compilation stages and the native assembler
// Returns the assembly timing summary (empty string for system assembler)
func buildExecutable(filename string, assemblerType string, verbose bool, timer *timing.Timer) (string, error) {
	// Compile the source using IR-based pipeline
	assemblyOutput, err := compileSourceWithIR(filename, verbose, timer)
	if err != nil {
		return "", err
	}

	// Create assembler based on type
	var nasm *nativeasm.NativeAssembler
	var asm assembler.Assembler
	switch assemblerType {
	case "native":
		nasm = nativeasm.New()
		nasm.Logger.SetEnabled(verbose)
		asm = nasm
	case "system":
		asm = system.New()
	default:
		return "", fmt.Errorf("unknown assembler type: %s (use 'system' or 'native')", assemblerType)
	}

	opts := assembler.BuildOptions{
		AssemblyPath:      "build/output.s",
		ObjectPath:        "build/output.o",
		OutputPath:        "build/output",
		KeepIntermediates: true, // Keep .s and .o files for inspection
	}

	if err := asm.Build(assemblyOutput, opts); err != nil {
		return "", fmt.Errorf("[assemble] %w", err)
	}

	// Return assembly timing summary if using native assembler
	var asmSummary string
	if nasm != nil {
		asmSummary = nasm.TimingSummary()
	}

	return asmSummary, nil
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

					// Build the executable
					asmSummary, err := buildExecutable(file, assemblerType, verbose, timer)
					if err != nil {
						return err
					}

					fmt.Println("Compilation successful")
					fmt.Println(timer.Summary())
					if asmSummary != "" {
						fmt.Print(asmSummary)
					}
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

					// Build the executable
					asmSummary, err := buildExecutable(file, assemblerType, verbose, timer)
					if err != nil {
						return err
					}

					// Show compilation and assembly summaries before execution
					fmt.Println(timer.Summary())
					if asmSummary != "" {
						fmt.Print(asmSummary)
					}

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

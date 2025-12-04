// Package sl_test contains end-to-end tests for the Slang compiler.
package sl_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/seanrogers2657/slang/backend/as"
	slangErrors "github.com/seanrogers2657/slang/errors"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
	"github.com/seanrogers2657/slang/frontend/semantic"
	"github.com/seanrogers2657/slang/test/testutil"
)

// getExamplesDir returns the path to the _examples/slang directory.
func getExamplesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current file path")
	}
	// Go up from test/sl/ to repo root, then into _examples/slang
	return filepath.Join(filepath.Dir(filename), "..", "..", "_examples", "slang")
}

func TestE2E(t *testing.T) {
	examplesDir := getExamplesDir()

	testCases, err := testutil.LoadTestCases(examplesDir, "*.sl")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}

	if len(testCases) == 0 {
		t.Fatalf("no test cases found in %s", examplesDir)
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			if tc.Skip != "" {
				t.Skipf("skipping: %s", tc.Skip)
			}

			runSlangTest(t, tc)
		})
	}
}

func runSlangTest(t *testing.T, tc *testutil.TestExpectation) {
	t.Helper()

	// Read the source file
	source, err := os.ReadFile(tc.FilePath)
	if err != nil {
		t.Fatalf("failed to read source file: %v", err)
	}

	// Lexer stage
	l := lexer.NewLexer(source)
	l.Parse()

	if len(l.Errors) > 0 {
		if tc.ExpectError && tc.ErrorStage == "lexer" {
			checkErrorContains(t, l.Errors, tc.ErrorContains)
			return
		}
		t.Fatalf("lexer errors: %v", l.Errors)
	}

	// Parser stage
	p := parser.NewParser(l.Tokens)
	program := p.Parse()

	if len(p.Errors) > 0 {
		if tc.ExpectError && tc.ErrorStage == "parser" {
			checkErrorContains(t, p.Errors, tc.ErrorContains)
			return
		}
		t.Fatalf("parser errors: %v", p.Errors)
	}

	if program == nil || (len(program.Statements) == 0 && len(program.Declarations) == 0) {
		t.Fatalf("parser returned nil or empty program")
	}

	// Semantic analysis stage
	analyzer := semantic.NewAnalyzer("<test>")
	semanticErrors, _ := analyzer.Analyze(program)

	if len(semanticErrors) > 0 {
		if tc.ExpectError && tc.ErrorStage == "semantic" {
			checkSemanticErrorContains(t, semanticErrors, tc.ErrorContains)
			return
		}
		t.Fatalf("semantic errors: %v", semanticErrors)
	}

	// If we expected an error but got none
	if tc.ExpectError {
		t.Fatalf("expected %s error but compilation succeeded", tc.ErrorStage)
	}

	// Code generation stage - uses raw AST, not typed AST
	sourceLines := strings.Split(string(source), "\n")
	generator := as.NewAsGenerator(program, sourceLines)
	asmOutput, err := generator.Generate()

	if err != nil {
		if tc.ExpectError && tc.ErrorStage == "codegen" {
			if tc.ErrorContains != "" && !strings.Contains(err.Error(), tc.ErrorContains) {
				t.Errorf("error should contain %q, got: %v", tc.ErrorContains, err)
			}
			return
		}
		t.Fatalf("codegen error: %v", err)
	}

	// If stdout expectations exist, build and run with slasm
	if tc.Stdout != "" || tc.ExitCode != 0 {
		runWithSlasm(t, tc, asmOutput)
	}
}

func runWithSlasm(t *testing.T, tc *testutil.TestExpectation, asmOutput string) {
	t.Helper()

	// Create assembler and build
	asm := slasm.New()
	// Replace slashes with underscores to avoid creating subdirectories
	safeName := strings.ReplaceAll(tc.Name, "/", "_")
	outputPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_%s", safeName))

	err := asm.Build(asmOutput, assembler.BuildOptions{
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("slasm build failed: %v", err)
	}

	// Execute the built program
	cmd := exec.Command(outputPath)
	output, err := cmd.CombinedOutput()

	actualExit := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		actualExit = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to execute program: %v", err)
	}

	// Check exit code
	if actualExit != tc.ExitCode {
		t.Errorf("exit code: got %d, want %d", actualExit, tc.ExitCode)
	}

	// Check stdout if specified
	if tc.Stdout != "" {
		if string(output) != tc.Stdout {
			t.Errorf("stdout:\ngot:  %q\nwant: %q", string(output), tc.Stdout)
		}
	}
}

func checkErrorContains(t *testing.T, errors []error, contains string) {
	t.Helper()
	if contains == "" {
		return
	}
	for _, err := range errors {
		if strings.Contains(err.Error(), contains) {
			return
		}
	}
	t.Errorf("no error contains %q, got: %v", contains, errors)
}

func checkSemanticErrorContains(t *testing.T, errs []*slangErrors.CompilerError, contains string) {
	t.Helper()
	if contains == "" {
		return
	}
	for _, err := range errs {
		if strings.Contains(err.Message, contains) {
			return
		}
	}
	t.Errorf("no error contains %q, got: %v", contains, errs)
}

// Package sl_test contains end-to-end tests for the Slang compiler.
package sl_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/seanrogers2657/slang/compiler/ir/backend"
	"github.com/seanrogers2657/slang/compiler/ir/backend/arm64"
	"github.com/seanrogers2657/slang/compiler/slpackage"
	slangErrors "github.com/seanrogers2657/slang/errors"
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

func TestE2EProjects(t *testing.T) {
	examplesDir := getExamplesDir()
	projectsDir := filepath.Join(filepath.Dir(examplesDir), "projects")

	// Skip if no projects directory exists yet
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		t.Skip("no projects directory found")
	}

	testCases, err := testutil.LoadProjectTestCases(projectsDir)
	if err != nil {
		t.Fatalf("failed to load project test cases: %v", err)
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			if tc.Skip != "" {
				t.Skipf("skipping: %s", tc.Skip)
			}

			runProjectTest(t, tc)
		})
	}
}

func runProjectTest(t *testing.T, tc *testutil.TestExpectation) {
	t.Helper()
	rootDir := filepath.Dir(tc.FilePath)
	rootFiles, err := slpackage.DiscoverSlFiles(rootDir)
	if err != nil {
		t.Fatalf("failed to discover root files: %v", err)
	}
	if len(rootFiles) == 0 {
		t.Fatal("no .sl files found in project root")
	}
	compileAndRun(t, tc, rootDir, rootFiles)
}

func runSlangTest(t *testing.T, tc *testutil.TestExpectation) {
	t.Helper()
	rootDir := filepath.Dir(tc.FilePath)
	compileAndRun(t, tc, rootDir, []string{tc.FilePath})
}

// compileAndRun runs the full compilation pipeline and checks expectations.
func compileAndRun(t *testing.T, tc *testutil.TestExpectation, rootDir string, rootFiles []string) {
	t.Helper()

	compiler := slpackage.NewCompiler(rootDir, tc.FilePath, rootFiles)

	// Phase 1: Discovery & Parsing
	pkgFiles, phase1Errs := compiler.DiscoverAndParse()
	if len(phase1Errs) > 0 {
		if tc.ExpectError && (tc.ErrorStage == "lexer" || tc.ErrorStage == "parser" || tc.ErrorStage == "module") {
			checkCompilerErrorContains(t, phase1Errs, tc.ErrorContains)
			return
		}
		t.Fatalf("phase 1 errors: %v", phase1Errs)
	}

	// Phase 2: Semantic Analysis
	phase2Errs, typedPrograms := compiler.Analyze(pkgFiles)
	if len(phase2Errs) > 0 {
		if tc.ExpectError && (tc.ErrorStage == "semantic" || tc.ErrorStage == "module") {
			checkCompilerErrorContains(t, phase2Errs, tc.ErrorContains)
			return
		}
		t.Fatalf("semantic errors: %v", phase2Errs)
	}

	// If we expected an error but got none before code generation
	if tc.ExpectError && tc.ErrorStage != "codegen" {
		t.Fatalf("expected %s error but compilation succeeded", tc.ErrorStage)
	}

	// Phase 3: IR Generation
	irProg, irErr := compiler.GenerateIR(typedPrograms)
	if irErr != nil {
		if tc.ExpectError && tc.ErrorStage == "codegen" {
			if tc.ErrorContains != "" && !strings.Contains(irErr.Error(), tc.ErrorContains) {
				t.Errorf("error should contain %q, got: %v", tc.ErrorContains, irErr)
			}
			return
		}
		t.Fatalf("IR generation error: %v", irErr)
	}

	// Phase 4: ARM64 Backend
	arm64Backend := arm64.New(backend.DefaultConfig())
	asmOutput, err := arm64Backend.Generate(irProg)
	if err != nil {
		if tc.ExpectError && tc.ErrorStage == "codegen" {
			if tc.ErrorContains != "" && !strings.Contains(err.Error(), tc.ErrorContains) {
				t.Errorf("error should contain %q, got: %v", tc.ErrorContains, err)
			}
			return
		}
		t.Fatalf("ARM64 backend error: %v", err)
	}

	if tc.ExpectError {
		t.Fatalf("expected %s error but compilation succeeded", tc.ErrorStage)
	}

	// If stdout expectations exist, build and run
	if tc.Stdout != "" || tc.ExitCode != 0 || tc.StderrContains != "" {
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

	runAndCheck(t, tc, outputPath)
}

func runAndCheck(t *testing.T, tc *testutil.TestExpectation, outputPath string) {
	t.Helper()

	// Execute the built program with a 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, outputPath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("test timed out after 10 seconds (possible infinite loop)")
	}

	actualExit := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		actualExit = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to execute program: %v", err)
	}

	// Check exit code
	if actualExit != tc.ExitCode {
		t.Errorf("exit code: got %d, want %d\nstdout: %s\nstderr: %s", actualExit, tc.ExitCode, stdout.String(), stderr.String())
	}

	// Check stdout if specified
	if tc.Stdout != "" {
		if stdout.String() != tc.Stdout {
			t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), tc.Stdout)
		}
	}

	// Check stderr if specified
	if tc.Stderr != "" {
		if stderr.String() != tc.Stderr {
			t.Errorf("stderr:\ngot:  %q\nwant: %q", stderr.String(), tc.Stderr)
		}
	}

	// Check stderr_contains if specified
	if tc.StderrContains != "" {
		if !strings.Contains(stderr.String(), tc.StderrContains) {
			t.Errorf("stderr should contain %q, got: %q", tc.StderrContains, stderr.String())
		}
	}
}

// checkCompilerErrorContains checks if any CompilerError contains the expected string.
// Used for lexer and semantic errors which both use CompilerError.
func checkCompilerErrorContains(t *testing.T, errs []*slangErrors.CompilerError, contains string) {
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

// getProgramsDir returns the path to the _programs directory.
func getProgramsDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current file path")
	}
	// Go up from test/sl/ to repo root, then into _programs
	return filepath.Join(filepath.Dir(filename), "..", "..", "_programs")
}

// TestProgramsCompile tests that all programs in _programs/ compile successfully.
// Finds any .sl file with a main function, recursively.
func TestProgramsCompile(t *testing.T) {
	programsDir := getProgramsDir()

	testCases, err := testutil.LoadProgramTestCases(programsDir)
	if err != nil {
		t.Fatalf("failed to load program test cases: %v", err)
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			runProjectTest(t, tc)
		})
	}
}

// Package slasm_test contains end-to-end tests for the slasm assembler.
package slasm_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/seanrogers2657/slang/test/testutil"
)

// getExamplesDir returns the path to the _examples/arm64 directory.
func getExamplesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current file path")
	}
	// Go up from test/slasm/ to repo root, then into _examples/arm64
	return filepath.Join(filepath.Dir(filename), "..", "..", "_examples", "arm64")
}

func TestE2E(t *testing.T) {
	examplesDir := getExamplesDir()

	testCases, err := testutil.LoadTestCases(examplesDir, "*.s")
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

			runAssemblyTest(t, tc)
		})
	}
}

func runAssemblyTest(t *testing.T, tc *testutil.TestExpectation) {
	t.Helper()

	// Read the assembly source
	source, err := os.ReadFile(tc.FilePath)
	if err != nil {
		t.Fatalf("failed to read source file: %v", err)
	}

	// Create assembler and build
	asm := slasm.New()
	outputPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_%s", tc.Name))

	err = asm.Build(string(source), assembler.BuildOptions{
		OutputPath: outputPath,
	})
	if err != nil {
		if tc.ExpectError {
			// Expected an error, test passes
			return
		}
		t.Fatalf("build failed: %v", err)
	}

	if tc.ExpectError {
		t.Fatalf("expected build error but build succeeded")
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

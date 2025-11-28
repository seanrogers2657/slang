package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSLBuildCommand tests the 'sl build' command
func TestSLBuildCommand(t *testing.T) {
	t.Skip("build command has hardcoded paths that need to be fixed - test the run command instead")

	// TODO: The build command currently has hardcoded paths to _examples/arm64/simple.s
	// which makes it difficult to test in isolation. This should be refactored to use
	// the generated output.s file like the run command does.
}

// TestSLRunCommand tests the 'sl run' command
func TestSLRunCommand(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		description string
	}{
		{
			name:        "simple addition",
			source:      "2 + 5",
			expectError: false, // Program exits with code 0 (success)
			description: "compiles and runs a simple addition",
		},
		{
			name:        "subtraction",
			source:      "5 - 2",
			expectError: false, // Program exits with code 0 (success)
			description: "compiles and runs a subtraction",
		},
		{
			name:        "multiplication",
			source:      "3 * 4",
			expectError: false, // Program exits with code 0 (success)
			description: "compiles and runs a multiplication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary test file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.sl")
			err := os.WriteFile(testFile, []byte(tt.source), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Build the sl binary
			slBinary := filepath.Join(tmpDir, "sl")
			cmd := exec.Command("go", "build", "-o", slBinary, ".")
			cmd.Dir = "."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("failed to build sl binary: %v\nOutput: %s", err, output)
			}

			// Create build directory
			buildDir := filepath.Join(tmpDir, "build")
			err = os.Mkdir(buildDir, 0755)
			if err != nil {
				t.Fatalf("failed to create build directory: %v", err)
			}

			// Change to tmpDir to ensure build artifacts go to the right place
			origDir, _ := os.Getwd()
			defer os.Chdir(origDir)

			err = os.Chdir(tmpDir)
			if err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}

			// Run the run command
			cmd = exec.Command(slBinary, "run", testFile)
			output, err = cmd.CombinedOutput()

			// Check if error matches expectation
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v\nOutput: %s", err, output)
			}

			// Verify that the expected build artifacts were created
			expectedFiles := []string{
				filepath.Join(buildDir, "output.s"),
				filepath.Join(buildDir, "output.o"),
				filepath.Join(buildDir, "output"),
			}

			for _, file := range expectedFiles {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("expected file %s was not created", file)
				}
			}

			// Verify assembly file contains expected content
			asmContent, err := os.ReadFile(filepath.Join(buildDir, "output.s"))
			if err != nil {
				t.Fatalf("failed to read assembly file: %v", err)
			}

			// Basic sanity checks on generated assembly
			asmStr := string(asmContent)
			requiredStrings := []string{".global _start", ".align 4", "_start:", "svc #0"}
			for _, req := range requiredStrings {
				if !strings.Contains(asmStr, req) {
					t.Errorf("generated assembly missing required string: %s", req)
				}
			}
		})
	}
}

// TestSLRunCommandMissingFile tests error handling when source file is missing
func TestSLRunCommandMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Build the sl binary
	slBinary := filepath.Join(tmpDir, "sl")
	cmd := exec.Command("go", "build", "-o", slBinary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build sl binary: %v\nOutput: %s", err, output)
	}

	// Try to run with non-existent file
	cmd = exec.Command(slBinary, "run", "nonexistent.sl")
	output, err = cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error for missing file but got none")
	}

	if !strings.Contains(string(output), "no such file") {
		t.Errorf("expected 'no such file' error, got: %s", output)
	}
}

// TestSLRunCommandNoArguments tests error handling when no arguments are provided
func TestSLRunCommandNoArguments(t *testing.T) {
	tmpDir := t.TempDir()

	// Build the sl binary
	slBinary := filepath.Join(tmpDir, "sl")
	cmd := exec.Command("go", "build", "-o", slBinary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build sl binary: %v\nOutput: %s", err, output)
	}

	// Try to run without file argument
	cmd = exec.Command(slBinary, "run")
	output, err = cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error for missing argument but got none")
	}

	if !strings.Contains(string(output), "source file required") {
		t.Errorf("expected 'source file required' error, got: %s", output)
	}
}

// TestSLRunCommandWithMainFunction tests the 'sl run' command with main function syntax
func TestSLRunCommandWithMainFunction(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		description string
	}{
		{
			name: "simple main function",
			source: `fn main() {
    2 + 5
}`,
			expectError: false,
			description: "compiles and runs a simple main function",
		},
		{
			name: "main function with print",
			source: `fn main() {
    print 42
}`,
			expectError: false,
			description: "compiles and runs main function with print statement",
		},
		{
			name: "main function with multiple statements",
			source: `fn main() {
    print 1
    print 2
    5 + 3
}`,
			expectError: false,
			description: "compiles and runs main function with multiple statements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary test file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.sl")
			err := os.WriteFile(testFile, []byte(tt.source), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Build the sl binary
			slBinary := filepath.Join(tmpDir, "sl")
			cmd := exec.Command("go", "build", "-o", slBinary, ".")
			cmd.Dir = "."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("failed to build sl binary: %v\nOutput: %s", err, output)
			}

			// Create build directory
			buildDir := filepath.Join(tmpDir, "build")
			err = os.Mkdir(buildDir, 0755)
			if err != nil {
				t.Fatalf("failed to create build directory: %v", err)
			}

			// Change to tmpDir to ensure build artifacts go to the right place
			origDir, _ := os.Getwd()
			defer os.Chdir(origDir)

			err = os.Chdir(tmpDir)
			if err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}

			// Run the run command
			cmd = exec.Command(slBinary, "run", testFile)
			output, err = cmd.CombinedOutput()

			// Check if error matches expectation
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v\nOutput: %s", err, output)
			}

			// Verify that the expected build artifacts were created
			expectedFiles := []string{
				filepath.Join(buildDir, "output.s"),
				filepath.Join(buildDir, "output.o"),
				filepath.Join(buildDir, "output"),
			}

			for _, file := range expectedFiles {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("expected file %s was not created", file)
				}
			}

			// Verify assembly file contains expected content
			asmContent, err := os.ReadFile(filepath.Join(buildDir, "output.s"))
			if err != nil {
				t.Fatalf("failed to read assembly file: %v", err)
			}

			// Basic sanity checks on generated assembly
			asmStr := string(asmContent)
			requiredStrings := []string{".global _start", ".align 4", "_start:", "main:", "svc #0"}
			for _, req := range requiredStrings {
				if !strings.Contains(asmStr, req) {
					t.Errorf("generated assembly missing required string: %s", req)
				}
			}
		})
	}
}

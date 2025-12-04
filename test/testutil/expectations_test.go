package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExpectations(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected TestExpectation
	}{
		{
			name: "basic exit code",
			content: `// @test: exit_code=42
.global _start
_start:
    mov x0, #42
`,
			expected: TestExpectation{
				Name:     "test",
				ExitCode: 42,
			},
		},
		{
			name: "semicolon comment style",
			content: `; @test: exit_code=10
.global _start
`,
			expected: TestExpectation{
				Name:     "test",
				ExitCode: 10,
			},
		},
		{
			name: "multiple directives",
			content: `// @test: exit_code=0
// @test: stdout=hello\nworld
fn main() {
`,
			expected: TestExpectation{
				Name:     "test",
				ExitCode: 0,
				Stdout:   "hello\nworld",
			},
		},
		{
			name: "skip directive",
			content: `// @test: skip=not implemented yet
// @test: exit_code=0
fn main() {
`,
			expected: TestExpectation{
				Name:     "test",
				ExitCode: 0,
				Skip:     "not implemented yet",
			},
		},
		{
			name: "expect error",
			content: `// @test: expect_error=true
// @test: error_stage=lexer
// @test: error_contains=invalid character
5 @ 2
`,
			expected: TestExpectation{
				Name:          "test",
				ExitCode:      0,
				ExpectError:   true,
				ErrorStage:    "lexer",
				ErrorContains: "invalid character",
			},
		},
		{
			name: "empty lines before directives",
			content: `
// @test: exit_code=5

.global _start
`,
			expected: TestExpectation{
				Name:     "test",
				ExitCode: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.s")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			exp, err := ParseExpectations(tmpFile)
			if err != nil {
				t.Fatalf("ParseExpectations failed: %v", err)
			}

			if exp.Name != tt.expected.Name {
				t.Errorf("Name: got %q, want %q", exp.Name, tt.expected.Name)
			}
			if exp.ExitCode != tt.expected.ExitCode {
				t.Errorf("ExitCode: got %d, want %d", exp.ExitCode, tt.expected.ExitCode)
			}
			if exp.Stdout != tt.expected.Stdout {
				t.Errorf("Stdout: got %q, want %q", exp.Stdout, tt.expected.Stdout)
			}
			if exp.Skip != tt.expected.Skip {
				t.Errorf("Skip: got %q, want %q", exp.Skip, tt.expected.Skip)
			}
			if exp.ExpectError != tt.expected.ExpectError {
				t.Errorf("ExpectError: got %v, want %v", exp.ExpectError, tt.expected.ExpectError)
			}
			if exp.ErrorStage != tt.expected.ErrorStage {
				t.Errorf("ErrorStage: got %q, want %q", exp.ErrorStage, tt.expected.ErrorStage)
			}
			if exp.ErrorContains != tt.expected.ErrorContains {
				t.Errorf("ErrorContains: got %q, want %q", exp.ErrorContains, tt.expected.ErrorContains)
			}
		})
	}
}

func TestDiscoverTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create some test files
	files := []string{"test1.s", "test2.s", "test3.sl", "readme.txt"}
	for _, f := range files {
		err := os.WriteFile(filepath.Join(tmpDir, f), []byte("content"), 0644)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Test discovering .s files
	matches, err := DiscoverTestFiles(tmpDir, "*.s")
	if err != nil {
		t.Fatalf("DiscoverTestFiles failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 .s files, got %d", len(matches))
	}
}

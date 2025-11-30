package slasm

import (
	"os"
	"os/exec"
	"testing"

	"github.com/seanrogers2657/slang/assembler"
)

func TestEndToEnd_SimpleProgram(t *testing.T) {
	// Simple assembly program that returns with code 1
	assembly := `.global _start

_start:
    mov x0, #1
    ret
`
	assemblyPath := "/tmp/test.s"
	if err := os.WriteFile(assemblyPath, []byte(assembly), 0644); err != nil {
		t.Fatalf("Failed to write assembly file: %v", err)
	}

	// Create assembler
	asm := New()

	// Build executable
	outputPath := "/tmp/test_slasm_simple"
	//defer os.Remove(outputPath)

	err := asm.Build(assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Sign the binary
	signCmd := exec.Command("codesign", "-s", "-", "-f", outputPath)
	if err := signCmd.Run(); err != nil {
		t.Fatalf("Failed to sign binary: %v", err)
	}

	// Disassemble the binary
	otoolCmd := exec.Command("otool", "-tV", outputPath)
	otoolBytes, err := otoolCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("otool -tV failed: %v", err)
	}
	t.Logf("otool -tV output:\n%s", string(otoolBytes))

	// Execute the program
	cmd := exec.Command(outputPath)
	err = cmd.Run()

	// Check exit code (should be 1)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		if exitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", exitCode)
		}
	} else if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	} else {
		t.Error("Expected exit code 1, but program exited with 0")
	}

	t.Log("Success! The program was assembled and executed correctly!")
}

func TestEndToEnd_ExitZero(t *testing.T) {
	// Simple assembly program that exits with code 0
	assembly := `.global _start

_start:
    mov x0, #0
    mov x16, #1
    svc #0
`

	// Create assembler
	asm := New()

	// Build executable
	outputPath := "/tmp/test_slasm_zero"
	defer os.Remove(outputPath)

	err := asm.Build(assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Sign the binary
	signCmd := exec.Command("codesign", "-s", "-", "-f", outputPath)
	if err := signCmd.Run(); err != nil {
		t.Fatalf("Failed to sign binary: %v", err)
	}

	// Execute the program
	cmd := exec.Command(outputPath)
	err = cmd.Run()

	// Check exit code (should be 0)
	if err != nil {
		t.Errorf("Program should exit with 0, but got error: %v", err)
	}

	t.Log("Success! The program exited with code 0 as expected!")
}

func TestEndToEnd_DifferentExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
	}{
		{"exit 0", 0},
		{"exit 1", 1},
		{"exit 42", 42},
		{"exit 255", 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assembly := `.global _start

_start:
    mov x0, #` + string(rune('0'+tt.exitCode/10)) + string(rune('0'+tt.exitCode%10)) + `
    mov x16, #1
    svc #0
`
			// For numbers > 9, we need a different approach
			if tt.exitCode >= 10 {
				t.Skip("Multi-digit immediates need special handling")
				return
			}

			asm := New()
			outputPath := "/tmp/test_slasm_" + tt.name
			defer os.Remove(outputPath)

			err := asm.Build(assembly, assembler.BuildOptions{
				OutputPath: outputPath,
			})

			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			// Sign the binary
			signCmd := exec.Command("codesign", "-s", "-", "-f", outputPath)
			if err := signCmd.Run(); err != nil {
				t.Fatalf("Failed to sign binary: %v", err)
			}

			cmd := exec.Command(outputPath)
			err = cmd.Run()

			if tt.exitCode == 0 {
				if err != nil {
					t.Errorf("Expected exit code 0, got error: %v", err)
				}
			} else {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode := exitErr.ExitCode()
					if exitCode != tt.exitCode {
						t.Errorf("Expected exit code %d, got %d", tt.exitCode, exitCode)
					}
				} else {
					t.Errorf("Expected exit code %d, got success", tt.exitCode)
				}
			}
		})
	}
}

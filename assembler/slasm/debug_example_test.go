package slasm

import (
	"os"
	"os/exec"
	"testing"

	"github.com/seanrogers2657/slang/assembler"
)

// TestDebugExample_MinimalProgram demonstrates the complete build pipeline
// with comprehensive debug output for the simplest possible program
func TestDebugExample_MinimalProgram(t *testing.T) {
	t.Log("========================================")
	t.Log("MINIMAL PROGRAM TEST")
	t.Log("This test creates the simplest possible ARM64 program:")
	t.Log("  - Loads exit code 42 into register x0")
	t.Log("  - Returns (exits with code 42)")
	t.Log("========================================\n")

	// The simplest program: just set a register and return
	assembly := `.global _start

_start:
    mov x0, #42
    ret
`

	// Create assembler
	asm := New()

	// Build executable
	outputPath := "/tmp/test_slasm_minimal"
	defer os.Remove(outputPath)

	t.Log("Starting build process...")
	t.Log("All debug output will be shown below:\n")

	err := asm.Build(assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	t.Log("\n========================================")
	t.Log("BUILD COMPLETED SUCCESSFULLY!")
	t.Log("========================================\n")

	// Verify the binary with otool
	t.Log("Verifying generated binary with otool...\n")
	otoolCmd := exec.Command("otool", "-tV", outputPath)
	otoolBytes, err := otoolCmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: otool failed: %v", err)
	} else {
		t.Logf("Disassembly:\n%s\n", string(otoolBytes))
	}

	// Verify code signature
	t.Log("Verifying code signature...\n")
	verifyCmd := exec.Command("codesign", "--verify", "--verbose", outputPath)
	verifyBytes, err := verifyCmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: codesign verify failed: %v\nOutput: %s", err, string(verifyBytes))
	} else {
		t.Logf("Code signature verification:\n%s\n", string(verifyBytes))
	}

	// Check file size
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}
	t.Logf("Final binary size: %d bytes\n", fileInfo.Size())

	// Execute the program and check exit code
	t.Log("========================================")
	t.Log("EXECUTING PROGRAM...")
	t.Log("========================================\n")

	cmd := exec.Command(outputPath)
	err = cmd.Run()

	// Check exit code (should be 42)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		t.Logf("Program exited with code: %d\n", exitCode)

		if exitCode != 42 {
			t.Errorf("Expected exit code 42, got %d", exitCode)
		} else {
			t.Log("✓ Exit code matches expected value (42)")
		}
	} else if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	} else {
		t.Error("Expected exit code 42, but program exited with 0")
	}

	t.Log("\n========================================")
	t.Log("TEST COMPLETE!")
	t.Log("========================================")
}

// TestDebugExample_SimpleSyscall demonstrates a program that makes a syscall
func TestDebugExample_SimpleSyscall(t *testing.T) {
	t.Log("========================================")
	t.Log("SYSCALL PROGRAM TEST")
	t.Log("This test creates a program that makes the exit syscall:")
	t.Log("  - Loads exit code 7 into x0")
	t.Log("  - Loads syscall number 1 (exit) into x16")
	t.Log("  - Makes the syscall with svc #0")
	t.Log("========================================\n")

	// Program that makes an exit syscall
	assembly := `.global _start

_start:
    mov x0, #7
    mov x16, #1
    svc #0
`

	// Create assembler
	asm := New()

	// Build executable
	outputPath := "/tmp/test_slasm_syscall"
	defer os.Remove(outputPath)

	t.Log("Starting build process...")
	t.Log("All debug output will be shown below:\n")

	err := asm.Build(assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	t.Log("\n========================================")
	t.Log("BUILD COMPLETED SUCCESSFULLY!")
	t.Log("========================================\n")

	// Disassemble
	t.Log("Verifying generated binary with otool...\n")
	otoolCmd := exec.Command("otool", "-tV", outputPath)
	otoolBytes, err := otoolCmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: otool failed: %v", err)
	} else {
		t.Logf("Disassembly:\n%s\n", string(otoolBytes))
	}

	// Execute the program and check exit code
	t.Log("========================================")
	t.Log("EXECUTING PROGRAM...")
	t.Log("========================================\n")

	cmd := exec.Command(outputPath)
	err = cmd.Run()

	// Check exit code (should be 7)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		t.Logf("Program exited with code: %d\n", exitCode)

		if exitCode != 7 {
			t.Errorf("Expected exit code 7, got %d", exitCode)
		} else {
			t.Log("✓ Exit code matches expected value (7)")
		}
	} else if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	} else {
		t.Error("Expected exit code 7, but program exited with 0")
	}

	t.Log("\n========================================")
	t.Log("TEST COMPLETE!")
	t.Log("========================================")
}

// TestDebugExample_Arithmetic demonstrates instruction encoding for arithmetic
func TestDebugExample_Arithmetic(t *testing.T) {
	t.Log("========================================")
	t.Log("ARITHMETIC PROGRAM TEST")
	t.Log("This test creates a program with arithmetic operations:")
	t.Log("  - Loads values into registers")
	t.Log("  - Performs add and sub operations")
	t.Log("  - Returns with the result")
	t.Log("========================================\n")

	// Program with arithmetic instructions
	assembly := `.global _start

_start:
    mov x0, #10
    mov x1, #5
    add x2, x0, x1
    sub x0, x2, x1
    ret
`

	// Create assembler
	asm := New()

	// Build executable
	outputPath := "/tmp/test_slasm_arithmetic"
	defer os.Remove(outputPath)

	t.Log("Starting build process...")
	t.Log("All debug output will be shown below:\n")

	err := asm.Build(assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	t.Log("\n========================================")
	t.Log("BUILD COMPLETED SUCCESSFULLY!")
	t.Log("========================================\n")

	// Disassemble to verify instruction encoding
	t.Log("Verifying generated binary with otool...\n")
	otoolCmd := exec.Command("otool", "-tV", outputPath)
	otoolBytes, err := otoolCmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: otool failed: %v", err)
	} else {
		t.Logf("Disassembly:\n%s\n", string(otoolBytes))
		t.Log("Compare the disassembly above with the encoding shown in STEP 4")
	}

	// Execute the program
	t.Log("\n========================================")
	t.Log("EXECUTING PROGRAM...")
	t.Log("========================================\n")

	cmd := exec.Command(outputPath)
	err = cmd.Run()

	// The program does: x0=10, x1=5, x2=10+5=15, x0=15-5=10, so exit code should be 10
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		t.Logf("Program exited with code: %d\n", exitCode)

		if exitCode != 10 {
			t.Errorf("Expected exit code 10 (from arithmetic 10+5-5), got %d", exitCode)
		} else {
			t.Log("✓ Exit code matches expected value (10)")
		}
	} else if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	} else {
		t.Error("Expected exit code 10, but program exited with 0")
	}

	t.Log("\n========================================")
	t.Log("TEST COMPLETE!")
	t.Log("========================================")
}

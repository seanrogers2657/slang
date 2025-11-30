// slasm-debug is a diagnostic tool for the slasm assembler.
// It builds a test program and shows detailed output from every stage of the pipeline.
//
// Usage:
//   go run cmd/slasm-debug/main.go
//   # or
//   go build -o slasm-debug cmd/slasm-debug/main.go
//   ./slasm-debug

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/slasm"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         SLASM ASSEMBLER - DEBUG BUILD EXAMPLE                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Simple test program
	assembly := `.global _start

_start:
    mov x0, #42
    mov x16, #1
    svc #0
`

	fmt.Println("Source Assembly:")
	fmt.Println("----------------")
	fmt.Println(assembly)
	fmt.Println()

	// Create assembler
	asm := slasm.New()

	// Build executable
	outputPath := "./test_slasm_binary"

	fmt.Println("Building executable...")
	fmt.Println()

	err := asm.Build(assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   VERIFICATION STEPS                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. File info
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to stat file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ File created: %s (%d bytes)\n\n", outputPath, fileInfo.Size())

	// 2. File type
	fmt.Println("File type (using 'file' command):")
	fileCmd := exec.Command("file", outputPath)
	fileOutput, _ := fileCmd.Output()
	fmt.Printf("  %s\n", string(fileOutput))

	// 3. Disassemble with otool
	fmt.Println("Disassembly (using 'otool -tV'):")
	fmt.Println("----------------------------------")
	otoolCmd := exec.Command("otool", "-tV", outputPath)
	otoolOutput, err := otoolCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: otool failed: %v\n", err)
	} else {
		fmt.Printf("%s\n", string(otoolOutput))
	}

	// 4. Verify code signature
	fmt.Println("Code Signature Verification (using 'codesign --verify --verbose'):")
	fmt.Println("-------------------------------------------------------------------")
	verifyCmd := exec.Command("codesign", "--verify", "--verbose", outputPath)
	verifyOutput, err := verifyCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  Warning: %v\n  %s\n", err, string(verifyOutput))
	} else {
		fmt.Printf("  ✓ %s\n", string(verifyOutput))
	}

	// 5. Display code signature info
	fmt.Println("\nCode Signature Info (using 'codesign --display --verbose=4'):")
	fmt.Println("--------------------------------------------------------------")
	displayCmd := exec.Command("codesign", "--display", "--verbose=4", outputPath)
	displayOutput, _ := displayCmd.CombinedOutput()
	fmt.Printf("%s\n", string(displayOutput))

	// 6. Check Mach-O header
	fmt.Println("Mach-O Header (using 'otool -hv'):")
	fmt.Println("-----------------------------------")
	headerCmd := exec.Command("otool", "-hv", outputPath)
	headerOutput, _ := headerCmd.Output()
	fmt.Printf("%s\n", string(headerOutput))

	// 7. List load commands
	fmt.Println("Load Commands (using 'otool -l'):")
	fmt.Println("----------------------------------")
	loadCmd := exec.Command("otool", "-l", outputPath)
	loadOutput, _ := loadCmd.Output()
	fmt.Printf("%s\n", string(loadOutput))

	// 8. Try to execute
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   EXECUTION TEST                               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Attempting to execute the binary...")
	fmt.Println()

	execCmd := exec.Command(outputPath)
	err = execCmd.Run()

	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		fmt.Printf("Program exited with code: %d\n", exitCode)
		if exitCode == 42 {
			fmt.Println("✓ SUCCESS! Exit code matches expected value (42)")
		} else {
			fmt.Printf("❌ Expected exit code 42, got %d\n", exitCode)
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Execution failed: %v\n", err)
		fmt.Println("\nThis is the known issue mentioned in the README:")
		fmt.Println("The binary is generated correctly but fails at runtime.")
	} else {
		fmt.Println("Program exited with code: 0")
		fmt.Println("❌ Expected exit code 42, got 0")
	}

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        COMPLETE!                               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nGenerated binary: %s\n", outputPath)
	fmt.Println("You can examine it with the commands shown above.")
}

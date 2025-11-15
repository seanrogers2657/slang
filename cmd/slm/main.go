package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "make",
		Usage: "Build tool for the Slang compiler",
		Commands: []*cli.Command{
			// Test commands
			{
				Name:  "test",
				Usage: "Run all tests",
				Action: func(c *cli.Context) error {
					fmt.Println("Running all tests...")
					return runCommand("go", "test", "./...", "-v")
				},
			},
			{
				Name:  "test-verbose",
				Usage: "Run tests with verbose output",
				Action: func(c *cli.Context) error {
					fmt.Println("Running tests with verbose output...")
					return runCommand("go", "test", "./...", "-v", "-count=1")
				},
			},
			{
				Name:  "test-coverage",
				Usage: "Run tests and generate coverage report",
				Action: func(c *cli.Context) error {
					fmt.Println("Running tests with coverage...")
					if err := runCommand("go", "test", "./...", "-coverprofile=coverage.out"); err != nil {
						return err
					}
					if err := runCommand("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html"); err != nil {
						return err
					}
					fmt.Println("Coverage report generated: coverage.html")
					return nil
				},
			},
			{
				Name:  "test-integration",
				Usage: "Run only integration tests",
				Action: func(c *cli.Context) error {
					fmt.Println("Running integration tests...")
					tests := []string{"TestEndToEnd", "TestCompilationPipeline", "TestExampleFile", "TestRegressions"}
					for _, test := range tests {
						if err := runCommand("go", "test", "-run", test, "-v"); err != nil {
							return err
						}
					}
					return nil
				},
			},
			{
				Name:  "test-report",
				Usage: "Run tests with detailed report",
				Action: func(c *cli.Context) error {
					fmt.Println("Running tests with detailed report...")
					cmd := exec.Command("sh", "-c", `go test ./... -v -json | grep -E '"Test"|"Pass"|"Fail"'`)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					return cmd.Run()
				},
			},
			{
				Name:  "bench",
				Usage: "Run benchmarks",
				Action: func(c *cli.Context) error {
					fmt.Println("Running benchmarks...")
					return runCommand("go", "test", "./...", "-bench=.", "-benchmem")
				},
			},
			{
				Name:  "test-race",
				Usage: "Run tests with race detector",
				Action: func(c *cli.Context) error {
					fmt.Println("Running tests with race detector...")
					return runCommand("go", "test", "./...", "-race", "-v")
				},
			},
			// Build commands
			{
				Name:  "build",
				Usage: "Build the compiler binary",
				Action: func(c *cli.Context) error {
					fmt.Println("Building compiler...")
					if err := runCommand("go", "build", "-o", "sl", "./cmd/sl"); err != nil {
						return err
					}
					fmt.Println("Build complete: sl")
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "Run compiler on example file",
				Action: func(c *cli.Context) error {
					fmt.Println("Running compiler on example file...")
					return runCommand("go", "run", "./cmd/sl", "_examples/slang/add.sl")
				},
			},
			{
				Name:  "run-and-test",
				Usage: "Compile, assemble, link, and run example",
				Action: func(c *cli.Context) error {
					fmt.Println("Compiling example...")
					if err := runCommand("go", "run", "./cmd/sl", "_examples/slang/add.sl"); err != nil {
						return err
					}

					fmt.Println("Assembling output...")
					if err := runCommand("as", "-arch", "arm64", "build/output.s", "-o", "build/simple.o"); err != nil {
						return err
					}

					fmt.Println("Linking...")
					sdkPath, err := exec.Command("xcrun", "-sdk", "macosx", "--show-sdk-path").Output()
					if err != nil {
						return fmt.Errorf("failed to get SDK path: %w", err)
					}
					sdkPathStr := string(sdkPath[:len(sdkPath)-1]) // Remove trailing newline

					if err := runCommand("ld", "build/simple.o", "-o", "build/simple", "-lSystem",
						"-syslibroot", sdkPathStr, "-e", "_start", "-arch", "arm64"); err != nil {
						return err
					}

					fmt.Println("Running compiled binary...")
					cmd := exec.Command("./build/simple")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							fmt.Printf("Exit code: %d\n", exitErr.ExitCode())
							return nil
						}
						return err
					}
					fmt.Println("Exit code: 0")
					return nil
				},
			},
			// Clean command
			{
				Name:  "clean",
				Usage: "Remove test artifacts and build files",
				Action: func(c *cli.Context) error {
					fmt.Println("Cleaning test artifacts...")
					filesToRemove := []string{"coverage.out", "coverage.html"}
					for _, file := range filesToRemove {
						os.Remove(file)
					}

					buildFiles := []string{"build/*.o", "build/*.s", "build/output", "build/simple"}
					for _, pattern := range buildFiles {
						matches, _ := filepath.Glob(pattern)
						for _, file := range matches {
							os.Remove(file)
						}
					}
					fmt.Println("Clean complete")
					return nil
				},
			},
			// Code quality commands
			{
				Name:  "fmt",
				Usage: "Format source code",
				Action: func(c *cli.Context) error {
					fmt.Println("Formatting code...")
					if err := runCommand("go", "fmt", "./..."); err != nil {
						return err
					}
					fmt.Println("Formatting complete")
					return nil
				},
			},
			{
				Name:  "lint",
				Usage: "Run linter",
				Action: func(c *cli.Context) error {
					fmt.Println("Running linter...")
					if err := runCommand("go", "vet", "./..."); err != nil {
						return err
					}
					fmt.Println("Linting complete")
					return nil
				},
			},
			{
				Name:  "check",
				Usage: "Run fmt, lint, and test",
				Action: func(c *cli.Context) error {
					// Run fmt
					fmt.Println("Formatting code...")
					if err := runCommand("go", "fmt", "./..."); err != nil {
						return err
					}

					// Run lint
					fmt.Println("Running linter...")
					if err := runCommand("go", "vet", "./..."); err != nil {
						return err
					}

					// Run tests
					fmt.Println("Running all tests...")
					if err := runCommand("go", "test", "./...", "-v"); err != nil {
						return err
					}

					fmt.Println("All quality checks passed!")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runCommand executes a command and streams output to stdout/stderr
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

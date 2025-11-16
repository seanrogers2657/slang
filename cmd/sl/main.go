package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
	"github.com/seanrogers2657/slang/backend/as"
	"github.com/seanrogers2657/slang/frontend/lexer"
	"github.com/seanrogers2657/slang/frontend/parser"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "sl",
		Usage: "Compile a Slang program",
		Commands: []*cli.Command{
			{
				Name:      "build",
				Usage:     "Build a Slang source file",
				ArgsUsage: "<source-file>",
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					dat, err := os.ReadFile(file)
					if err != nil {
						log.Fatal(err)
					}

					lexer := lexer.NewLexer(dat)
					lexer.Parse()
					spew.Dump("done lexing...")

					parser := parser.NewParser(lexer.Tokens)
					ast := parser.Parse()
					spew.Dump("done parsing...")

					codeGenerator := as.NewAsGenerator(ast)
					assemblyOutput, err := codeGenerator.Generate()
					if err != nil {
						panic(err)
					}

					err = os.WriteFile("build/output.s", []byte(assemblyOutput), fs.ModePerm)
					if err != nil {
						panic(err)
					}

					cmd := exec.Command("as", "-arch", "arm64", "_examples/arm64/simple.s", "-o", "build/simple.o")
					err = cmd.Run()
					if err != nil {
						panic(err)
					}

					// cmd = exec.Command("xcrun", "-sdk", "macosx", "--show-sdk-path")
					// err = cmd.Run()
					// if err != nil {
					// 	panic(err)
					// }
					//sdkPath, err := cmd.Output()
					sdkPath := "/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX15.5.sdk"

					cmd = exec.Command(
						"ld",
						"-o",
						"build/simple",
						"build/simple.o",
						"-lSystem",
						"-syslibroot",
						sdkPath,
						"-e",
						"_start",
						"-arch",
						"arm64",
					)
					err = cmd.Run()
					fmt.Print(err)

					if err != nil {
						panic(err)
					}

					spew.Dump("compilation done")
					return nil
				},
			},
			{
				Name:      "run",
				Usage:     "Compile and execute a Slang source file",
				ArgsUsage: "<source-file>",
				Action: func(c *cli.Context) error {
					file := c.Args().First()
					if file == "" {
						return fmt.Errorf("source file required")
					}

					// Read and compile the source file
					dat, err := os.ReadFile(file)
					if err != nil {
						log.Fatal(err)
					}

					lexer := lexer.NewLexer(dat)
					lexer.Parse()

					parser := parser.NewParser(lexer.Tokens)
					ast := parser.Parse()

					codeGenerator := as.NewAsGenerator(ast)
					assemblyOutput, err := codeGenerator.Generate()
					if err != nil {
						return fmt.Errorf("code generation failed: %w", err)
					}

					// Write assembly output
					err = os.WriteFile("build/output.s", []byte(assemblyOutput), fs.ModePerm)
					if err != nil {
						return fmt.Errorf("failed to write assembly: %w", err)
					}

					// Assemble
					cmd := exec.Command("as", "-arch", "arm64", "build/output.s", "-o", "build/output.o")
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("assembly failed: %w", err)
					}

					// Link
					sdkPath := "/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX15.5.sdk"
					cmd = exec.Command(
						"ld",
						"-o",
						"build/output",
						"build/output.o",
						"-lSystem",
						"-syslibroot",
						sdkPath,
						"-e",
						"_start",
						"-arch",
						"arm64",
					)
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("linking failed: %w", err)
					}

					// Execute the compiled binary
					cmd = exec.Command("build/output")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							return fmt.Errorf("program exited with code %d", exitErr.ExitCode())
						}
						return fmt.Errorf("execution failed: %w", err)
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
		log.Fatal(err)
	}
}

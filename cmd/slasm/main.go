package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/slasm"
	"github.com/seanrogers2657/slang/assembler/system"
	"github.com/seanrogers2657/slang/errors"
	"github.com/urfave/cli/v2"
)

// Global error handler for slasm
var errorHandler = errors.NewHandler(errors.ToolSlasm)

// createAssembler creates the appropriate assembler based on the backend flag
func createAssembler(backend string) (assembler.Assembler, error) {
	switch backend {
	case "system":
		return system.New(), nil
	case "slasm":
		return slasm.New(), nil
	default:
		return nil, fmt.Errorf("unknown backend: %s (valid options: system, slasm)", backend)
	}
}

func main() {
	app := &cli.App{
		Name:  "slasm",
		Usage: "Assemble and link ARM64 assembly programs",
		Commands: []*cli.Command{
			{
				Name:      "assemble",
				Aliases:   []string{"a"},
				Usage:     "Assemble an ARM64 assembly file to an object file",
				ArgsUsage: "<input.s>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Output object file path (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "arch",
						Aliases: []string{"a"},
						Value:   "arm64",
						Usage:   "Target architecture",
					},
					&cli.StringFlag{
						Name:    "backend",
						Aliases: []string{"b"},
						Value:   "system",
						Usage:   "Assembler backend to use (system or slasm)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose debug output",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("requires exactly 1 argument: <input.s>")
					}

					inputPath := c.Args().First()
					outputPath := c.String("output")

					// Create assembler based on backend flag
					asm, err := createAssembler(c.String("backend"))
					if err != nil {
						return err
					}

					// Set architecture and verbose (both implementations support this field)
					switch a := asm.(type) {
					case *system.SystemAssembler:
						a.Arch = c.String("arch")
					case *slasm.NativeAssembler:
						a.Arch = c.String("arch")
						a.Logger.SetEnabled(c.Bool("verbose"))
					}

					fmt.Printf("Assembling %s -> %s (backend: %s)\n", inputPath, outputPath, c.String("backend"))
					if err := asm.Assemble(inputPath, outputPath); err != nil {
						return fmt.Errorf("[assemble] %w", err)
					}

					fmt.Printf("Successfully created object file: %s\n", outputPath)
					return nil
				},
			},
			{
				Name:      "link",
				Aliases:   []string{"l"},
				Usage:     "Link object files to create an executable",
				ArgsUsage: "<input.o> [input2.o ...]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Output executable path (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "arch",
						Aliases: []string{"a"},
						Value:   "arm64",
						Usage:   "Target architecture",
					},
					&cli.StringFlag{
						Name:    "sdk",
						Aliases: []string{"s"},
						Usage:   "Path to macOS SDK (auto-detected if not specified)",
					},
					&cli.StringFlag{
						Name:    "entry",
						Aliases: []string{"e"},
						Value:   "_start",
						Usage:   "Entry point symbol",
					},
					&cli.BoolFlag{
						Name:  "no-system",
						Usage: "Don't link system libraries",
					},
					&cli.StringFlag{
						Name:    "backend",
						Aliases: []string{"b"},
						Value:   "system",
						Usage:   "Assembler backend to use (system or slasm)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose debug output",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("at least one object file required")
					}

					// Collect all object files from arguments
					objectFiles := c.Args().Slice()
					outputPath := c.String("output")

					// Create assembler based on backend flag
					asm, err := createAssembler(c.String("backend"))
					if err != nil {
						return err
					}

					// Set configuration based on assembler type
					switch a := asm.(type) {
					case *system.SystemAssembler:
						a.Arch = c.String("arch")
						a.EntryPoint = c.String("entry")
						a.SystemLibs = !c.Bool("no-system")
						if c.String("sdk") != "" {
							a.SDKPath = c.String("sdk")
						}
					case *slasm.NativeAssembler:
						a.Arch = c.String("arch")
						a.EntryPoint = c.String("entry")
						a.SystemLibs = !c.Bool("no-system")
						if c.String("sdk") != "" {
							a.SDKPath = c.String("sdk")
						}
						a.Logger.SetEnabled(c.Bool("verbose"))
					}

					fmt.Printf("Linking %s -> %s (backend: %s)\n", strings.Join(objectFiles, ", "), outputPath, c.String("backend"))
					if err := asm.Link(objectFiles, outputPath); err != nil {
						return fmt.Errorf("[link] %w", err)
					}

					fmt.Printf("Successfully created executable: %s\n", outputPath)
					return nil
				},
			},
			{
				Name:      "build",
				Aliases:   []string{"b"},
				Usage:     "Assemble and link an assembly file to create an executable",
				ArgsUsage: "<input.s>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Output executable path (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "arch",
						Aliases: []string{"a"},
						Value:   "arm64",
						Usage:   "Target architecture",
					},
					&cli.StringFlag{
						Name:    "sdk",
						Aliases: []string{"s"},
						Usage:   "Path to macOS SDK (auto-detected if not specified)",
					},
					&cli.StringFlag{
						Name:    "entry",
						Aliases: []string{"e"},
						Value:   "_start",
						Usage:   "Entry point symbol",
					},
					&cli.BoolFlag{
						Name:  "keep-intermediates",
						Usage: "Keep intermediate .o files",
					},
					&cli.StringFlag{
						Name:    "backend",
						Aliases: []string{"be"},
						Value:   "system",
						Usage:   "Assembler backend to use (system or slasm)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Enable verbose debug output",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("requires exactly 1 argument: <input.s>")
					}

					inputPath := c.Args().First()
					outputPath := c.String("output")

					// Read assembly file
					assemblyCode, err := os.ReadFile(inputPath)
					if err != nil {
						return fmt.Errorf("failed to read assembly file: %w", err)
					}

					// Create assembler based on backend flag
					asm, err := createAssembler(c.String("backend"))
					if err != nil {
						return err
					}

					// Set configuration based on assembler type
					switch a := asm.(type) {
					case *system.SystemAssembler:
						a.Arch = c.String("arch")
						a.EntryPoint = c.String("entry")
						if c.String("sdk") != "" {
							a.SDKPath = c.String("sdk")
						}
					case *slasm.NativeAssembler:
						a.Arch = c.String("arch")
						a.EntryPoint = c.String("entry")
						if c.String("sdk") != "" {
							a.SDKPath = c.String("sdk")
						}
						a.Logger.SetEnabled(c.Bool("verbose"))
					}

					fmt.Printf("Building %s -> %s (backend: %s)\n", inputPath, outputPath, c.String("backend"))

					// Build
					opts := assembler.BuildOptions{
						AssemblyPath:      inputPath,
						OutputPath:        outputPath,
						KeepIntermediates: c.Bool("keep-intermediates"),
					}

					if err := asm.Build(string(assemblyCode), opts); err != nil {
						return fmt.Errorf("[build] %w", err)
					}

					fmt.Printf("Successfully created executable: %s\n", outputPath)
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			return cli.ShowAppHelp(c)
		},
	}

	if err := app.Run(os.Args); err != nil {
		// Wrap the error with slasm tool identification and display
		compilerErr := errorHandler.Wrap(err, "")
		errorHandler.Handle([]*errors.CompilerError{compilerErr})
		os.Exit(1)
	}
}

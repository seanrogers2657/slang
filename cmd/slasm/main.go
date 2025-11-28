package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/assembler/system"
	"github.com/urfave/cli/v2"
)

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
				},
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("requires exactly 1 argument: <input.s>")
					}

					inputPath := c.Args().First()
					outputPath := c.String("output")

					// Create assembler and assemble
					asm := system.New()
					asm.Arch = c.String("arch")

					fmt.Printf("Assembling %s -> %s\n", inputPath, outputPath)
					if err := asm.Assemble(inputPath, outputPath); err != nil {
						return err
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
				},
				Action: func(c *cli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("at least one object file required")
					}

					// Collect all object files from arguments
					objectFiles := c.Args().Slice()
					outputPath := c.String("output")

					// Create assembler (which also handles linking)
					asm := system.New()
					asm.Arch = c.String("arch")
					asm.EntryPoint = c.String("entry")
					asm.SystemLibs = !c.Bool("no-system")
					if c.String("sdk") != "" {
						asm.SDKPath = c.String("sdk")
					}

					fmt.Printf("Linking %s -> %s\n", strings.Join(objectFiles, ", "), outputPath)
					if err := asm.Link(objectFiles, outputPath); err != nil {
						return err
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

					// Create assembler
					asm := system.New()
					asm.Arch = c.String("arch")
					asm.EntryPoint = c.String("entry")
					if c.String("sdk") != "" {
						asm.SDKPath = c.String("sdk")
					}

					fmt.Printf("Building %s -> %s\n", inputPath, outputPath)

					// Build
					opts := assembler.BuildOptions{
						AssemblyPath:      inputPath,
						OutputPath:        outputPath,
						KeepIntermediates: c.Bool("keep-intermediates"),
					}

					if err := asm.Build(string(assemblyCode), opts); err != nil {
						return err
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
		log.Fatal(err)
	}
}

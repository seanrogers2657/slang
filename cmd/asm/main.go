package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "asm",
		Usage: "Assemble and link ARM64 assembly programs",
		Commands: []*cli.Command{
			{
				Name:      "assemble",
				Aliases:   []string{"a"},
				Usage:     "Assemble an ARM64 assembly file to an object file",
				ArgsUsage: "<input.s> [output.o]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "arch",
						Aliases: []string{"a"},
						Value:   "arm64",
						Usage:   "Target architecture",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output object file path",
					},
				},
				Action: func(c *cli.Context) error {
					inputPath := c.Args().First()
					if inputPath == "" {
						return fmt.Errorf("input assembly file required")
					}

					// Determine output path
					outputPath := c.String("output")
					if outputPath == "" {
						outputPath = c.Args().Get(1)
					}
					if outputPath == "" {
						// Default: replace .s extension with .o
						ext := filepath.Ext(inputPath)
						if ext == ".s" {
							outputPath = strings.TrimSuffix(inputPath, ext) + ".o"
						} else {
							outputPath = inputPath + ".o"
						}
					}

					// Create assembler and assemble
					asm := assembler.New()
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
				ArgsUsage: "<input.o> [input2.o ...] [output]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "arch",
						Aliases: []string{"a"},
						Value:   "arm64",
						Usage:   "Target architecture",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output executable path",
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

					// Collect all object files
					var objectFiles []string
					outputPath := c.String("output")

					for i := 0; i < c.NArg(); i++ {
						arg := c.Args().Get(i)
						// If output not specified and this is the last arg without .o extension,
						// treat it as output path
						if outputPath == "" && i == c.NArg()-1 && !strings.HasSuffix(arg, ".o") {
							outputPath = arg
						} else {
							objectFiles = append(objectFiles, arg)
						}
					}

					if len(objectFiles) == 0 {
						return fmt.Errorf("no object files specified")
					}

					// Default output path
					if outputPath == "" {
						// Use the name of the first object file without extension
						base := filepath.Base(objectFiles[0])
						outputPath = strings.TrimSuffix(base, filepath.Ext(base))
					}

					// Create assembler (which also handles linking)
					asm := assembler.New()
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
				ArgsUsage: "<input.s> [output]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "arch",
						Aliases: []string{"a"},
						Value:   "arm64",
						Usage:   "Target architecture",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output executable path",
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
					inputPath := c.Args().First()
					if inputPath == "" {
						return fmt.Errorf("input assembly file required")
					}

					// Determine output path
					outputPath := c.String("output")
					if outputPath == "" {
						outputPath = c.Args().Get(1)
					}
					if outputPath == "" {
						// Default: use input filename without extension
						ext := filepath.Ext(inputPath)
						if ext == ".s" {
							outputPath = strings.TrimSuffix(inputPath, ext)
						} else {
							outputPath = inputPath + ".out"
						}
					}

					// Read assembly file
					assemblyCode, err := os.ReadFile(inputPath)
					if err != nil {
						return fmt.Errorf("failed to read assembly file: %w", err)
					}

					// Create assembler
					asm := assembler.New()
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

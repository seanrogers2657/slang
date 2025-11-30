package slasm

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/seanrogers2657/slang/assembler"
)

// NativeAssembler is a custom implementation of Assembler that directly generates
// machine code without relying on system tools (as, ld)
type NativeAssembler struct {
	// Arch specifies the target architecture (currently only arm64 is supported)
	Arch string
	// EntryPoint is the entry point symbol (default: _start)
	EntryPoint string
	// SystemLibs specifies whether to link system libraries (default: true)
	SystemLibs bool
	// SDKPath is the path to the macOS SDK (optional)
	SDKPath string
	// Logger for assembler output (defaults to enabled stderr logger)
	Logger *Logger
}

// New creates a new NativeAssembler with default settings
func New() *NativeAssembler {
	return &NativeAssembler{
		Arch:       "arm64",
		EntryPoint: "_start",
		SystemLibs: true,
		Logger:     NewDefaultLogger(false), // Disabled by default, enable with --verbose
	}
}

// Assemble converts an assembly file (.s) to an object file (.o)
// This is the native implementation that parses and encodes ARM64 assembly
func (a *NativeAssembler) Assemble(inputPath, outputPath string) error {
	// TODO: Implement native assembly
	// 1. Read assembly file
	// 2. Lex tokens
	// 3. Parse into IR
	// 4. Resolve symbols (two-pass)
	// 5. Encode instructions
	// 6. Generate Mach-O object file
	return fmt.Errorf("native assembler not yet implemented")
}

// Link creates an executable from object files
// This is the native implementation that links object files without using ld
func (a *NativeAssembler) Link(objectFiles []string, outputPath string) error {
	// TODO: Implement native linker
	// 1. Read all object files
	// 2. Resolve symbols across files
	// 3. Apply relocations
	// 4. Generate executable Mach-O
	// 5. Link with system libraries if needed
	return fmt.Errorf("native linker not yet implemented")
}

// Build performs the complete build process from assembly string to executable
func (a *NativeAssembler) Build(assembly string, opts assembler.BuildOptions) error {
	a.Logger.Header("========== SLASM ASSEMBLER - BUILD PIPELINE ==========")

	// Step 1: Lex the assembly source
	a.Logger.Section("STEP 1: LEXER")
	a.Logger.Printf("Input assembly:\n%s\n\n", assembly)

	lexer := NewLexer(assembly)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("lexer error: %w", err)
	}

	a.Logger.Printf("Lexer produced %d tokens:\n", len(tokens))
	for i, token := range tokens {
		a.Logger.Printf("  [%3d] %-15v: %s\n", i, token.Type, token.Value)
	}
	a.Logger.Println()

	// Step 2: Parse tokens into IR
	a.Logger.Section("STEP 2: PARSER")

	parser := NewParser(tokens)
	program, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parser error: %w", err)
	}

	a.Logger.Printf("Parser produced %d section(s):\n", len(program.Sections))
	for i, section := range program.Sections {
		a.Logger.Printf("  Section %d: %v (%d items)\n", i, section.Type, len(section.Items))
		for j, item := range section.Items {
			switch v := item.(type) {
			case *Label:
				a.Logger.Printf("    [%3d] Label: %s\n", j, v.Name)
			case *Instruction:
				operands := ""
				for k, op := range v.Operands {
					if k > 0 {
						operands += ", "
					}
					operands += op.Value
				}
				a.Logger.Printf("    [%3d] Instruction: %-6s %s\n", j, v.Mnemonic, operands)
			case *Directive:
				a.Logger.Printf("    [%3d] Directive: .%s %v\n", j, v.Name, v.Args)
			}
		}
	}
	a.Logger.Printf("\n")

	// Step 3: Calculate layout and build symbol table
	a.Logger.Section("STEP 3: LAYOUT & SYMBOL TABLE")

	layout := NewLayout(program)
	err = layout.Calculate()
	if err != nil {
		return fmt.Errorf("layout error: %w", err)
	}

	symbolTable := layout.GetSymbolTable()
	a.Logger.Printf("Symbol table:\n")
	for name, sym := range symbolTable.symbols {
		globalFlag := ""
		if sym.Global {
			globalFlag = " [GLOBAL]"
		}
		a.Logger.Printf("  %-15s: addr=0x%04x section=%v%s\n", name, sym.Address, sym.Section, globalFlag)
	}
	a.Logger.Printf("\n")

	// Step 4: Encode instructions to machine code
	a.Logger.Section("STEP 4: INSTRUCTION ENCODING")
	

	encoder := NewEncoder(layout.GetSymbolTable())
	var codeBytes []byte
	instructionCount := 0

	for _, section := range program.Sections {
		if section.Type == SectionText {
			for _, item := range section.Items {
				if inst, ok := item.(*Instruction); ok {
					currentAddr := uint64(len(codeBytes))
					machineCode, err := encoder.Encode(inst, currentAddr)
					if err != nil {
						return fmt.Errorf("encoding error for instruction '%s': %w", inst.Mnemonic, err)
					}

					// Format operands for display
					operands := ""
					for k, op := range inst.Operands {
						if k > 0 {
							operands += ", "
						}
						operands += op.Value
					}

					a.Logger.Printf("  [0x%04x] %-20s -> %02x %02x %02x %02x (0x%08x)\n",
						currentAddr,
						fmt.Sprintf("%s %s", inst.Mnemonic, operands),
						machineCode[0], machineCode[1], machineCode[2], machineCode[3],
						uint32(machineCode[0]) | uint32(machineCode[1])<<8 | uint32(machineCode[2])<<16 | uint32(machineCode[3])<<24)

					codeBytes = append(codeBytes, machineCode...)
					instructionCount++
				}
			}
		}
	}

	a.Logger.Printf("\nEncoded %d instructions (%d bytes total)\n", instructionCount, len(codeBytes))
	a.Logger.Printf("Complete machine code: %x\n\n", codeBytes)

	// Step 5: Generate Mach-O executable
	a.Logger.Section("STEP 5: MACH-O GENERATION")


	writer := NewMachOWriter(a.Arch, a.Logger)
	err = writer.WriteExecutable(opts.OutputPath, codeBytes, nil, layout.GetSymbolTable(), a.EntryPoint)
	if err != nil {
		return fmt.Errorf("mach-o generation error: %w", err)
	}

	a.Logger.Printf("Generated Mach-O executable: %s\n", opts.OutputPath)

	// Step 6: Make the file executable
	a.Logger.Section("\nSTEP 6: FILE PERMISSIONS")

	err = makeExecutable(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("chmod error: %w", err)
	}

	a.Logger.Printf("Set executable permissions (0755)\n")

	// Step 7: Sign the binary (required on macOS)
	a.Logger.Section("\nSTEP 7: CODE SIGNING")

	err = signBinary(opts.OutputPath)
	if err != nil {
		// Non-fatal: binary is still usable but may not run without signing
		a.Logger.Printf("Warning: failed to sign binary: %v\n", err)
	} else {
		a.Logger.Printf("Successfully signed binary with ad-hoc signature\n")
	}

	// Summary
	a.Logger.Printf("\n========== BUILD SUMMARY ==========\n")
	a.Logger.Printf("Output file:       %s\n", opts.OutputPath)
	a.Logger.Printf("Architecture:      %s\n", a.Arch)
	a.Logger.Printf("Entry point:       %s\n", a.EntryPoint)
	a.Logger.Printf("Instructions:      %d\n", instructionCount)
	a.Logger.Printf("Code size:         %d bytes\n", len(codeBytes))
	a.Logger.Printf("Symbols:           %d\n", len(symbolTable.symbols))
	a.Logger.Printf("===================================\n\n")

	return nil
}

// makeExecutable sets the executable permission on a file
func makeExecutable(path string) error {
	return os.Chmod(path, 0755)
}

// signBinary signs the binary with an ad-hoc signature
func signBinary(path string) error {
	cmd := exec.Command("codesign", "--sign", "-", "--force", "--deep", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Return more detailed error
		return fmt.Errorf("codesign failed: %w, output: %s", err, string(output))
	}
	return nil
}

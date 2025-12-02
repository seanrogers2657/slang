package slasm

import (
	"fmt"
	"os"

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

	// Write intermediate files if paths are specified
	if opts.AssemblyPath != "" {
		// Ensure directory exists
		if err := os.MkdirAll("build", 0755); err != nil {
			return fmt.Errorf("failed to create build directory: %w", err)
		}
		if err := os.WriteFile(opts.AssemblyPath, []byte(assembly), 0644); err != nil {
			return fmt.Errorf("failed to write assembly file: %w", err)
		}
		a.Logger.Printf("Wrote assembly to %s\n", opts.AssemblyPath)
	}

	// Create placeholder object file (slasm doesn't use separate object files)
	if opts.ObjectPath != "" && opts.KeepIntermediates {
		// Ensure directory exists
		if err := os.MkdirAll("build", 0755); err != nil {
			return fmt.Errorf("failed to create build directory: %w", err)
		}
		// Write a placeholder comment - slasm goes directly to executable
		placeholder := []byte("// slasm object placeholder - direct compilation to executable\n")
		if err := os.WriteFile(opts.ObjectPath, placeholder, 0644); err != nil {
			return fmt.Errorf("failed to write object file placeholder: %w", err)
		}
		a.Logger.Printf("Wrote object placeholder to %s\n", opts.ObjectPath)
	}

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
	a.Logger.Printf("Symbol table (before adjustment):\n")
	for name, sym := range symbolTable.symbols {
		globalFlag := ""
		if sym.Global {
			globalFlag = " [GLOBAL]"
		}
		a.Logger.Printf("  %-15s: addr=0x%04x section=%v%s\n", name, sym.Address, sym.Section, globalFlag)
	}

	// Calculate base VM addresses for symbol adjustment
	// These must match the values used in WriteExecutable
	vmAddr := uint64(0x100000000) // Standard base for ARM64 executables
	headerSize := uint64(32)
	segmentCmdSize := uint64(72)
	sectionHeaderSize := uint64(80)
	pagezeroSize := uint64(72)
	linkeditSegmentSize := uint64(72)
	dylinkerCmdSize := uint64(32) // Aligned size for /usr/lib/dyld
	dylibCmdSize := uint64(56)    // Aligned size for /usr/lib/libSystem.B.dylib
	entryPointCmdSize := uint64(24)
	uuidCmdSize := uint64(24)
	buildVersionCmdSize := uint64(32)
	sourceVersionCmdSize := uint64(16)
	chainedFixupsCmdSize := uint64(16)
	exportsTrieCmdSize := uint64(16)
	symtabCmdSize := uint64(24)
	dysymtabCmdSize := uint64(80)
	functionStartsCmdSize := uint64(16)
	dataInCodeCmdSize := uint64(16)
	codeSignatureCmdSize := uint64(16)

	// Check if we have data
	hasData := false
	for _, section := range program.Sections {
		if section.Type == SectionData && len(section.Items) > 0 {
			hasData = true
			break
		}
	}

	// Calculate load commands size
	loadCmdsSize := pagezeroSize + segmentCmdSize + sectionHeaderSize + linkeditSegmentSize +
		dylinkerCmdSize + dylibCmdSize + entryPointCmdSize + uuidCmdSize +
		buildVersionCmdSize + sourceVersionCmdSize +
		chainedFixupsCmdSize + exportsTrieCmdSize +
		symtabCmdSize + dysymtabCmdSize +
		functionStartsCmdSize + dataInCodeCmdSize +
		codeSignatureCmdSize

	if hasData {
		dataSegmentCmdSize := uint64(72)
		dataSectionSize := uint64(80)
		loadCmdsSize += dataSegmentCmdSize + dataSectionSize
	}

	// Calculate code offset and base addresses
	codeOffset := ((headerSize + loadCmdsSize + 7) / 8) * 8 // Align to 8 bytes
	textBase := vmAddr + codeOffset

	// Calculate text segment size for data base calculation
	textSegmentFileSize := uint64(0x4000) // 16KB minimum

	// Calculate data base address
	dataBase := vmAddr + textSegmentFileSize // Data comes after TEXT segment

	// Adjust symbol addresses to actual VM addresses
	symbolTable.AdjustAddresses(textBase, dataBase)

	a.Logger.Printf("\nSymbol table (after adjustment):\n")
	for name, sym := range symbolTable.symbols {
		globalFlag := ""
		if sym.Global {
			globalFlag = " [GLOBAL]"
		}
		a.Logger.Printf("  %-15s: addr=0x%08x section=%v%s\n", name, sym.Address, sym.Section, globalFlag)
	}
	a.Logger.Printf("\n")

	// Step 4: Encode instructions to machine code
	a.Logger.Section("STEP 4: INSTRUCTION ENCODING")


	encoder := NewEncoder(layout.GetSymbolTable(), layout.GetConstants())
	var codeBytes []byte
	var dataBytes []byte
	instructionCount := 0
	dataItemCount := 0

	for _, section := range program.Sections {
		if section.Type == SectionText {
			for _, item := range section.Items {
				switch v := item.(type) {
				case *Instruction:
					relativeAddr := uint64(len(codeBytes))
					// Use absolute VM address for encoding (needed for branch offset calculations
					// since symbol addresses have been adjusted to absolute)
					currentAddr := textBase + relativeAddr
					machineCode, err := encoder.Encode(v, currentAddr)
					if err != nil {
						return fmt.Errorf("encoding error for instruction '%s': %w", v.Mnemonic, err)
					}

					// Format operands for display
					operands := ""
					for k, op := range v.Operands {
						if k > 0 {
							operands += ", "
						}
						operands += op.Value
					}

					a.Logger.Printf("  [0x%04x] %-20s -> %02x %02x %02x %02x (0x%08x)\n",
						currentAddr,
						fmt.Sprintf("%s %s", v.Mnemonic, operands),
						machineCode[0], machineCode[1], machineCode[2], machineCode[3],
						uint32(machineCode[0]) | uint32(machineCode[1])<<8 | uint32(machineCode[2])<<16 | uint32(machineCode[3])<<24)

					codeBytes = append(codeBytes, machineCode...)
					instructionCount++

				case *Directive:
					// Handle alignment directives by emitting NOP padding
					if v.Name == "align" && len(v.Args) > 0 {
						alignValue := parseAlignment(v.Args[0])
						if alignValue > 0 {
							alignment := uint64(1 << alignValue) // 2^n
							relativeAddr := uint64(len(codeBytes))
							padding := alignmentPadding(relativeAddr, alignment)
							if padding > 0 {
								// Emit NOPs (0x1f2003d5 = ARM64 NOP)
								nopCount := padding / 4
								a.Logger.Printf("  [0x%04x] .align %d -> %d NOP(s)\n",
									textBase+relativeAddr, alignValue, nopCount)
								for i := uint64(0); i < nopCount; i++ {
									codeBytes = append(codeBytes, 0x1f, 0x20, 0x03, 0xd5) // NOP in little endian
								}
							}
						}
					}
				}
			}
		} else if section.Type == SectionData {
			for _, item := range section.Items {
				if data, ok := item.(*DataDeclaration); ok {
					currentAddr := uint64(len(dataBytes))
					bytes, err := encoder.EncodeData(data)
					if err != nil {
						return fmt.Errorf("encoding error for data '.%s': %w", data.Type, err)
					}

					a.Logger.Printf("  [0x%04x] .%-10s %-20s -> %d bytes\n",
						currentAddr, data.Type, truncateValue(data.Value, 20), len(bytes))

					dataBytes = append(dataBytes, bytes...)
					dataItemCount++
				}
			}
		}
	}

	a.Logger.Printf("\nEncoded %d instructions (%d bytes)\n", instructionCount, len(codeBytes))
	if dataItemCount > 0 {
		a.Logger.Printf("Encoded %d data items (%d bytes)\n", dataItemCount, len(dataBytes))
	}
	a.Logger.Printf("Complete machine code: %x\n\n", codeBytes)

	// Step 5: Generate Mach-O executable
	a.Logger.Section("STEP 5: MACH-O GENERATION")


	writer := NewMachOWriter(a.Arch, a.Logger)
	err = writer.WriteExecutable(opts.OutputPath, codeBytes, dataBytes, layout.GetSymbolTable(), a.EntryPoint)
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
	a.Logger.Printf("Code signature: embedded during Mach-O generation\n")

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

// truncateValue truncates a string to maxLen characters, adding "..." if truncated
func truncateValue(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

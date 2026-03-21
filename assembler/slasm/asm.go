package slasm

import (
	"fmt"
	"os"

	"github.com/seanrogers2657/slang/assembler"
	"github.com/seanrogers2657/slang/internal/timing"
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
	// Timer for tracking assembly stages (optional)
	Timer *timing.Timer
}

// New creates a new NativeAssembler with default settings
func New() *NativeAssembler {
	return &NativeAssembler{
		Arch:       "arm64",
		EntryPoint: "_start",
		SystemLibs: true,
		Logger:     NewDefaultLogger(false), // Disabled by default, enable with --verbose
		Timer:      timing.NewTimer(),
	}
}

// TimingSummary returns a formatted string with timing information for the assembly stages
func (a *NativeAssembler) TimingSummary() string {
	return a.Timer.SummaryWithTitle("Assembly Summary")
}

// Assemble converts an assembly file (.s) to an object file (.o)
// This is the native implementation that parses and encodes ARM64 assembly
func (a *NativeAssembler) Assemble(inputPath, outputPath string) error {
	a.Logger.Header("========== SLASM ASSEMBLER - OBJECT FILE GENERATION ==========")

	// Step 1: Read assembly file
	a.Logger.Section("STEP 1: READ SOURCE")
	sourceBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read assembly file: %w", err)
	}
	assembly := string(sourceBytes)
	a.Logger.Printf("Read %d bytes from %s\n", len(sourceBytes), inputPath)

	// Step 2: Lex the assembly source
	a.Logger.Section("STEP 2: LEXER")
	lexer := NewLexer(assembly)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("lexer error: %w", err)
	}
	a.Logger.Printf("Lexer produced %d tokens\n", len(tokens))

	// Step 3: Parse tokens into IR
	a.Logger.Section("STEP 3: PARSER")
	parser := NewParser(tokens)
	program, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parser error: %w", err)
	}
	a.Logger.Printf("Parser produced %d section(s)\n", len(program.Sections))

	// Step 4: Calculate layout and build symbol table
	a.Logger.Section("STEP 4: LAYOUT & SYMBOL TABLE")
	layout := NewLayout(program)
	err = layout.Calculate()
	if err != nil {
		return fmt.Errorf("layout error: %w", err)
	}

	symbolTable := layout.GetSymbolTable()
	a.Logger.Printf("Symbol table contains %d symbols\n", symbolTable.Count())

	// For object files, addresses are relative to section start (0-based)
	// No need to adjust addresses like we do for executables

	// Step 5: Encode instructions to machine code
	a.Logger.Section("STEP 5: INSTRUCTION ENCODING")
	encoder := NewEncoder(layout.GetSymbolTable(), layout.GetConstants())
	var codeBytes []byte
	var dataBytes []byte
	var textRelocations []TextRelocation
	instructionCount := 0
	dataItemCount := 0

	for _, section := range program.Sections {
		if section.Type == SectionText {
			for _, item := range section.Items {
				switch v := item.(type) {
				case *Instruction:
					// For object files, use section-relative addresses
					currentAddr := uint64(len(codeBytes))
					result, err := encoder.EncodeWithRelocations(v, currentAddr)
					if err != nil {
						return fmt.Errorf("encoding error for instruction '%s': %w", v.Mnemonic, err)
					}
					codeBytes = append(codeBytes, result.Bytes...)
					if result.Relocation != nil {
						textRelocations = append(textRelocations, *result.Relocation)
					}
					instructionCount++

				case *Directive:
					if v.Name == "align" && len(v.Args) > 0 {
						alignValue, err := parseAlignment(v.Args[0])
						if err != nil {
							return fmt.Errorf("line %d: %w", v.Line, err)
						}
						if alignValue > 0 {
							alignment := uint64(1 << alignValue)
							relativeAddr := uint64(len(codeBytes))
							padding := alignmentPadding(relativeAddr, alignment)
							if padding > 0 {
								for i := uint64(0); i < padding/4; i++ {
									codeBytes = append(codeBytes, EncodeLittleEndian(ARM64_NOP)...) // NOP
								}
							}
						}
					}
				}
			}
		} else if section.Type == SectionData {
			for _, item := range section.Items {
				switch v := item.(type) {
				case *DataDeclaration:
					currentAddr := uint64(len(dataBytes))
					bytes, _, err := encoder.EncodeDataWithRelocations(v, currentAddr)
					if err != nil {
						return fmt.Errorf("encoding error for data '.%s': %w", v.Type, err)
					}
					dataBytes = append(dataBytes, bytes...)
					dataItemCount++

				case *Directive:
					if v.Name == "align" && len(v.Args) > 0 {
						alignValue, err := parseAlignment(v.Args[0])
						if err != nil {
							return fmt.Errorf("line %d: %w", v.Line, err)
						}
						if alignValue > 0 {
							alignment := uint64(1 << alignValue)
							currentAddr := uint64(len(dataBytes))
							padding := alignmentPadding(currentAddr, alignment)
							for i := uint64(0); i < padding; i++ {
								dataBytes = append(dataBytes, 0)
							}
						}
					}
				}
			}
		}
	}

	a.Logger.Printf("Encoded %d instructions (%d bytes)\n", instructionCount, len(codeBytes))
	if dataItemCount > 0 {
		a.Logger.Printf("Encoded %d data items (%d bytes)\n", dataItemCount, len(dataBytes))
	}

	// Step 6: Generate Mach-O object file
	a.Logger.Section("STEP 6: MACH-O OBJECT FILE GENERATION")
	writer := NewMachOWriter(a.Arch, a.Logger)
	err = writer.WriteObjectFile(outputPath, codeBytes, dataBytes, layout.GetSymbolTable(), textRelocations)
	if err != nil {
		return fmt.Errorf("object file generation error: %w", err)
	}

	a.Logger.Printf("Generated object file: %s\n", outputPath)

	// Summary
	a.Logger.Printf("\n========== ASSEMBLE SUMMARY ==========\n")
	a.Logger.Printf("Input file:        %s\n", inputPath)
	a.Logger.Printf("Output file:       %s\n", outputPath)
	a.Logger.Printf("Architecture:      %s\n", a.Arch)
	a.Logger.Printf("Instructions:      %d\n", instructionCount)
	a.Logger.Printf("Code size:         %d bytes\n", len(codeBytes))
	a.Logger.Printf("Data size:         %d bytes\n", len(dataBytes))
	a.Logger.Printf("Symbols:           %d\n", symbolTable.Count())
	a.Logger.Printf("=======================================\n\n")

	return nil
}

// Link creates an executable from object files
// This is the native implementation that links object files without using ld
func (a *NativeAssembler) Link(objectFiles []string, outputPath string) error {
	a.Logger.Header("========== SLASM LINKER ==========")

	linker := NewLinker(a.Logger)
	err := linker.Link(objectFiles, outputPath, a.EntryPoint)
	if err != nil {
		return fmt.Errorf("linker error: %w", err)
	}

	a.Logger.Printf("Successfully linked %d object file(s) -> %s\n", len(objectFiles), outputPath)
	return nil
}

// Build performs the complete build process from assembly string to executable
func (a *NativeAssembler) Build(assembly string, opts assembler.BuildOptions) error {
	a.Logger.Header("========== SLASM ASSEMBLER - BUILD PIPELINE ==========")

	// Reset timer for this build
	a.Timer = timing.NewTimer()

	// Write intermediate files asynchronously (these are just for debugging/inspection)
	if opts.AssemblyPath != "" || (opts.ObjectPath != "" && opts.KeepIntermediates) {
		go func() {
			// Ensure directory exists
			if err := os.MkdirAll("build", 0755); err != nil {
				a.Logger.Printf("Warning: failed to create build directory: %v\n", err)
				return
			}

			if opts.AssemblyPath != "" {
				if err := os.WriteFile(opts.AssemblyPath, []byte(assembly), 0644); err != nil {
					a.Logger.Printf("Warning: failed to write assembly file: %v\n", err)
				}
			}

			// Create placeholder object file (slasm doesn't use separate object files)
			if opts.ObjectPath != "" && opts.KeepIntermediates {
				placeholder := []byte("// slasm object placeholder - direct compilation to executable\n")
				if err := os.WriteFile(opts.ObjectPath, placeholder, 0644); err != nil {
					a.Logger.Printf("Warning: failed to write object file placeholder: %v\n", err)
				}
			}
		}()
	}

	// Step 1: Lex the assembly source
	a.Timer.Start("Lexer")
	a.Logger.Section("STEP 1: LEXER")
	a.Logger.Printf("Input assembly:\n%s\n\n", assembly)

	lexer := NewLexer(assembly)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("lexer error: %w", err)
	}
	a.Timer.End()

	a.Logger.Printf("Lexer produced %d tokens:\n", len(tokens))
	for i, token := range tokens {
		a.Logger.Printf("  [%3d] %-15v: %s\n", i, token.Type, token.Value)
	}
	a.Logger.Println()

	// Step 2: Parse tokens into IR
	a.Timer.Start("Parser")
	a.Logger.Section("STEP 2: PARSER")

	parser := NewParser(tokens)
	program, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parser error: %w", err)
	}
	a.Timer.End()

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
	a.Timer.Start("Layout & Symbols")
	a.Logger.Section("STEP 3: LAYOUT & SYMBOL TABLE")

	layout := NewLayout(program)
	err = layout.Calculate()
	if err != nil {
		return fmt.Errorf("layout error: %w", err)
	}
	a.Timer.End()

	symbolTable := layout.GetSymbolTable()
	a.Logger.Printf("Symbol table (before adjustment):\n")
	symbolTable.ForEach(func(name string, sym *Symbol) {
		globalFlag := ""
		if sym.Global {
			globalFlag = " [GLOBAL]"
		}
		a.Logger.Printf("  %-15s: addr=0x%04x section=%v%s\n", name, sym.Address, sym.Section, globalFlag)
	})

	// Calculate base VM addresses for symbol adjustment
	// These must match the values used in WriteExecutable (using constants from macho.go)
	vmAddr := uint64(VMBaseAddress)

	// Calculate dylinker and dylib command sizes (path + alignment)
	dylinkerCmdSize := uint64(DylinkerCmdBaseSize + len(DylinkerPath) + 1)
	dylinkerCmdSize = (dylinkerCmdSize + 7) &^ 7 // Align to 8 bytes
	dylibCmdSize := uint64(DylibCmdBaseSize + len(LibSystemPath) + 1)
	dylibCmdSize = (dylibCmdSize + 7) &^ 7

	// Check if we have data
	hasData := false
	for _, section := range program.Sections {
		if section.Type == SectionData && len(section.Items) > 0 {
			hasData = true
			break
		}
	}

	// Calculate load commands size using constants from macho.go
	loadCmdsSize := uint64(SegmentCommand64Size) + // __PAGEZERO
		uint64(SegmentCommand64Size+Section64Size) + // __TEXT + __text section
		uint64(SegmentCommand64Size) + // __LINKEDIT
		dylinkerCmdSize +
		dylibCmdSize +
		uint64(EntryPointCmdSize) +
		uint64(UUIDCmdSize) +
		uint64(BuildVersionCmdSize) +
		uint64(SourceVersionCmdSize) +
		uint64(LinkeditDataCmdSize) + // chained fixups
		uint64(LinkeditDataCmdSize) + // exports trie
		uint64(SymtabCmdSize) +
		uint64(DysymtabCmdSize) +
		uint64(LinkeditDataCmdSize) + // function starts
		uint64(LinkeditDataCmdSize) + // data in code
		uint64(CodeSignatureCmdSize)

	if hasData {
		loadCmdsSize += uint64(SegmentCommand64Size + Section64Size) // __DATA + __data section
	}

	// Calculate code offset and base addresses
	codeOffset := ((uint64(MachHeader64Size) + loadCmdsSize + 7) / 8) * 8 // Align to 8 bytes
	textBase := vmAddr + codeOffset

	// Calculate text segment size for data base calculation
	textSegmentFileSize := uint64(MinSegmentFileSize) // 16KB minimum

	// Calculate data base address
	dataBase := vmAddr + textSegmentFileSize // Data comes after TEXT segment

	// Adjust symbol addresses to actual VM addresses
	symbolTable.AdjustAddresses(textBase, dataBase)

	a.Logger.Printf("\nSymbol table (after adjustment):\n")
	symbolTable.ForEach(func(name string, sym *Symbol) {
		globalFlag := ""
		if sym.Global {
			globalFlag = " [GLOBAL]"
		}
		a.Logger.Printf("  %-15s: addr=0x%08x section=%v%s\n", name, sym.Address, sym.Section, globalFlag)
	})
	a.Logger.Printf("\n")

	// Step 4: Encode instructions to machine code
	a.Timer.Start("Encoding")
	a.Logger.Section("STEP 4: INSTRUCTION ENCODING")

	encoder := NewEncoder(layout.GetSymbolTable(), layout.GetConstants())
	var codeBytes []byte
	var dataBytes []byte
	var dataRelocations []DataRelocation
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
						uint32(machineCode[0])|uint32(machineCode[1])<<8|uint32(machineCode[2])<<16|uint32(machineCode[3])<<24)

					codeBytes = append(codeBytes, machineCode...)
					instructionCount++

				case *Directive:
					// Handle alignment directives by emitting NOP padding
					if v.Name == "align" && len(v.Args) > 0 {
						alignValue, err := parseAlignment(v.Args[0])
						if err != nil {
							return fmt.Errorf("line %d: %w", v.Line, err)
						}
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
									codeBytes = append(codeBytes, EncodeLittleEndian(ARM64_NOP)...) // NOP
								}
							}
						}
					}
				}
			}
		} else if section.Type == SectionData {
			for _, item := range section.Items {
				switch v := item.(type) {
				case *DataDeclaration:
					currentAddr := uint64(len(dataBytes))
					bytes, relocs, err := encoder.EncodeDataWithRelocations(v, currentAddr)
					if err != nil {
						return fmt.Errorf("encoding error for data '.%s': %w", v.Type, err)
					}

					a.Logger.Printf("  [0x%04x] .%-10s %-20s -> %d bytes\n",
						currentAddr, v.Type, truncateValue(v.Value, 20), len(bytes))

					dataBytes = append(dataBytes, bytes...)
					dataRelocations = append(dataRelocations, relocs...)
					dataItemCount++

				case *Directive:
					// Handle alignment directives by emitting zero padding
					if v.Name == "align" && len(v.Args) > 0 {
						alignValue, err := parseAlignment(v.Args[0])
						if err != nil {
							return fmt.Errorf("line %d: %w", v.Line, err)
						}
						if alignValue > 0 {
							alignment := uint64(1 << alignValue) // 2^n
							currentAddr := uint64(len(dataBytes))
							padding := alignmentPadding(currentAddr, alignment)
							if padding > 0 {
								a.Logger.Printf("  [0x%04x] .align %d -> %d bytes padding\n",
									currentAddr, alignValue, padding)
								// Emit zero padding for data section alignment
								for i := uint64(0); i < padding; i++ {
									dataBytes = append(dataBytes, 0)
								}
							}
						}
					}
				}
			}
		}
	}

	a.Logger.Printf("\nEncoded %d instructions (%d bytes)\n", instructionCount, len(codeBytes))
	if dataItemCount > 0 {
		a.Logger.Printf("Encoded %d data items (%d bytes)\n", dataItemCount, len(dataBytes))
	}
	a.Logger.Printf("Complete machine code: %x\n\n", codeBytes)
	a.Timer.End()

	// Step 5: Generate Mach-O executable
	a.Timer.Start("Mach-O Generation")
	a.Logger.Section("STEP 5: MACH-O GENERATION")

	writer := NewMachOWriter(a.Arch, a.Logger)
	err = writer.WriteExecutable(opts.OutputPath, codeBytes, dataBytes, dataRelocations, layout.GetSymbolTable(), a.EntryPoint)
	if err != nil {
		return fmt.Errorf("mach-o generation error: %w", err)
	}

	a.Logger.Printf("Generated Mach-O executable: %s\n", opts.OutputPath)
	a.Timer.End()

	// Step 6: Make the file executable
	a.Timer.Start("Set Permissions")
	a.Logger.Section("\nSTEP 6: FILE PERMISSIONS")

	err = makeExecutable(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("chmod error: %w", err)
	}

	a.Logger.Printf("Set executable permissions (0755)\n")
	a.Logger.Printf("Code signature: embedded during Mach-O generation\n")
	a.Timer.End()

	// Summary
	a.Logger.Printf("\n========== BUILD SUMMARY ==========\n")
	a.Logger.Printf("Output file:       %s\n", opts.OutputPath)
	a.Logger.Printf("Architecture:      %s\n", a.Arch)
	a.Logger.Printf("Entry point:       %s\n", a.EntryPoint)
	a.Logger.Printf("Instructions:      %d\n", instructionCount)
	a.Logger.Printf("Code size:         %d bytes\n", len(codeBytes))
	a.Logger.Printf("Symbols:           %d\n", symbolTable.Count())
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

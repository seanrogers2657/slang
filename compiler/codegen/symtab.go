package codegen

import (
	"fmt"
	"strings"
)

// SymbolEntry represents a function in the symbol table for stack traces.
type SymbolEntry struct {
	Name      string // function name
	Filename  string // source file name
	StartLine int    // function start line number
	Label     string // assembly label for function start (e.g., "_main")
	EndLabel  string // assembly label for function end (e.g., "_main_end")
}

// LineEntry maps a code address (label) to a source line number.
// Used for precise line reporting in stack traces at call sites.
type LineEntry struct {
	Label string // assembly label (placed after bl instruction)
	Line  int    // source line number
}

// SymbolTable tracks all functions for generating runtime symbol table data.
type SymbolTable struct {
	Entries     []SymbolEntry
	LineEntries []LineEntry
	Filename    string // current source filename
	lineCounter int    // for generating unique line labels
}

// NewSymbolTable creates a new symbol table.
func NewSymbolTable(filename string) *SymbolTable {
	return &SymbolTable{
		Filename: filename,
	}
}

// AddFunction registers a function in the symbol table.
func (s *SymbolTable) AddFunction(name string, startLine int) {
	s.Entries = append(s.Entries, SymbolEntry{
		Name:      name,
		Filename:  s.Filename,
		StartLine: startLine,
		Label:     fmt.Sprintf("_%s", name),
		EndLabel:  fmt.Sprintf("_%s_end", name),
	})
}

// AddLineEntry adds a line table entry mapping a label to a source line.
// Returns the label that should be emitted after the bl instruction.
func (s *SymbolTable) AddLineEntry(line int) string {
	label := fmt.Sprintf("_line_%d", s.lineCounter)
	s.lineCounter++
	s.LineEntries = append(s.LineEntries, LineEntry{
		Label: label,
		Line:  line,
	})
	return label
}

// GenerateCallWithLine generates a function call (bl) with line tracking.
// Returns assembly code that includes the bl instruction followed by a label
// for stack trace line number lookup.
func (s *SymbolTable) GenerateCallWithLine(funcName string, line int) string {
	label := s.AddLineEntry(line)
	return fmt.Sprintf("    bl _%s\n%s:\n", funcName, label)
}

// GenerateDataSection produces the .data section entries for the symbol table.
// This includes function address ranges, names, filenames, and line numbers.
//
// Symbol table entry format (56 bytes each):
//   - .quad start_address      (8 bytes) - function start label
//   - .quad end_address        (8 bytes) - function end label
//   - .quad name_ptr           (8 bytes) - pointer to function name string
//   - .quad name_len           (8 bytes) - function name length
//   - .quad file_ptr           (8 bytes) - pointer to filename string
//   - .quad file_len           (8 bytes) - filename length
//   - .quad line_number        (8 bytes) - function start line
func (s *SymbolTable) GenerateDataSection() string {
	if len(s.Entries) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("// Symbol table for stack traces\n")
	b.WriteString(".data\n")
	b.WriteString(".align 3\n")

	// Generate string labels for names and filenames
	// First, collect unique filenames
	filenames := make(map[string]string) // filename -> label
	filenameCount := 0
	for _, entry := range s.Entries {
		if _, exists := filenames[entry.Filename]; !exists {
			label := fmt.Sprintf("_symtab_file_%d", filenameCount)
			filenames[entry.Filename] = label
			filenameCount++
		}
	}

	// Generate filename strings
	for filename, label := range filenames {
		escapedFilename := EscapeStringForAsm(filename)
		b.WriteString(fmt.Sprintf("%s: .asciz \"%s\"\n", label, escapedFilename))
	}

	// Generate function name strings
	for i, entry := range s.Entries {
		b.WriteString(fmt.Sprintf("_symtab_name_%d: .asciz \"%s\"\n", i, entry.Name))
	}

	b.WriteString("\n")

	// Generate the symbol table reference (for ASLR slide computation)
	b.WriteString(".align 3\n")
	b.WriteString("_slang_symtab_ref:\n")
	b.WriteString("    .quad _slang_symtab\n\n")

	// Generate the symbol table
	b.WriteString(".align 3\n")
	b.WriteString(".global _slang_symtab\n")
	b.WriteString("_slang_symtab:\n")

	for i, entry := range s.Entries {
		fileLabel := filenames[entry.Filename]
		nameLabel := fmt.Sprintf("_symtab_name_%d", i)

		b.WriteString(fmt.Sprintf("    // %s at %s:%d\n", entry.Name, entry.Filename, entry.StartLine))
		b.WriteString(fmt.Sprintf("    .quad %s\n", entry.Label))         // start address
		b.WriteString(fmt.Sprintf("    .quad %s\n", entry.EndLabel))      // end address
		b.WriteString(fmt.Sprintf("    .quad %s\n", nameLabel))           // name pointer
		b.WriteString(fmt.Sprintf("    .quad %d\n", len(entry.Name)))     // name length
		b.WriteString(fmt.Sprintf("    .quad %s\n", fileLabel))           // file pointer
		b.WriteString(fmt.Sprintf("    .quad %d\n", len(entry.Filename))) // file length
		b.WriteString(fmt.Sprintf("    .quad %d\n", entry.StartLine))     // line number
	}

	// Sentinel entry (null terminator)
	b.WriteString("    // sentinel\n")
	b.WriteString("    .quad 0\n")

	b.WriteString("\n")

	// Generate the line number table for precise call site reporting
	b.WriteString(s.generateLineTable())

	return b.String()
}

// generateLineTable produces the line number table section.
// This maps return addresses (labels after bl instructions) to source line numbers.
//
// Line table entry format (16 bytes each):
//   - .quad address       (8 bytes) - label address (return address after bl)
//   - .quad line_number   (8 bytes) - source line number
func (s *SymbolTable) generateLineTable() string {
	if len(s.LineEntries) == 0 {
		// Still emit an empty table with sentinel for the runtime
		var b strings.Builder
		b.WriteString("// Line number table for call site reporting\n")
		b.WriteString(".align 3\n")
		b.WriteString(".global _slang_linetab\n")
		b.WriteString("_slang_linetab:\n")
		b.WriteString("    // sentinel\n")
		b.WriteString("    .quad 0\n")
		b.WriteString("\n")
		return b.String()
	}

	var b strings.Builder

	b.WriteString("// Line number table for call site reporting\n")
	b.WriteString(".align 3\n")
	b.WriteString(".global _slang_linetab\n")
	b.WriteString("_slang_linetab:\n")

	for _, entry := range s.LineEntries {
		b.WriteString(fmt.Sprintf("    .quad %s    // line %d\n", entry.Label, entry.Line))
		b.WriteString(fmt.Sprintf("    .quad %d\n", entry.Line))
	}

	// Sentinel entry (null terminator)
	b.WriteString("    // sentinel\n")
	b.WriteString("    .quad 0\n")

	b.WriteString("\n")

	return b.String()
}

// GenerateFunctionEndLabel produces the end label for a function.
// This should be emitted right after the function body.
func GenerateFunctionEndLabel(name string) string {
	return fmt.Sprintf("_%s_end:\n", name)
}

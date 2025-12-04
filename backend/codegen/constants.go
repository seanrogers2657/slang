package codegen

// StackAlignment returns the required stack alignment for the target architecture.
// This delegates to the default emitter.
var StackAlignment = 16 // Will be replaced by defaultEmitter.StackAlignment() at init

func init() {
	// Set StackAlignment from the default emitter
	// This is done in init() because defaultEmitter is defined in asm_helpers.go
	StackAlignment = defaultEmitter.StackAlignment()
}

// File descriptors (platform-independent)
const (
	// Stdout is the file descriptor for standard output
	Stdout = 1
)

// ASCII character codes (platform-independent)
const (
	// ASCIIZero is the ASCII code for '0'
	ASCIIZero = 48
	// ASCIIMinus is the ASCII code for '-'
	ASCIIMinus = 45
	// ASCIINewline is the ASCII code for newline
	ASCIINewline = 10
)

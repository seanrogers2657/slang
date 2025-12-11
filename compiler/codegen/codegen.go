package codegen

// AsGenerator defines the interface for ARM64 assembly code generators.
// Implementations convert parsed programs into ARM64 assembly targeting macOS.
type AsGenerator interface {
	// Generate produces ARM64 assembly code as a string.
	// Returns an error if code generation fails.
	Generate() (string, error)
}

// intToStringFunctionText generates the int-to-string conversion routine.
func intToStringFunctionText() string {
	return defaultEmitter.IntToStringFunction()
}

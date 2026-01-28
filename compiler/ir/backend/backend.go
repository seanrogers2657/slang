// Package backend provides interfaces for IR code generation backends.
package backend

import (
	"github.com/seanrogers2657/slang/compiler/ir"
)

// Backend is the interface that all code generation backends must implement.
// A backend takes IR and produces target-specific assembly code.
type Backend interface {
	// Generate produces assembly code from an IR program.
	// Returns the generated assembly as a string and any errors encountered.
	Generate(prog *ir.Program) (string, error)

	// Name returns the name of this backend (e.g., "arm64", "x86_64").
	Name() string
}

// FunctionGenerator handles code generation for a single function.
// This interface allows backends to process functions independently,
// which is useful for parallel compilation.
type FunctionGenerator interface {
	// GenerateFunction produces assembly code for a single function.
	GenerateFunction(fn *ir.Function) (string, error)
}

// Config holds configuration options for code generation.
type Config struct {
	// OptLevel is the optimization level (0 = none, 1 = basic, 2 = full).
	OptLevel int

	// Debug enables generation of debug information.
	Debug bool

	// Filename is the source file name for debug info and error messages.
	Filename string

	// SourceLines are the original source lines for stack traces.
	SourceLines []string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		OptLevel: 0,
		Debug:    false,
	}
}

// RegisterInfo describes how a backend uses registers.
type RegisterInfo struct {
	// NumGPRegs is the number of general-purpose registers available.
	NumGPRegs int

	// NumFPRegs is the number of floating-point registers available.
	NumFPRegs int

	// CallerSaved lists registers that the caller must save.
	CallerSaved []int

	// CalleeSaved lists registers that the callee must save.
	CalleeSaved []int

	// ParamRegs lists registers used for passing parameters.
	ParamRegs []int

	// ReturnReg is the register used for return values.
	ReturnReg int
}

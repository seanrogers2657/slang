// Package arch defines the architecture abstraction layer for code generation.
// It provides interfaces that allow the code generator to emit instructions
// for different target architectures (ARM64, AMD64, etc.).
package arch

import (
	"fmt"
	"strings"
)

// Emitter defines the interface for architecture-specific code emission.
// Implementations provide the actual assembly instructions for each target.
type Emitter interface {
	// Program structure
	EmitDataSection(hasPrint bool) string
	EmitProgramEntry() string
	EmitFunctionLabel(name string) string

	// Function structure
	EmitFunctionPrologue(stackSize int) string
	EmitFunctionEpilogue(hasLocals bool) string
	EmitReturnEpilogue() string

	// Integer operations - operands in LeftReg/RightReg, result in ResultReg
	EmitIntOp(op string, signed bool) (string, error)

	// Float operations - operands in float regs, result in FloatResultReg
	EmitFloatOp(op string) (string, error)

	// Register operations
	EmitMoveReg(dst, src string) string
	EmitMoveImm(reg, value string) string

	// Stack operations (offset is positive, relative to frame pointer)
	EmitStoreToStack(reg string, offset int) string
	EmitLoadFromStack(reg string, offset int) string
	EmitPushToStack(reg string) string
	EmitPopFromStack(reg string) string

	// Address loading
	EmitLoadAddress(reg, label string) string

	// Control flow
	EmitBranchLink(label string) string

	// Syscalls
	EmitExitSyscall() string
	EmitWriteSyscall(bufferReg, lengthReg string) string
	EmitNewline() string

	// Print support
	EmitPrintInt() string
	IntToStringFunction() string

	// ABI constants
	StackAlignment() int
	ResultReg() string      // Register for integer expression results (e.g., "x2")
	FloatResultReg() string // Register for float expression results (e.g., "d0")
	LeftReg() string        // Register for left operand (e.g., "x0")
	RightReg() string       // Register for right operand (e.g., "x1")
	FloatLeftReg() string   // Register for float left operand (e.g., "d1")
	FloatRightReg() string  // Register for float right operand (e.g., "d0")
	ArgRegs() []string      // Registers for function arguments
	FramePointer() string   // Frame pointer register (e.g., "x29")
	LinkReg() string        // Link register (e.g., "x30")
}

// Target represents a compilation target (architecture + platform).
type Target struct {
	Arch     Architecture
	Platform Platform
}

// Architecture represents the target CPU architecture.
type Architecture string

const (
	ArchARM64 Architecture = "arm64"
	ArchAMD64 Architecture = "amd64"
)

// Platform represents the target operating system.
type Platform string

const (
	PlatformDarwin Platform = "darwin"
	PlatformLinux  Platform = "linux"
)

// EmitterBuilder is a helper for emitters that use strings.Builder.
type EmitterBuilder struct {
	Builder *strings.Builder
}

// NewEmitterBuilder creates a new EmitterBuilder with an initialized Builder.
func NewEmitterBuilder() *EmitterBuilder {
	return &EmitterBuilder{Builder: &strings.Builder{}}
}

// String returns the accumulated string and resets the builder.
func (eb *EmitterBuilder) String() string {
	s := eb.Builder.String()
	eb.Builder.Reset()
	return s
}

// Write writes a string to the builder.
func (eb *EmitterBuilder) Write(s string) {
	eb.Builder.WriteString(s)
}

// Writef writes a formatted string to the builder.
func (eb *EmitterBuilder) Writef(format string, args ...interface{}) {
	eb.Builder.WriteString(fmt.Sprintf(format, args...))
}

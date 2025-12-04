package codegen

import (
	"strings"

	"github.com/seanrogers2657/slang/backend/arch"
	"github.com/seanrogers2657/slang/backend/arch/arm64"
)

// defaultEmitter is the default architecture emitter (ARM64 for now).
// This can be replaced with a different emitter to target other architectures.
var defaultEmitter arch.Emitter = arm64.New()

// EmitFunctionPrologue generates the function prologue using the default emitter.
func EmitFunctionPrologue(builder *strings.Builder, stackSize int) {
	builder.WriteString(defaultEmitter.EmitFunctionPrologue(stackSize))
}

// EmitFunctionEpilogue generates the function epilogue using the default emitter.
func EmitFunctionEpilogue(builder *strings.Builder, hasLocals bool) {
	builder.WriteString(defaultEmitter.EmitFunctionEpilogue(hasLocals))
}

// EmitReturnEpilogue generates the epilogue for a return statement.
func EmitReturnEpilogue(builder *strings.Builder) {
	builder.WriteString(defaultEmitter.EmitReturnEpilogue())
}

// EmitExitSyscall generates the exit syscall using the default emitter.
func EmitExitSyscall(builder *strings.Builder) {
	builder.WriteString(defaultEmitter.EmitExitSyscall())
}

// EmitWriteSyscall generates assembly code to write data to stdout.
func EmitWriteSyscall(builder *strings.Builder, bufferReg, lengthReg string) {
	builder.WriteString(defaultEmitter.EmitWriteSyscall(bufferReg, lengthReg))
}

// EmitNewline generates assembly code to write a newline to stdout.
func EmitNewline(builder *strings.Builder) {
	builder.WriteString(defaultEmitter.EmitNewline())
}

// EmitPrintInt generates code to print an integer value.
func EmitPrintInt(builder *strings.Builder) {
	builder.WriteString(defaultEmitter.EmitPrintInt())
}

// EmitDataSection generates the .data section header with optional print support.
func EmitDataSection(builder *strings.Builder, hasPrint bool) {
	builder.WriteString(defaultEmitter.EmitDataSection(hasPrint))
}

// EmitProgramEntry generates the _start entry point that calls main.
func EmitProgramEntry(builder *strings.Builder) {
	builder.WriteString(defaultEmitter.EmitProgramEntry())
}

// EmitFunctionLabel generates a function label with proper alignment.
func EmitFunctionLabel(builder *strings.Builder, name string) {
	builder.WriteString(defaultEmitter.EmitFunctionLabel(name))
}

// EmitStoreToStack stores a register value to the stack relative to frame pointer.
func EmitStoreToStack(builder *strings.Builder, reg string, offset int) {
	builder.WriteString(defaultEmitter.EmitStoreToStack(reg, offset))
}

// EmitLoadFromStack loads a value from the stack relative to frame pointer.
func EmitLoadFromStack(builder *strings.Builder, reg string, offset int) {
	builder.WriteString(defaultEmitter.EmitLoadFromStack(reg, offset))
}

// EmitPushToStack pushes a register value onto the stack (pre-decrement).
func EmitPushToStack(builder *strings.Builder, reg string) {
	builder.WriteString(defaultEmitter.EmitPushToStack(reg))
}

// EmitPopFromStack pops a value from the stack into a register (post-increment).
func EmitPopFromStack(builder *strings.Builder, reg string) {
	builder.WriteString(defaultEmitter.EmitPopFromStack(reg))
}

// EmitMoveReg generates a register-to-register move.
func EmitMoveReg(builder *strings.Builder, dst, src string) {
	builder.WriteString(defaultEmitter.EmitMoveReg(dst, src))
}

// EmitMoveImm generates an immediate value load.
func EmitMoveImm(builder *strings.Builder, reg, value string) {
	builder.WriteString(defaultEmitter.EmitMoveImm(reg, value))
}

// EmitBranchLink generates a branch-and-link (function call).
func EmitBranchLink(builder *strings.Builder, label string) {
	builder.WriteString(defaultEmitter.EmitBranchLink(label))
}

// EmitLoadAddress generates code to load a label address.
func EmitLoadAddress(builder *strings.Builder, reg, label string) {
	builder.WriteString(defaultEmitter.EmitLoadAddress(reg, label))
}

// EscapeStringForAsm escapes special characters for assembly string literals.
func EscapeStringForAsm(s string) string {
	var result strings.Builder
	for _, c := range s {
		switch c {
		case '\n':
			result.WriteString("\\n")
		case '\t':
			result.WriteString("\\t")
		case '\r':
			result.WriteString("\\r")
		case '\\':
			result.WriteString("\\\\")
		case '"':
			result.WriteString("\\\"")
		default:
			result.WriteRune(c)
		}
	}
	return result.String()
}

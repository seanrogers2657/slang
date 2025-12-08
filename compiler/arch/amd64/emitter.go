// Package amd64 provides AMD64 (x86-64) code emission.
// TODO: This is a placeholder for future AMD64 support.
package amd64

import (
	"fmt"

	"github.com/seanrogers2657/slang/compiler/arch"
)

// Emitter implements arch.Emitter for AMD64.
// TODO: Implement all methods for AMD64 code generation.
type Emitter struct {
	// TODO: Add platform field for Linux vs macOS syscall differences
}

// New creates a new AMD64 emitter.
func New() *Emitter {
	return &Emitter{}
}

// Ensure Emitter implements arch.Emitter at compile time.
var _ arch.Emitter = (*Emitter)(nil)

// ABI constants for AMD64
// TODO: Verify these for System V AMD64 ABI
const (
	stackAlignment = 16
	resultReg      = "rax"
	floatResultReg = "xmm0"
	leftReg        = "rdi"  // First integer arg
	rightReg       = "rsi"  // Second integer arg
	floatLeftReg   = "xmm0" // First float arg
	floatRightReg  = "xmm1" // Second float arg
	framePointer   = "rbp"
	stackPointer   = "rsp"
)

// StackAlignment returns the required stack alignment for AMD64 (16 bytes).
func (e *Emitter) StackAlignment() int { return stackAlignment }

// ResultReg returns the register for integer expression results.
func (e *Emitter) ResultReg() string { return resultReg }

// FloatResultReg returns the register for float expression results.
func (e *Emitter) FloatResultReg() string { return floatResultReg }

// LeftReg returns the register for left operand.
func (e *Emitter) LeftReg() string { return leftReg }

// RightReg returns the register for right operand.
func (e *Emitter) RightReg() string { return rightReg }

// FloatLeftReg returns the register for float left operand.
func (e *Emitter) FloatLeftReg() string { return floatLeftReg }

// FloatRightReg returns the register for float right operand.
func (e *Emitter) FloatRightReg() string { return floatRightReg }

// FramePointer returns the frame pointer register.
func (e *Emitter) FramePointer() string { return framePointer }

// LinkReg returns the link register (AMD64 uses stack for return address).
func (e *Emitter) LinkReg() string { return "" } // AMD64 uses stack

// ArgRegs returns the registers used for function arguments (System V ABI).
func (e *Emitter) ArgRegs() []string {
	return []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
}

// EmitDataSection generates the .data section header.
// TODO: Implement for AMD64 (different syntax for gas vs nasm)
func (e *Emitter) EmitDataSection(hasPrint bool) string {
	panic("TODO: implement AMD64 EmitDataSection")
}

// EmitProgramEntry generates the _start entry point.
// TODO: Implement for AMD64
func (e *Emitter) EmitProgramEntry() string {
	panic("TODO: implement AMD64 EmitProgramEntry")
}

// EmitFunctionLabel generates a function label.
// TODO: Implement for AMD64
func (e *Emitter) EmitFunctionLabel(name string) string {
	panic("TODO: implement AMD64 EmitFunctionLabel")
}

// EmitFunctionPrologue generates the function prologue.
// TODO: Implement for AMD64 (push rbp; mov rbp, rsp; sub rsp, N)
func (e *Emitter) EmitFunctionPrologue(stackSize int) string {
	panic("TODO: implement AMD64 EmitFunctionPrologue")
}

// EmitFunctionEpilogue generates the function epilogue.
// TODO: Implement for AMD64 (mov rsp, rbp; pop rbp; ret)
func (e *Emitter) EmitFunctionEpilogue(hasLocals bool) string {
	panic("TODO: implement AMD64 EmitFunctionEpilogue")
}

// EmitReturnEpilogue generates the epilogue for a return statement.
// TODO: Implement for AMD64
func (e *Emitter) EmitReturnEpilogue() string {
	panic("TODO: implement AMD64 EmitReturnEpilogue")
}

// EmitIntOp generates AMD64 assembly for an integer binary operation.
// TODO: Implement for AMD64
// Note: AMD64 operations are typically two-operand (dest = dest op src)
// May need to use different register allocation strategy
func (e *Emitter) EmitIntOp(op string, signed bool) (string, error) {
	return "", fmt.Errorf("TODO: implement AMD64 EmitIntOp for %s", op)
}

// EmitFloatOp generates AMD64 assembly for a floating-point operation.
// TODO: Implement for AMD64 (SSE/AVX instructions)
func (e *Emitter) EmitFloatOp(op string) (string, error) {
	return "", fmt.Errorf("TODO: implement AMD64 EmitFloatOp for %s", op)
}

// EmitMoveReg generates a register-to-register move.
// TODO: Implement for AMD64 (mov dst, src)
func (e *Emitter) EmitMoveReg(dst, src string) string {
	panic("TODO: implement AMD64 EmitMoveReg")
}

// EmitMoveImm generates an immediate value load.
// TODO: Implement for AMD64 (mov reg, imm)
func (e *Emitter) EmitMoveImm(reg, value string) string {
	panic("TODO: implement AMD64 EmitMoveImm")
}

// EmitStoreToStack stores a register to the stack.
// TODO: Implement for AMD64 (mov [rbp-offset], reg)
func (e *Emitter) EmitStoreToStack(reg string, offset int) string {
	panic("TODO: implement AMD64 EmitStoreToStack")
}

// EmitLoadFromStack loads a value from the stack.
// TODO: Implement for AMD64 (mov reg, [rbp-offset])
func (e *Emitter) EmitLoadFromStack(reg string, offset int) string {
	panic("TODO: implement AMD64 EmitLoadFromStack")
}

// EmitPushToStack pushes a register onto the stack.
// TODO: Implement for AMD64 (push reg)
func (e *Emitter) EmitPushToStack(reg string) string {
	panic("TODO: implement AMD64 EmitPushToStack")
}

// EmitPopFromStack pops a value from the stack.
// TODO: Implement for AMD64 (pop reg)
func (e *Emitter) EmitPopFromStack(reg string) string {
	panic("TODO: implement AMD64 EmitPopFromStack")
}

// EmitLoadAddress generates code to load a label address.
// TODO: Implement for AMD64 (lea reg, [rip+label] for PIC)
func (e *Emitter) EmitLoadAddress(reg, label string) string {
	panic("TODO: implement AMD64 EmitLoadAddress")
}

// EmitBranchLink generates a function call.
// TODO: Implement for AMD64 (call label)
func (e *Emitter) EmitBranchLink(label string) string {
	panic("TODO: implement AMD64 EmitBranchLink")
}

// EmitExitSyscall generates the exit syscall.
// TODO: Implement for AMD64
// Linux: mov rax, 60; syscall
// macOS: mov rax, 0x2000001; syscall
func (e *Emitter) EmitExitSyscall() string {
	panic("TODO: implement AMD64 EmitExitSyscall")
}

// EmitWriteSyscall generates a write syscall.
// TODO: Implement for AMD64
// Linux: mov rax, 1; mov rdi, 1; syscall
// macOS: mov rax, 0x2000004; mov rdi, 1; syscall
func (e *Emitter) EmitWriteSyscall(bufferReg, lengthReg string) string {
	panic("TODO: implement AMD64 EmitWriteSyscall")
}

// EmitNewline generates code to write a newline.
// TODO: Implement for AMD64
func (e *Emitter) EmitNewline() string {
	panic("TODO: implement AMD64 EmitNewline")
}

// EmitPrintInt generates code to print an integer.
// TODO: Implement for AMD64
func (e *Emitter) EmitPrintInt() string {
	panic("TODO: implement AMD64 EmitPrintInt")
}

// IntToStringFunction returns the int-to-string conversion routine.
// TODO: Implement for AMD64
func (e *Emitter) IntToStringFunction() string {
	panic("TODO: implement AMD64 IntToStringFunction")
}

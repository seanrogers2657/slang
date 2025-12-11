// Package arm64 provides ARM64 (AArch64) code emission for macOS.
package arm64

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/seanrogers2657/slang/compiler/arch"
)

// Emitter implements arch.Emitter for ARM64 macOS.
type Emitter struct{}

// New creates a new ARM64 emitter.
func New() *Emitter {
	return &Emitter{}
}

// Ensure Emitter implements arch.Emitter.
var _ arch.Emitter = (*Emitter)(nil)

// ABI constants
const (
	stackAlignment = 16
	resultReg      = "x2"
	floatResultReg = "d0"
	leftReg        = "x0"
	rightReg       = "x1"
	floatLeftReg   = "d1"
	floatRightReg  = "d0"
	framePointer   = "x29"
	linkReg        = "x30"
)

// macOS syscall numbers
const (
	syscallExit  = 1
	syscallWrite = 4
)

// StackAlignment returns the required stack alignment for ARM64 (16 bytes).
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

// LinkReg returns the link register.
func (e *Emitter) LinkReg() string { return linkReg }

// ArgRegs returns the registers used for function arguments.
func (e *Emitter) ArgRegs() []string {
	return []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7"}
}

// EmitDataSection generates the .data section header with optional print support.
func (e *Emitter) EmitDataSection(hasPrint bool) string {
	var b strings.Builder
	b.WriteString(".data\n")
	b.WriteString(".align 3\n")
	if hasPrint {
		b.WriteString("buffer: .space 32\n")
		b.WriteString("newline: .byte 10\n")
	}
	return b.String()
}

// EmitProgramEntry generates the _start entry point that calls main.
func (e *Emitter) EmitProgramEntry() string {
	var b strings.Builder
	b.WriteString(".global _start\n")
	b.WriteString(".align 4\n")
	b.WriteString("_start:\n")
	b.WriteString("    bl _main\n")
	b.WriteString("    mov x16, #1\n")
	b.WriteString("    svc #0\n")
	b.WriteString("\n")
	return b.String()
}

// EmitFunctionLabel generates a function label with proper alignment.
func (e *Emitter) EmitFunctionLabel(name string) string {
	var b strings.Builder
	b.WriteString(".align 4\n")
	b.WriteString(fmt.Sprintf("_%s:\n", name))
	return b.String()
}

// EmitFunctionPrologue generates the standard ARM64 function prologue.
func (e *Emitter) EmitFunctionPrologue(stackSize int) string {
	var b strings.Builder
	b.WriteString("    stp x29, x30, [sp, #-16]!\n")
	b.WriteString("    mov x29, sp\n")
	if stackSize > 0 {
		b.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", stackSize))
	}
	return b.String()
}

// EmitFunctionEpilogue generates the standard ARM64 function epilogue.
func (e *Emitter) EmitFunctionEpilogue(hasLocals bool) string {
	var b strings.Builder
	if hasLocals {
		b.WriteString("    mov sp, x29\n")
	}
	b.WriteString("    ldp x29, x30, [sp], #16\n")
	b.WriteString("    ret\n")
	return b.String()
}

// EmitReturnEpilogue generates the epilogue for a return statement.
func (e *Emitter) EmitReturnEpilogue() string {
	var b strings.Builder
	b.WriteString("    mov sp, x29\n")
	b.WriteString("    ldp x29, x30, [sp], #16\n")
	b.WriteString("    ret\n")
	return b.String()
}

// EmitIntOp generates ARM64 assembly for an integer binary operation.
func (e *Emitter) EmitIntOp(op string, signed bool) (string, error) {
	var b strings.Builder

	switch op {
	case "+":
		b.WriteString("    add x2, x0, x1\n")
	case "-":
		b.WriteString("    sub x2, x0, x1\n")
	case "*":
		b.WriteString("    mul x2, x0, x1\n")
	case "/":
		if signed {
			b.WriteString("    sdiv x2, x0, x1\n")
		} else {
			b.WriteString("    udiv x2, x0, x1\n")
		}
	case "%":
		if signed {
			b.WriteString("    sdiv x3, x0, x1\n")
		} else {
			b.WriteString("    udiv x3, x0, x1\n")
		}
		b.WriteString("    msub x2, x3, x1, x0\n")
	case "==":
		b.WriteString("    cmp x0, x1\n")
		b.WriteString("    cset x2, eq\n")
	case "!=":
		b.WriteString("    cmp x0, x1\n")
		b.WriteString("    cset x2, ne\n")
	case "<":
		b.WriteString("    cmp x0, x1\n")
		if signed {
			b.WriteString("    cset x2, lt\n")
		} else {
			b.WriteString("    cset x2, lo\n")
		}
	case ">":
		b.WriteString("    cmp x0, x1\n")
		if signed {
			b.WriteString("    cset x2, gt\n")
		} else {
			b.WriteString("    cset x2, hi\n")
		}
	case "<=":
		b.WriteString("    cmp x0, x1\n")
		if signed {
			b.WriteString("    cset x2, le\n")
		} else {
			b.WriteString("    cset x2, ls\n")
		}
	case ">=":
		b.WriteString("    cmp x0, x1\n")
		if signed {
			b.WriteString("    cset x2, ge\n")
		} else {
			b.WriteString("    cset x2, hs\n")
		}
	default:
		return "", fmt.Errorf("unsupported integer operation: %s", op)
	}

	return b.String(), nil
}

// EmitFloatOp generates ARM64 assembly for a floating-point binary operation.
func (e *Emitter) EmitFloatOp(op string) (string, error) {
	var b strings.Builder

	switch op {
	case "+":
		b.WriteString("    fadd d0, d1, d0\n")
	case "-":
		b.WriteString("    fsub d0, d1, d0\n")
	case "*":
		b.WriteString("    fmul d0, d1, d0\n")
	case "/":
		b.WriteString("    fdiv d0, d1, d0\n")
	case "==":
		b.WriteString("    fcmp d1, d0\n")
		b.WriteString("    cset x2, eq\n")
	case "!=":
		b.WriteString("    fcmp d1, d0\n")
		b.WriteString("    cset x2, ne\n")
	case "<":
		b.WriteString("    fcmp d1, d0\n")
		b.WriteString("    cset x2, mi\n")
	case ">":
		b.WriteString("    fcmp d1, d0\n")
		b.WriteString("    cset x2, gt\n")
	case "<=":
		b.WriteString("    fcmp d1, d0\n")
		b.WriteString("    cset x2, ls\n")
	case ">=":
		b.WriteString("    fcmp d1, d0\n")
		b.WriteString("    cset x2, ge\n")
	default:
		return "", fmt.Errorf("unsupported float operation: %s", op)
	}

	return b.String(), nil
}

// EmitMoveReg generates a register-to-register move.
func (e *Emitter) EmitMoveReg(dst, src string) string {
	return fmt.Sprintf("    mov %s, %s\n", dst, src)
}

// EmitMoveImm generates an immediate value load.
// For values that fit in 16 bits, uses a simple MOV.
// For larger values, uses MOVZ/MOVK sequence to build the value in chunks.
func (e *Emitter) EmitMoveImm(reg, value string) string {
	// Try to parse as uint64 first (handles large unsigned values like u64 max)
	uval, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		// Try parsing as signed int64 (handles negative values)
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			// If parsing fails, just emit a simple mov (may fail at assembly time)
			return fmt.Sprintf("    mov %s, #%s\n", reg, value)
		}
		uval = uint64(val)
	}

	// For small values that fit in 16 bits, use simple mov
	if uval <= 65535 {
		return fmt.Sprintf("    mov %s, #%d\n", reg, uval)
	}

	// For larger values, use MOVZ/MOVK sequence
	var b strings.Builder

	// Find the first non-zero 16-bit chunk
	firstChunk := true
	for shift := 0; shift <= 48; shift += 16 {
		chunk := (uval >> shift) & 0xFFFF
		if chunk != 0 || (shift == 0 && uval == 0) {
			if firstChunk {
				b.WriteString(fmt.Sprintf("    movz %s, #%d", reg, chunk))
				if shift > 0 {
					b.WriteString(fmt.Sprintf(", lsl #%d", shift))
				}
				b.WriteString("\n")
				firstChunk = false
			} else {
				b.WriteString(fmt.Sprintf("    movk %s, #%d, lsl #%d\n", reg, chunk, shift))
			}
		}
	}

	// If value is 0 and we haven't written anything, emit mov #0
	if firstChunk {
		return fmt.Sprintf("    mov %s, #0\n", reg)
	}

	return b.String()
}

// EmitStoreToStack stores a register value to the stack relative to frame pointer.
func (e *Emitter) EmitStoreToStack(reg string, offset int) string {
	return fmt.Sprintf("    str %s, [x29, #-%d]\n", reg, offset)
}

// EmitLoadFromStack loads a value from the stack relative to frame pointer.
func (e *Emitter) EmitLoadFromStack(reg string, offset int) string {
	return fmt.Sprintf("    ldr %s, [x29, #-%d]\n", reg, offset)
}

// EmitPushToStack pushes a register value onto the stack (pre-decrement).
func (e *Emitter) EmitPushToStack(reg string) string {
	return fmt.Sprintf("    str %s, [sp, #-16]!\n", reg)
}

// EmitPopFromStack pops a value from the stack into a register (post-increment).
func (e *Emitter) EmitPopFromStack(reg string) string {
	return fmt.Sprintf("    ldr %s, [sp], #16\n", reg)
}

// EmitLoadAddress generates code to load a label address using adrp/add.
func (e *Emitter) EmitLoadAddress(reg, label string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("    adrp %s, %s@PAGE\n", reg, label))
	b.WriteString(fmt.Sprintf("    add %s, %s, %s@PAGEOFF\n", reg, reg, label))
	return b.String()
}

// EmitBranchLink generates a branch-and-link (function call).
func (e *Emitter) EmitBranchLink(label string) string {
	return fmt.Sprintf("    bl _%s\n", label)
}

// EmitExitSyscall generates the macOS exit syscall.
func (e *Emitter) EmitExitSyscall() string {
	var b strings.Builder
	b.WriteString("    mov x0, x2\n")
	b.WriteString("    mov x16, #1\n")
	b.WriteString("    svc #0\n")
	return b.String()
}

// EmitWriteSyscall generates assembly code to write data to stdout.
func (e *Emitter) EmitWriteSyscall(bufferReg, lengthReg string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("    mov x2, %s\n", lengthReg))
	b.WriteString(fmt.Sprintf("    mov x1, %s\n", bufferReg))
	b.WriteString("    mov x0, #1\n")
	b.WriteString("    mov x16, #4\n")
	b.WriteString("    svc #0x80\n")
	return b.String()
}

// EmitNewline generates assembly code to write a newline to stdout.
func (e *Emitter) EmitNewline() string {
	var b strings.Builder
	b.WriteString("    adrp x1, newline@PAGE\n")
	b.WriteString("    add x1, x1, newline@PAGEOFF\n")
	b.WriteString("    mov x2, #1\n")
	b.WriteString("    mov x0, #1\n")
	b.WriteString("    mov x16, #4\n")
	b.WriteString("    svc #0x80\n")
	return b.String()
}

// EmitPrintInt generates code to print an integer value.
func (e *Emitter) EmitPrintInt() string {
	var b strings.Builder
	b.WriteString("    mov x0, x2\n")
	b.WriteString("    bl int_to_string\n")
	b.WriteString("\n")
	b.WriteString(e.EmitWriteSyscall("x0", "x1"))
	b.WriteString("\n")
	b.WriteString(e.EmitNewline())
	return b.String()
}

// IntToStringFunction returns the int-to-string conversion routine.
func (e *Emitter) IntToStringFunction() string {
	return `.align 4
int_to_string:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

    adrp x19, buffer@PAGE
    add x19, x19, buffer@PAGEOFF
    mov x20, x0
    mov x21, #0

    cmp x20, #0
    bne check_negative
    mov w10, #48
    strb w10, [x19]
    mov x0, x19
    mov x1, #1
    b restore_regs

check_negative:
    cmp x20, #0
    bge convert_loop_setup
    mov x21, #1
    neg x20, x20

convert_loop_setup:
    mov x22, #0
    add x19, x19, #31

convert_loop:
    mov x10, #10
    udiv x11, x20, x10
    msub x12, x11, x10, x20
    add x12, x12, #48
    strb w12, [x19]
    sub x19, x19, #1
    add x22, x22, #1
    mov x20, x11
    cmp x20, #0
    bne convert_loop

    cmp x21, #1
    bne finalize
    mov w10, #45
    strb w10, [x19]
    sub x19, x19, #1
    add x22, x22, #1

finalize:
    add x19, x19, #1
    mov x0, x19
    mov x1, x22

restore_regs:
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret
`
}

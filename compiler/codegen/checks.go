package codegen

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/compiler/runtime"
)

// CheckGenerator manages runtime check code generation.
type CheckGenerator struct {
	labelCounter int
	filename     string
}

// NewCheckGenerator creates a new check generator.
func NewCheckGenerator(filename string) *CheckGenerator {
	return &CheckGenerator{
		filename: filename,
	}
}

// nextLabel returns a unique label suffix for panic branches.
func (c *CheckGenerator) nextLabel() int {
	c.labelCounter++
	return c.labelCounter
}

// CheckContext contains information needed to generate a runtime check.
type CheckContext struct {
	ErrorCode runtime.RuntimeError
	Line      int
	LabelID   int
}

// EmitSignedAddCheck generates overflow check for signed addition.
// Must be called INSTEAD of regular add - this emits adds with flags.
// Operands must be in x0 and x1, result will be in x2.
func (c *CheckGenerator) EmitSignedAddCheck(line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Use adds to set overflow flag, branch if NO overflow (vc = overflow clear)
	b.WriteString("    adds x2, x0, x1\n")
	b.WriteString(fmt.Sprintf("    b.vc _continue_%d\n", labelID))

	// Panic code (only reached on overflow)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrOverflowAddSigned))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")
	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

// EmitSignedSubCheck generates overflow check for signed subtraction.
func (c *CheckGenerator) EmitSignedSubCheck(line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Use subs to set overflow flag, branch if NO overflow (vc = overflow clear)
	b.WriteString("    subs x2, x0, x1\n")
	b.WriteString(fmt.Sprintf("    b.vc _continue_%d\n", labelID))

	// Panic code (only reached on overflow)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrOverflowSubSigned))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")
	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

// EmitSignedMulCheck generates overflow check for signed multiplication.
// This is more complex - we need to check if the high bits match the sign extension.
func (c *CheckGenerator) EmitSignedMulCheck(line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// mul gives low 64 bits, smulh gives high 64 bits
	b.WriteString("    mul x2, x0, x1\n")
	b.WriteString("    smulh x3, x0, x1\n")
	// If no overflow, high bits should be sign extension of low bits
	// Sign extension of x2 is: x2, asr #63 (all 0s or all 1s)
	// Branch if equal (no overflow)
	b.WriteString("    cmp x3, x2, asr #63\n")
	b.WriteString(fmt.Sprintf("    b.eq _continue_%d\n", labelID))

	// Panic code (only reached on overflow)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrOverflowMulSigned))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")
	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

// EmitUnsignedAddCheck generates overflow check for unsigned addition.
func (c *CheckGenerator) EmitUnsignedAddCheck(line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Branch if NO overflow (cc = carry clear)
	b.WriteString("    adds x2, x0, x1\n")
	b.WriteString(fmt.Sprintf("    b.cc _continue_%d\n", labelID))

	// Panic code (only reached on overflow)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrOverflowAddUnsigned))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")
	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

// EmitUnsignedSubCheck generates underflow check for unsigned subtraction.
func (c *CheckGenerator) EmitUnsignedSubCheck(line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Branch if NO underflow (cs = carry set means no borrow)
	b.WriteString("    subs x2, x0, x1\n")
	b.WriteString(fmt.Sprintf("    b.cs _continue_%d\n", labelID))

	// Panic code (only reached on underflow)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrUnderflowSubUnsigned))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")
	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

// EmitUnsignedMulCheck generates overflow check for unsigned multiplication.
func (c *CheckGenerator) EmitUnsignedMulCheck(line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	b.WriteString("    mul x2, x0, x1\n")
	b.WriteString("    umulh x3, x0, x1\n")
	// If no overflow, high bits should be zero - branch if zero (no overflow)
	b.WriteString(fmt.Sprintf("    cbz x3, _continue_%d\n", labelID))

	// Panic code (only reached on overflow)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrOverflowMulUnsigned))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")
	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

// EmitDivByZeroCheck generates division by zero check.
// Must be called BEFORE the division - divisor should be in x1.
func (c *CheckGenerator) EmitDivByZeroCheck(line int, signed bool) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Branch if divisor is non-zero (skip panic)
	b.WriteString(fmt.Sprintf("    cbnz x1, _ok_%d\n", labelID))

	// Panic code (only reached on division by zero)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrDivByZero))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")

	b.WriteString(fmt.Sprintf("_ok_%d:\n", labelID))
	if signed {
		b.WriteString("    sdiv x2, x0, x1\n")
	} else {
		b.WriteString("    udiv x2, x0, x1\n")
	}

	return b.String()
}

// EmitModByZeroCheck generates modulo by zero check.
// Must be called BEFORE the modulo - divisor should be in x1.
func (c *CheckGenerator) EmitModByZeroCheck(line int, signed bool) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Branch if divisor is non-zero (skip panic)
	b.WriteString(fmt.Sprintf("    cbnz x1, _ok_%d\n", labelID))

	// Panic code (only reached on modulo by zero)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrModByZero))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")

	b.WriteString(fmt.Sprintf("_ok_%d:\n", labelID))
	if signed {
		b.WriteString("    sdiv x3, x0, x1\n")
	} else {
		b.WriteString("    udiv x3, x0, x1\n")
	}
	b.WriteString("    msub x2, x3, x1, x0\n")

	return b.String()
}

// IntOperationChecked generates the appropriate checked operation.
// Returns the assembly code for the operation with runtime checks.
func (c *CheckGenerator) IntOperationChecked(op string, signed bool, line int) (string, error) {
	switch op {
	case "+":
		if signed {
			return c.EmitSignedAddCheck(line), nil
		}
		return c.EmitUnsignedAddCheck(line), nil

	case "-":
		if signed {
			return c.EmitSignedSubCheck(line), nil
		}
		return c.EmitUnsignedSubCheck(line), nil

	case "*":
		if signed {
			return c.EmitSignedMulCheck(line), nil
		}
		return c.EmitUnsignedMulCheck(line), nil

	case "/":
		return c.EmitDivByZeroCheck(line, signed), nil

	case "%":
		return c.EmitModByZeroCheck(line, signed), nil

	// Comparison operations don't need overflow checks
	case "==", "!=", "<", ">", "<=", ">=":
		return IntOperation(op, signed)

	default:
		return "", fmt.Errorf("unsupported operation: %s", op)
	}
}

// ArrayBoundsCheck generates runtime bounds check for array access.
// Index should be in x2, checks 0 <= x2 < size.
// Returns code that panics if out of bounds, otherwise continues.
func (c *CheckGenerator) ArrayBoundsCheck(size int, line int) string {
	labelID := c.nextLabel()
	var b strings.Builder

	// Use unsigned comparison: if index < size (as unsigned), it's in bounds
	// This handles both negative indices (which become large unsigned) and >= size
	b.WriteString(fmt.Sprintf("    cmp x2, #%d\n", size))
	b.WriteString(fmt.Sprintf("    b.lo _continue_%d\n", labelID)) // branch if lower (unsigned <), i.e., in bounds

	// Fall through to panic handler (out of bounds case)
	b.WriteString(fmt.Sprintf("    mov x0, #%d\n", runtime.ErrIndexOutOfBounds))
	b.WriteString(fmt.Sprintf("    mov x1, #%d\n", line))
	b.WriteString("    bl _slang_panic\n")

	b.WriteString(fmt.Sprintf("_continue_%d:\n", labelID))

	return b.String()
}

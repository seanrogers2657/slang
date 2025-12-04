package codegen

import "strings"

// BinaryExprEvaluator provides callbacks for generating binary expression code.
// This abstraction allows the same evaluation strategy to be used for both
// AST-based and typed code generation.
type BinaryExprEvaluator struct {
	// GenerateLeft generates code to evaluate the left operand (result in x2)
	GenerateLeft func() (string, error)
	// GenerateRight generates code to evaluate the right operand (result in x2)
	GenerateRight func() (string, error)
	// GenerateLeftToReg generates code to load the left operand into the given register
	GenerateLeftToReg func(reg string) (string, error)
	// GenerateRightToReg generates code to load the right operand into the given register
	GenerateRightToReg func(reg string) (string, error)
	// LeftIsComplex returns true if the left operand is a complex expression
	LeftIsComplex bool
	// RightIsComplex returns true if the right operand is a complex expression
	RightIsComplex bool
}

// EmitBinaryExprSetup generates the register setup code for a binary expression.
// After this code executes, x0 contains the left operand and x1 contains the right.
// Returns the generated assembly code.
func EmitBinaryExprSetup(eval *BinaryExprEvaluator) (string, error) {
	builder := strings.Builder{}

	switch {
	case eval.LeftIsComplex && eval.RightIsComplex:
		// Both complex: evaluate left → save → evaluate right → combine
		leftCode, err := eval.GenerateLeft()
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)
		EmitPushToStack(&builder, "x2")

		rightCode, err := eval.GenerateRight()
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
		EmitMoveReg(&builder, "x1", "x2")
		EmitPopFromStack(&builder, "x0")

	case eval.RightIsComplex:
		// Right is complex: evaluate right first, save, evaluate left
		rightCode, err := eval.GenerateRight()
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
		EmitPushToStack(&builder, "x2")

		leftCode, err := eval.GenerateLeftToReg("x0")
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)
		EmitPopFromStack(&builder, "x1")

	case eval.LeftIsComplex:
		// Left is complex: evaluate left first, save, evaluate right
		leftCode, err := eval.GenerateLeft()
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)
		EmitPushToStack(&builder, "x2")

		rightCode, err := eval.GenerateRightToReg("x1")
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
		EmitPopFromStack(&builder, "x0")

	default:
		// Both simple: load directly into registers
		leftCode, err := eval.GenerateLeftToReg("x0")
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)

		rightCode, err := eval.GenerateRightToReg("x1")
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
	}

	return builder.String(), nil
}

// EmitFloatBinaryExprSetup generates register setup for float binary expressions.
// After this code executes, d1 contains the left operand and d0 contains the right.
func EmitFloatBinaryExprSetup(generateLeft, generateRight func() (string, error)) (string, error) {
	builder := strings.Builder{}

	// Evaluate left operand into d0
	leftCode, err := generateLeft()
	if err != nil {
		return "", err
	}
	builder.WriteString(leftCode)
	builder.WriteString("    fmov d1, d0\n") // save left to d1

	// Evaluate right operand into d0
	rightCode, err := generateRight()
	if err != nil {
		return "", err
	}
	builder.WriteString(rightCode)
	// Now d1 = left, d0 = right

	return builder.String(), nil
}

// EmitCallSetup generates code to evaluate and stage function call arguments.
// Arguments are evaluated into x2, stored on stack, then loaded into x0-x7.
func EmitCallSetup(
	argCount int,
	generateArg func(index int) (string, error),
) (string, error) {
	if argCount == 0 {
		return "", nil
	}

	builder := strings.Builder{}

	// Allocate space for arguments on stack
	builder.WriteString("    sub sp, sp, #")
	builder.WriteString(intToStr(argCount * 16))
	builder.WriteString("\n")

	// Evaluate each argument and store on stack
	for i := 0; i < argCount; i++ {
		code, err := generateArg(i)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		// Store result (in x2) to stack
		builder.WriteString("    str x2, [sp, #")
		builder.WriteString(intToStr(i * 16))
		builder.WriteString("]\n")
	}

	// Load arguments from stack into registers x0-x7
	for i := 0; i < argCount && i < 8; i++ {
		builder.WriteString("    ldr x")
		builder.WriteString(intToStr(i))
		builder.WriteString(", [sp, #")
		builder.WriteString(intToStr(i * 16))
		builder.WriteString("]\n")
	}

	// Restore stack pointer
	builder.WriteString("    add sp, sp, #")
	builder.WriteString(intToStr(argCount * 16))
	builder.WriteString("\n")

	return builder.String(), nil
}

// intToStr is a simple int to string converter to avoid fmt import.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}

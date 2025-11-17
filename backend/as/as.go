package as

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/ast"
)

type AsGenerator interface {
	Generate() (string, error)
}

func NewAsGenerator(
	program *ast.Program,
) AsGenerator {
	return &asGenerator{
		program: program,
	}
}

type asGenerator struct {
	program *ast.Program
}

func (c *asGenerator) Generate() (string, error) {
	return GenerateProgram(c.program)
}

func GenerateProgram(program *ast.Program) (string, error) {
	builder := strings.Builder{}

	// Check if we need a .data section for strings or print statements
	hasStrings := false
	hasPrint := false
	stringIndex := 0
	stringMap := make(map[*ast.LiteralExpr]string) // Map literals to their label names

	// Helper function to collect string literals from an expression
	var collectStrings func(ast.Expression)
	collectStrings = func(expr ast.Expression) {
		switch e := expr.(type) {
		case *ast.BinaryExpr:
			collectStrings(e.Left)
			collectStrings(e.Right)
		case *ast.LiteralExpr:
			if e.Kind == ast.LiteralTypeString {
				hasStrings = true
				if _, exists := stringMap[e]; !exists {
					stringMap[e] = fmt.Sprintf("str_%d", stringIndex)
					stringIndex++
				}
			}
		}
	}

	for _, stmt := range program.Statements {
		// Check for print statements
		if _, ok := stmt.(*ast.PrintStmt); ok {
			hasPrint = true
		}

		// Check for string literals in statements
		switch s := stmt.(type) {
		case *ast.ExprStmt:
			collectStrings(s.Expr)
		case *ast.PrintStmt:
			collectStrings(s.Expr)
		}
	}

	// Write .data section if needed
	if hasStrings || hasPrint {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")

		// Add buffer for print statements
		if hasPrint {
			builder.WriteString("buffer: .space 32\n")
			builder.WriteString("newline: .byte 10\n")
		}

		// Define all unique string literals
		for literal, label := range stringMap {
			builder.WriteString(fmt.Sprintf("%s:\n", label))
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", literal.Value))
		}

		builder.WriteString("\n.text\n")
	}

	builder.WriteString(".global _start\n")
	builder.WriteString(".align 4\n")
	builder.WriteString("_start:\n")
	builder.WriteString("    b main\n")
	builder.WriteString("\n")

	// Add int-to-string conversion function if we have print statements
	if hasPrint {
		builder.WriteString(intToStringFunctionText())
		builder.WriteString("\n")
	}

	builder.WriteString("main:\n")

	// Generate code for each statement
	for _, stmt := range program.Statements {
		var code string
		var err error

		switch s := stmt.(type) {
		case *ast.ExprStmt:
			code, err = GenerateExprInline(s.Expr, stringMap)
		case *ast.PrintStmt:
			code, err = GeneratePrintStmt(s, stringMap)
		default:
			return "", fmt.Errorf("unknown statement type")
		}

		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Exit syscall (only at the end)
	builder.WriteString("    mov x0, #0\n")
	builder.WriteString("    mov x16, #1\n")
	builder.WriteString("    svc #0x80\n")

	return builder.String(), nil
}

func GenerateExprInline(expr ast.Expression, stringMap map[*ast.LiteralExpr]string) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		// Single literal - just load into x2
		if e.Kind == ast.LiteralTypeString {
			label := stringMap[e]
			builder.WriteString(fmt.Sprintf("    adrp x2, %s@PAGE\n", label))
			builder.WriteString(fmt.Sprintf("    add x2, x2, %s@PAGEOFF\n", label))
		} else {
			builder.WriteString(fmt.Sprintf("    mov x2, #%s\n", e.Value))
		}
		return builder.String(), nil

	case *ast.BinaryExpr:
		// Load left operand into x0
		if leftLit, ok := e.Left.(*ast.LiteralExpr); ok {
			if leftLit.Kind == ast.LiteralTypeString {
				label := stringMap[leftLit]
				builder.WriteString(fmt.Sprintf("    adrp x0, %s@PAGE\n", label))
				builder.WriteString(fmt.Sprintf("    add x0, x0, %s@PAGEOFF\n", label))
			} else {
				builder.WriteString(fmt.Sprintf("    mov x0, #%s\n", leftLit.Value))
			}
		} else {
			return "", fmt.Errorf("unsupported left operand type in binary expression")
		}

		// Load right operand into x1
		if rightLit, ok := e.Right.(*ast.LiteralExpr); ok {
			if rightLit.Kind == ast.LiteralTypeString {
				label := stringMap[rightLit]
				builder.WriteString(fmt.Sprintf("    adrp x1, %s@PAGE\n", label))
				builder.WriteString(fmt.Sprintf("    add x1, x1, %s@PAGEOFF\n", label))
			} else {
				builder.WriteString(fmt.Sprintf("    mov x1, #%s\n", rightLit.Value))
			}
		} else {
			return "", fmt.Errorf("unsupported right operand type in binary expression")
		}

		// Generate operation
		switch e.Op {
		case "+":
			builder.WriteString("    add x2, x0, x1\n")
		case "-":
			builder.WriteString("    sub x2, x0, x1\n")
		case "*":
			builder.WriteString("    mul x2, x0, x1\n")
		case "/":
			builder.WriteString("    sdiv x2, x0, x1\n")
		case "%":
			// Modulo: x2 = x0 - (x0 / x1) * x1
			builder.WriteString("    sdiv x3, x0, x1\n")
			builder.WriteString("    msub x2, x3, x1, x0\n")
		case "==":
			builder.WriteString("    cmp x0, x1\n")
			builder.WriteString("    cset x2, eq\n")
		case "!=":
			builder.WriteString("    cmp x0, x1\n")
			builder.WriteString("    cset x2, ne\n")
		case "<":
			builder.WriteString("    cmp x0, x1\n")
			builder.WriteString("    cset x2, lt\n")
		case ">":
			builder.WriteString("    cmp x0, x1\n")
			builder.WriteString("    cset x2, gt\n")
		case "<=":
			builder.WriteString("    cmp x0, x1\n")
			builder.WriteString("    cset x2, le\n")
		case ">=":
			builder.WriteString("    cmp x0, x1\n")
			builder.WriteString("    cset x2, ge\n")
		default:
			return "", fmt.Errorf("unsupported operation %s when generating code", e.Op)
		}

		return builder.String(), nil

	default:
		return "", fmt.Errorf("unsupported expression type")
	}
}

func GenerateExpr(expr *ast.BinaryExpr) (string, error) {
	builder := strings.Builder{}

	// Check if we need a .data section for strings
	hasStrings := false
	var leftLit, rightLit *ast.LiteralExpr

	if left, ok := expr.Left.(*ast.LiteralExpr); ok {
		leftLit = left
		if left.Kind == ast.LiteralTypeString {
			hasStrings = true
		}
	}

	if right, ok := expr.Right.(*ast.LiteralExpr); ok {
		rightLit = right
		if right.Kind == ast.LiteralTypeString {
			hasStrings = true
		}
	}

	if hasStrings {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")

		// Define string literals in .data section
		if leftLit != nil && leftLit.Kind == ast.LiteralTypeString {
			builder.WriteString("str_left:\n")
			// Escape the string for assembly
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", leftLit.Value))
		}

		if rightLit != nil && rightLit.Kind == ast.LiteralTypeString {
			builder.WriteString("str_right:\n")
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", rightLit.Value))
		}

		builder.WriteString("\n.text\n")
	}

	builder.WriteString(".global _start\n")
	builder.WriteString(".align 4\n")

	builder.WriteString("_start:\n")

	// Load left operand
	if leftLit != nil {
		if leftLit.Kind == ast.LiteralTypeString {
			builder.WriteString("    adr x0, str_left\n")
		} else {
			builder.WriteString(fmt.Sprintf("    mov x0, #%s\n", leftLit.Value))
		}
	}

	// Load right operand
	if rightLit != nil {
		if rightLit.Kind == ast.LiteralTypeString {
			builder.WriteString("    adr x1, str_right\n")
		} else {
			builder.WriteString(fmt.Sprintf("    mov x1, #%s\n", rightLit.Value))
		}
	}

	switch expr.Op {
	case "+":
		builder.WriteString("    add x2, x0, x1\n")
	case "-":
		builder.WriteString("    sub x2, x0, x1\n")
	case "*":
		builder.WriteString("    mul x2, x0, x1\n")
	case "/":
		builder.WriteString("    sdiv x2, x0, x1\n")
	case "%":
		// Modulo: x2 = x0 - (x0 / x1) * x1
		builder.WriteString("    sdiv x3, x0, x1\n")
		builder.WriteString("    msub x2, x3, x1, x0\n")
	case "==":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, eq\n")
	case "!=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, ne\n")
	case "<":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, lt\n")
	case ">":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, gt\n")
	case "<=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, le\n")
	case ">=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, ge\n")
	default:
		return "", fmt.Errorf("unsupported operation %s when generating code", expr.Op)
	}

	builder.WriteString("    mov x0, #0\n")
	builder.WriteString("    mov x16, #1\n")
	builder.WriteString("    svc #0x80\n")

	return builder.String(), nil
}

// GeneratePrintStmt generates code for a print statement.
// It evaluates the expression, converts the result to a string, and writes it to stdout.
func GeneratePrintStmt(stmt *ast.PrintStmt, stringMap map[*ast.LiteralExpr]string) (string, error) {
	builder := strings.Builder{}

	// Step 1: Evaluate the expression (result goes into x2)
	code, err := GenerateExprInline(stmt.Expr, stringMap)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Step 2: Convert integer to string
	builder.WriteString("    mov x0, x2              // Pass value to convert\n")
	builder.WriteString("    bl int_to_string        // Returns buffer in x0, length in x1\n")
	builder.WriteString("\n")

	// Step 3: Write the string to stdout
	builder.WriteString(generateWriteSyscall("x0", "x1"))
	builder.WriteString("\n")

	// Step 4: Write a newline character
	builder.WriteString(generateNewline())

	return builder.String(), nil
}

// generateWriteSyscall creates assembly code to write data to stdout.
// bufferReg is the register containing the buffer address.
// lengthReg is the register containing the length.
func generateWriteSyscall(bufferReg, lengthReg string) string {
	return fmt.Sprintf("    mov x2, %s              // Length\n", lengthReg) +
		fmt.Sprintf("    mov x1, %s              // Buffer address\n", bufferReg) +
		"    mov x0, #1              // File descriptor: stdout\n" +
		"    mov x16, #4             // Syscall number: write\n" +
		"    svc #0x80               // Make syscall\n"
}

// generateNewline creates assembly code to write a newline to stdout.
func generateNewline() string {
	return "    adrp x1, newline@PAGE\n" +
		"    add x1, x1, newline@PAGEOFF\n" +
		"    mov x2, #1              // Length of newline\n" +
		"    mov x0, #1              // File descriptor: stdout\n" +
		"    mov x16, #4             // Syscall number: write\n" +
		"    svc #0x80               // Make syscall\n"
}

// intToStringFunctionText generates the int-to-string conversion routine.
// This function converts an integer to its ASCII string representation.
//
// Input:  x0 = integer to convert
// Output: x0 = address of string buffer, x1 = length of string
//
// Register usage:
//   x19 = buffer pointer (moves through buffer)
//   x20 = number being converted (modified during conversion)
//   x21 = is_negative flag (1 if negative, 0 otherwise)
//   x22 = digit count
func intToStringFunctionText() string {
	return buildIntToStringFunction()
}

func buildIntToStringFunction() string {
	return functionPrologue() +
		setupConversion() +
		handleZeroCase() +
		handleNegativeNumber() +
		convertDigitsToASCII() +
		addMinusSignIfNeeded() +
		functionEpilogue()
}

// functionPrologue saves registers and sets up the stack frame
func functionPrologue() string {
	return `.align 4
int_to_string:
    // Save callee-saved registers to stack
    stp x29, x30, [sp, #-16]!   // Save frame pointer and link register
    mov x29, sp                  // Set up frame pointer
    stp x19, x20, [sp, #-16]!   // Save working registers
    stp x21, x22, [sp, #-16]!   // Save working registers

`
}

// setupConversion initializes registers for the conversion process
func setupConversion() string {
	return `    // Initialize conversion state
    adrp x19, buffer@PAGE        // Load buffer address (page)
    add x19, x19, buffer@PAGEOFF // Load buffer address (offset)
    mov x20, x0                  // x20 = number to convert
    mov x21, #0                  // x21 = is_negative flag (0 = positive)

`
}

// handleZeroCase handles the special case when the input is zero
func handleZeroCase() string {
	return `    // Special case: if number is 0, return "0"
    cmp x20, #0
    bne check_negative
    mov w10, #48                 // ASCII '0' = 48
    strb w10, [x19]             // Store '0' in buffer
    mov x0, x19                  // Return buffer address
    mov x1, #1                   // Return length = 1
    b restore_regs

`
}

// handleNegativeNumber checks for negative numbers and converts them to positive
func handleNegativeNumber() string {
	return `check_negative:
    // Check if number is negative
    cmp x20, #0
    bge convert_loop_setup       // If positive or zero, skip

    // Number is negative: set flag and make it positive
    mov x21, #1                  // Set is_negative flag
    neg x20, x20                 // x20 = -x20 (make positive)

`
}

// convertDigitsToASCII converts the number to ASCII digits (backwards)
func convertDigitsToASCII() string {
	return `convert_loop_setup:
    mov x22, #0                  // x22 = digit count (starts at 0)
    add x19, x19, #31            // Point to end of 32-byte buffer

convert_loop:
    // Extract rightmost digit using division by 10
    mov x10, #10
    udiv x11, x20, x10           // x11 = quotient (x20 / 10)
    msub x12, x11, x10, x20      // x12 = remainder (x20 % 10)

    // Convert digit to ASCII and store it
    add x12, x12, #48            // Convert to ASCII ('0' = 48)
    strb w12, [x19]              // Store digit in buffer
    sub x19, x19, #1             // Move pointer backwards
    add x22, x22, #1             // Increment digit count

    // Continue if there are more digits
    mov x20, x11                 // x20 = quotient
    cmp x20, #0
    bne convert_loop             // Loop if quotient > 0

`
}

// addMinusSignIfNeeded adds a minus sign if the number was negative
func addMinusSignIfNeeded() string {
	return `    // If number was negative, prepend '-' sign
    cmp x21, #1
    bne finalize
    mov w10, #45                 // ASCII '-' = 45
    strb w10, [x19]             // Store '-' in buffer
    sub x19, x19, #1             // Move pointer backwards
    add x22, x22, #1             // Increment digit count

`
}

// functionEpilogue finalizes the result and restores registers
func functionEpilogue() string {
	return `finalize:
    // Adjust pointer to start of string and set return values
    add x19, x19, #1             // Move to first character
    mov x0, x19                  // Return buffer address
    mov x1, x22                  // Return string length

restore_regs:
    // Restore callee-saved registers from stack
    ldp x21, x22, [sp], #16     // Restore working registers
    ldp x19, x20, [sp], #16     // Restore working registers
    ldp x29, x30, [sp], #16     // Restore frame pointer and link register
    ret
`
}

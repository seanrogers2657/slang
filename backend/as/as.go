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
	builder.WriteString("    mov x0, #1\n")
	builder.WriteString("    mov x16, #0\n")
	builder.WriteString("    svc #0\n")

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

	builder.WriteString("    mov x0, #1\n")
	builder.WriteString("    mov x16, #0\n")
	builder.WriteString("    svc #0\n")

	return builder.String(), nil
}

// GeneratePrintStmt generates code for a print statement
func GeneratePrintStmt(stmt *ast.PrintStmt, stringMap map[*ast.LiteralExpr]string) (string, error) {
	builder := strings.Builder{}

	// Evaluate the expression to print
	code, err := GenerateExprInline(stmt.Expr, stringMap)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// The result is in x2, move it to x0 for int_to_string
	builder.WriteString("    mov x0, x2\n")
	builder.WriteString("    bl int_to_string\n")

	// Write syscall (x0 has buffer address, x1 has length from int_to_string)
	builder.WriteString("    mov x2, x1          // length to x2\n")
	builder.WriteString("    mov x1, x0          // buffer to x1\n")
	builder.WriteString("    mov x0, #1          // stdout\n")
	builder.WriteString("    mov x16, #4         // write syscall\n")
	builder.WriteString("    svc #0x80\n")

	// Print newline
	builder.WriteString("    adrp x1, newline@PAGE\n")
	builder.WriteString("    add x1, x1, newline@PAGEOFF\n")
	builder.WriteString("    mov x2, #1\n")
	builder.WriteString("    mov x0, #1\n")
	builder.WriteString("    mov x16, #4\n")
	builder.WriteString("    svc #0x80\n")

	return builder.String(), nil
}

// intToStringFunctionText generates the int-to-string conversion routine (text only)
func intToStringFunctionText() string {
	return `.align 4
int_to_string:
    // Input: x0 = integer to convert
    // Output: x0 = address of string, x1 = length
    // Save registers
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

    adrp x19, buffer@PAGE        // x19 = buffer page address
    add x19, x19, buffer@PAGEOFF // x19 = buffer address
    mov x20, x0                  // x20 = number to convert
    mov x21, #0                  // x21 = is_negative flag

    // Handle zero special case
    cmp x20, #0
    bne check_negative
    mov w10, #48            // '0' = 48
    strb w10, [x19]
    mov x0, x19
    mov x1, #1
    b restore_regs

check_negative:
    // Check if negative
    cmp x20, #0
    bge convert_loop_setup

    // Handle negative: set flag and negate
    mov x21, #1
    neg x20, x20

convert_loop_setup:
    mov x22, #0             // x22 = digit count
    add x19, x19, #31       // Start at end of buffer

convert_loop:
    // Divide by 10
    mov x10, #10
    udiv x11, x20, x10      // x11 = x20 / 10
    msub x12, x11, x10, x20 // x12 = x20 % 10 (remainder)

    // Convert digit to ASCII
    add x12, x12, #48       // '0' = 48
    strb w12, [x19]         // Store byte
    sub x19, x19, #1        // Move backwards
    add x22, x22, #1        // Increment digit count

    // Continue if quotient > 0
    mov x20, x11
    cmp x20, #0
    bne convert_loop

    // Add minus sign if negative
    cmp x21, #1
    bne finalize
    strb wzr, [x19]         // This will be '-'
    mov w10, #45            // '-' = 45
    strb w10, [x19]
    sub x19, x19, #1
    add x22, x22, #1

finalize:
    add x19, x19, #1        // Adjust to start of string
    mov x0, x19             // Return buffer address
    mov x1, x22             // Return length

restore_regs:
    // Restore registers
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret
`
}

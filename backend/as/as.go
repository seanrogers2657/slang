package as

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/ast"
)

// CodeGenContext tracks state during code generation
type CodeGenContext struct {
	variables   map[string]int // variable name → stack offset (negative from frame pointer)
	stackOffset int            // current stack position (starts at -16, decrements by 16)
	stringMap   map[*ast.LiteralExpr]string
}

// newCodeGenContext creates a new code generation context
func newCodeGenContext(stringMap map[*ast.LiteralExpr]string) *CodeGenContext {
	return &CodeGenContext{
		variables:   make(map[string]int),
		stackOffset: 0, // We'll reserve stack space as we go
		stringMap:   stringMap,
	}
}

// declareVariable allocates stack space for a variable
func (ctx *CodeGenContext) declareVariable(name string) int {
	ctx.stackOffset += 16 // 16-byte aligned for ARM64
	ctx.variables[name] = ctx.stackOffset
	return ctx.stackOffset
}

// getVariableOffset returns the stack offset for a variable
func (ctx *CodeGenContext) getVariableOffset(name string) (int, bool) {
	offset, ok := ctx.variables[name]
	return offset, ok
}

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

	// Determine which statements to process (function-based or legacy)
	var statementsToProcess []ast.Statement

	if len(program.Declarations) > 0 {
		// Function-based program: extract statements from main function
		var mainFunc *ast.FunctionDecl
		for _, decl := range program.Declarations {
			if fn, ok := decl.(*ast.FunctionDecl); ok && fn.Name == "main" {
				mainFunc = fn
				break
			}
		}

		if mainFunc == nil {
			return "", fmt.Errorf("no main function found")
		}

		statementsToProcess = mainFunc.Body.Statements
	} else {
		// Legacy: use top-level statements
		statementsToProcess = program.Statements
	}

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

	for _, stmt := range statementsToProcess {
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

	// Create code generation context
	ctx := newCodeGenContext(stringMap)

	// Count variables to determine stack frame size
	varCount := countVariables(statementsToProcess)

	// Generate function prologue if we have variables
	if varCount > 0 {
		// Save frame pointer and link register
		builder.WriteString("    stp x29, x30, [sp, #-16]!\n")
		builder.WriteString("    mov x29, sp\n")

		// Allocate stack space for variables (16-byte aligned)
		stackSize := varCount * 16
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", stackSize))
	}

	// Generate code for each statement
	for _, stmt := range statementsToProcess {
		var code string
		var err error

		switch s := stmt.(type) {
		case *ast.ExprStmt:
			code, err = GenerateExprWithContext(s.Expr, ctx)
		case *ast.PrintStmt:
			code, err = GeneratePrintStmtWithContext(s, ctx)
		case *ast.VarDeclStmt:
			code, err = GenerateVarDecl(s, ctx)
		case *ast.AssignStmt:
			code, err = GenerateAssignStmt(s, ctx)
		default:
			return "", fmt.Errorf("unknown statement type: %T", s)
		}

		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Generate function epilogue if we have variables
	if varCount > 0 {
		builder.WriteString("    mov sp, x29\n")
		builder.WriteString("    ldp x29, x30, [sp], #16\n")
	}

	// Exit syscall (only at the end)
	builder.WriteString("    mov x0, #0\n")
	builder.WriteString("    mov x16, #1\n")
	builder.WriteString("    svc #0\n")

	return builder.String(), nil
}

// countVariables counts the number of variable declarations in statements
func countVariables(stmts []ast.Statement) int {
	count := 0
	for _, stmt := range stmts {
		if _, ok := stmt.(*ast.VarDeclStmt); ok {
			count++
		}
	}
	return count
}

// GenerateVarDecl generates code for a variable declaration.
// Similar to GenerateAssignStmt but declares a new variable slot.
func GenerateVarDecl(stmt *ast.VarDeclStmt, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	// Generate code to evaluate the initializer (result in x2)
	code, err := GenerateExprWithContext(stmt.Initializer, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Allocate stack slot for this variable
	offset := ctx.declareVariable(stmt.Name)

	// Store the value to stack (relative to frame pointer x29)
	builder.WriteString(fmt.Sprintf("    str x2, [x29, #-%d]\n", offset))

	return builder.String(), nil
}

// GenerateAssignStmt generates code for a variable assignment.
// Similar to GenerateVarDecl but uses an existing variable slot.
func GenerateAssignStmt(stmt *ast.AssignStmt, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	// Generate code to evaluate the value (result in x2)
	code, err := GenerateExprWithContext(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Get the existing stack slot for this variable
	offset, ok := ctx.getVariableOffset(stmt.Name)
	if !ok {
		return "", fmt.Errorf("undefined variable: %s", stmt.Name)
	}

	// Store the value to stack (relative to frame pointer x29)
	builder.WriteString(fmt.Sprintf("    str x2, [x29, #-%d]\n", offset))

	return builder.String(), nil
}

// GenerateExprWithContext generates code for an expression with variable support
func GenerateExprWithContext(expr ast.Expression, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		// Single literal - just load into x2
		if e.Kind == ast.LiteralTypeString {
			label := ctx.stringMap[e]
			builder.WriteString(fmt.Sprintf("    adrp x2, %s@PAGE\n", label))
			builder.WriteString(fmt.Sprintf("    add x2, x2, %s@PAGEOFF\n", label))
		} else {
			builder.WriteString(fmt.Sprintf("    mov x2, #%s\n", e.Value))
		}
		return builder.String(), nil

	case *ast.IdentifierExpr:
		// Load variable from stack
		offset, ok := ctx.getVariableOffset(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		builder.WriteString(fmt.Sprintf("    ldr x2, [x29, #-%d]\n", offset))
		return builder.String(), nil

	case *ast.BinaryExpr:
		// Check if operands are complex expressions (binary expressions that could clobber registers)
		_, leftIsBinary := e.Left.(*ast.BinaryExpr)
		_, rightIsBinary := e.Right.(*ast.BinaryExpr)

		if leftIsBinary && rightIsBinary {
			// Both are complex: evaluate left first, save it, evaluate right, then combine
			leftCode, err := GenerateExprWithContext(e.Left, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(leftCode)
			builder.WriteString("    str x2, [sp, #-16]!  // Save left operand\n")

			rightCode, err := GenerateExprWithContext(e.Right, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
			builder.WriteString("    mov x1, x2           // Move right to x1\n")
			builder.WriteString("    ldr x0, [sp], #16    // Restore left operand\n")

		} else if rightIsBinary {
			// Right is complex: evaluate right first, save it, then evaluate left
			rightCode, err := GenerateExprWithContext(e.Right, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
			builder.WriteString("    str x2, [sp, #-16]!  // Save right operand\n")

			leftCode, err := generateOperandToReg(e.Left, "x0", ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(leftCode)
			builder.WriteString("    ldr x1, [sp], #16    // Restore right operand\n")

		} else if leftIsBinary {
			// Left is complex: evaluate left first, save it, then evaluate right
			leftCode, err := GenerateExprWithContext(e.Left, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(leftCode)
			builder.WriteString("    str x2, [sp, #-16]!  // Save left operand\n")

			rightCode, err := generateOperandToReg(e.Right, "x1", ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
			builder.WriteString("    ldr x0, [sp], #16    // Restore left operand\n")

		} else {
			// Simple case: both operands are simple (literals or identifiers)
			leftCode, err := generateOperandToReg(e.Left, "x0", ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(leftCode)

			rightCode, err := generateOperandToReg(e.Right, "x1", ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
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
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// generateOperandToReg generates code to load an operand into a register
func generateOperandToReg(expr ast.Expression, reg string, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		if e.Kind == ast.LiteralTypeString {
			label := ctx.stringMap[e]
			builder.WriteString(fmt.Sprintf("    adrp %s, %s@PAGE\n", reg, label))
			builder.WriteString(fmt.Sprintf("    add %s, %s, %s@PAGEOFF\n", reg, reg, label))
		} else {
			builder.WriteString(fmt.Sprintf("    mov %s, #%s\n", reg, e.Value))
		}

	case *ast.IdentifierExpr:
		offset, ok := ctx.getVariableOffset(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		builder.WriteString(fmt.Sprintf("    ldr %s, [x29, #-%d]\n", reg, offset))

	case *ast.BinaryExpr:
		// For nested binary expressions, we need to evaluate and store in temp
		// First, evaluate the nested expression (result in x2)
		code, err := GenerateExprWithContext(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		// Move from x2 to target register
		if reg != "x2" {
			builder.WriteString(fmt.Sprintf("    mov %s, x2\n", reg))
		}

	default:
		return "", fmt.Errorf("unsupported operand type: %T", expr)
	}

	return builder.String(), nil
}

// GeneratePrintStmtWithContext generates code for a print statement with variable support
func GeneratePrintStmtWithContext(stmt *ast.PrintStmt, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	// Step 1: Evaluate the expression (result goes into x2)
	code, err := GenerateExprWithContext(stmt.Expr, ctx)
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
	builder.WriteString("    svc #0\n")

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

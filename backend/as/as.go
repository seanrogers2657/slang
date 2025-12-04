package as

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/semantic"
)

// CodeGenContext tracks state during code generation
type CodeGenContext struct {
	variables   map[string]int // variable name → stack offset (negative from frame pointer)
	stackOffset int            // current stack position (starts at -16, decrements by 16)
	stringMap   map[*ast.LiteralExpr]string
	sourceLines []string // source code lines for comment generation
}

// newCodeGenContext creates a new code generation context
func newCodeGenContext(stringMap map[*ast.LiteralExpr]string, sourceLines []string) *CodeGenContext {
	return &CodeGenContext{
		variables:   make(map[string]int),
		stackOffset: 0, // We'll reserve stack space as we go
		stringMap:   stringMap,
		sourceLines: sourceLines,
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

// getSourceLineComment returns a comment with the source line for a given position
func (ctx *CodeGenContext) getSourceLineComment(pos ast.Position) string {
	if ctx.sourceLines == nil || pos.Line <= 0 || pos.Line > len(ctx.sourceLines) {
		return ""
	}
	line := strings.TrimSpace(ctx.sourceLines[pos.Line-1])
	if line == "" {
		return ""
	}
	return fmt.Sprintf("// %d: %s\n", pos.Line, line)
}

type AsGenerator interface {
	Generate() (string, error)
}

func NewAsGenerator(
	program *ast.Program,
	sourceLines []string,
) AsGenerator {
	return &asGenerator{
		program:     program,
		sourceLines: sourceLines,
	}
}

type asGenerator struct {
	program     *ast.Program
	sourceLines []string
}

func (c *asGenerator) Generate() (string, error) {
	return GenerateProgram(c.program, c.sourceLines)
}

func GenerateProgram(program *ast.Program, sourceLines []string) (string, error) {
	builder := strings.Builder{}

	// Handle function-based programs with proper function support
	if len(program.Declarations) > 0 {
		return generateFunctionBasedProgram(program, &builder, sourceLines)
	}

	// Legacy: use top-level statements
	return generateLegacyProgram(program, &builder, sourceLines)
}

// generateFunctionBasedProgram generates code for programs with function declarations
func generateFunctionBasedProgram(program *ast.Program, builder *strings.Builder, sourceLines []string) (string, error) {
	// Collect all functions
	functions := make([]*ast.FunctionDecl, 0)
	for _, decl := range program.Declarations {
		if fn, ok := decl.(*ast.FunctionDecl); ok {
			functions = append(functions, fn)
		}
	}

	if len(functions) == 0 {
		return "", fmt.Errorf("no functions found")
	}

	// Check for print statements and collect string literals across all functions
	hasPrint := false
	stringIndex := 0
	stringMap := make(map[*ast.LiteralExpr]string)

	var collectStringsFromExpr func(ast.Expression)
	collectStringsFromExpr = func(expr ast.Expression) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.BinaryExpr:
			collectStringsFromExpr(e.Left)
			collectStringsFromExpr(e.Right)
		case *ast.LiteralExpr:
			if e.Kind == ast.LiteralTypeString {
				if _, exists := stringMap[e]; !exists {
					stringMap[e] = fmt.Sprintf("str_%d", stringIndex)
					stringIndex++
				}
			}
		case *ast.CallExpr:
			for _, arg := range e.Arguments {
				collectStringsFromExpr(arg)
			}
		}
	}

	// Check for print calls in expressions
	var checkForPrintCall func(ast.Expression)
	checkForPrintCall = func(expr ast.Expression) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.CallExpr:
			if e.Name == "print" {
				hasPrint = true
			}
			for _, arg := range e.Arguments {
				checkForPrintCall(arg)
			}
		case *ast.BinaryExpr:
			checkForPrintCall(e.Left)
			checkForPrintCall(e.Right)
		}
	}

	var collectStringsFromStmt func(ast.Statement)
	collectStringsFromStmt = func(stmt ast.Statement) {
		switch s := stmt.(type) {
		case *ast.ExprStmt:
			collectStringsFromExpr(s.Expr)
			checkForPrintCall(s.Expr)
		case *ast.VarDeclStmt:
			collectStringsFromExpr(s.Initializer)
			checkForPrintCall(s.Initializer)
		case *ast.AssignStmt:
			collectStringsFromExpr(s.Value)
			checkForPrintCall(s.Value)
		case *ast.ReturnStmt:
			collectStringsFromExpr(s.Value)
			checkForPrintCall(s.Value)
		}
	}

	for _, fn := range functions {
		for _, stmt := range fn.Body.Statements {
			collectStringsFromStmt(stmt)
		}
	}

	// Write .data section if needed
	if len(stringMap) > 0 || hasPrint {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")

		if hasPrint {
			builder.WriteString("buffer: .space 32\n")
			builder.WriteString("newline: .byte 10\n")
		}

		for literal, label := range stringMap {
			builder.WriteString(fmt.Sprintf("%s:\n", label))
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", literal.Value))
		}

		builder.WriteString("\n.text\n")
	}

	builder.WriteString(".global _start\n")
	builder.WriteString(".align 4\n")
	builder.WriteString("_start:\n")
	builder.WriteString("    bl _main\n")
	builder.WriteString("    mov x16, #1\n")
	builder.WriteString("    svc #0\n")
	builder.WriteString("\n")

	// Add int-to-string conversion function if we have print statements
	if hasPrint {
		builder.WriteString(intToStringFunctionText())
		builder.WriteString("\n")
	}

	// Generate code for each function
	for _, fn := range functions {
		code, err := GenerateFunction(fn, stringMap, sourceLines)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// GenerateFunction generates code for a single function
func GenerateFunction(fn *ast.FunctionDecl, stringMap map[*ast.LiteralExpr]string, sourceLines []string) (string, error) {
	builder := strings.Builder{}

	// Function label (prefix with _ to avoid conflicts)
	builder.WriteString(fmt.Sprintf(".align 4\n"))
	builder.WriteString(fmt.Sprintf("_%s:\n", fn.Name))

	// Create context for this function
	ctx := newCodeGenContext(stringMap, sourceLines)

	// Count locals (parameters + variables in body)
	paramCount := len(fn.Parameters)
	varCount := countVariables(fn.Body.Statements)
	totalLocals := paramCount + varCount

	// Function prologue - always save frame pointer and link register
	builder.WriteString("    stp x29, x30, [sp, #-16]!\n")
	builder.WriteString("    mov x29, sp\n")

	// Allocate stack space for locals if needed
	if totalLocals > 0 {
		stackSize := totalLocals * 16
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", stackSize))
	}

	// Store parameters from registers to stack
	for i, param := range fn.Parameters {
		offset := ctx.declareVariable(param.Name)
		// Parameters come in x0-x7
		builder.WriteString(fmt.Sprintf("    str x%d, [x29, #-%d]\n", i, offset))
	}

	// Generate code for function body
	for _, stmt := range fn.Body.Statements {
		// Add source line comment
		builder.WriteString(ctx.getSourceLineComment(stmt.Pos()))
		code, err := GenerateStmt(stmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// If this is main and we didn't have an explicit return, add default return
	if fn.Name == "main" && fn.ReturnType == "void" {
		builder.WriteString("    mov x0, #0\n")
	}

	// Function epilogue
	if totalLocals > 0 {
		builder.WriteString("    mov sp, x29\n")
	}
	builder.WriteString("    ldp x29, x30, [sp], #16\n")
	builder.WriteString("    ret\n")

	return builder.String(), nil
}

// GenerateStmt generates code for a statement
func GenerateStmt(stmt ast.Statement, ctx *CodeGenContext) (string, error) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return GenerateExprWithContext(s.Expr, ctx)
	case *ast.VarDeclStmt:
		return GenerateVarDecl(s, ctx)
	case *ast.AssignStmt:
		return GenerateAssignStmt(s, ctx)
	case *ast.ReturnStmt:
		return GenerateReturnStmt(s, ctx)
	default:
		return "", fmt.Errorf("unknown statement type: %T", s)
	}
}

// GenerateReturnStmt generates code for a return statement
func GenerateReturnStmt(stmt *ast.ReturnStmt, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	if stmt.Value != nil {
		// Evaluate return expression (result in x2)
		code, err := GenerateExprWithContext(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		// Move result to x0 (return register)
		builder.WriteString("    mov x0, x2\n")
	}

	// Epilogue and return
	builder.WriteString("    mov sp, x29\n")
	builder.WriteString("    ldp x29, x30, [sp], #16\n")
	builder.WriteString("    ret\n")

	return builder.String(), nil
}

// GenerateCallExpr generates code for a function call expression
func GenerateCallExpr(call *ast.CallExpr, ctx *CodeGenContext) (string, error) {
	// Check for built-in functions first
	if _, isBuiltin := semantic.Builtins[call.Name]; isBuiltin {
		return generateBuiltinCallAST(call, ctx)
	}

	builder := strings.Builder{}

	// Evaluate each argument and store on stack temporarily
	// We need to do this because evaluating arguments might clobber registers
	argCount := len(call.Arguments)

	if argCount > 0 {
		// Save space for arguments on stack
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", argCount*16))

		// Evaluate each argument and store on stack
		for i, arg := range call.Arguments {
			code, err := GenerateExprWithContext(arg, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
			// Store result (in x2) to stack
			builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", i*16))
		}

		// Load arguments from stack into registers x0-x7
		for i := 0; i < argCount && i < 8; i++ {
			builder.WriteString(fmt.Sprintf("    ldr x%d, [sp, #%d]\n", i, i*16))
		}

		// Restore stack pointer
		builder.WriteString(fmt.Sprintf("    add sp, sp, #%d\n", argCount*16))
	}

	// Call the function
	builder.WriteString(fmt.Sprintf("    bl _%s\n", call.Name))

	// Result is in x0, move to x2 (our convention)
	builder.WriteString("    mov x2, x0\n")

	return builder.String(), nil
}

// generateBuiltinCallAST generates code for a built-in function call (AST version)
func generateBuiltinCallAST(call *ast.CallExpr, ctx *CodeGenContext) (string, error) {
	switch call.Name {
	case "exit":
		return generateExitBuiltinAST(call, ctx)
	case "print":
		return generatePrintBuiltinAST(call, ctx)
	default:
		return "", fmt.Errorf("unknown built-in function: %s", call.Name)
	}
}

// generateExitBuiltinAST generates code for the exit() built-in (AST version)
func generateExitBuiltinAST(call *ast.CallExpr, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	// Generate code for the exit code argument (result in x2)
	if len(call.Arguments) > 0 {
		code, err := GenerateExprWithContext(call.Arguments[0], ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Move exit code from x2 to x0
	builder.WriteString("    mov x0, x2\n")
	// Syscall 1 = exit on macOS
	builder.WriteString("    mov x16, #1\n")
	// Invoke syscall
	builder.WriteString("    svc #0\n")

	return builder.String(), nil
}

// generatePrintBuiltinAST generates code for the print() built-in (AST version)
func generatePrintBuiltinAST(call *ast.CallExpr, ctx *CodeGenContext) (string, error) {
	builder := strings.Builder{}

	if len(call.Arguments) == 0 {
		return "", nil
	}

	arg := call.Arguments[0]

	// Check if the argument is a string literal
	if lit, ok := arg.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralTypeString {
		// Handle string literal printing
		label, exists := ctx.stringMap[lit]
		if !exists {
			return "", fmt.Errorf("string literal not found in string map: %s", lit.Value)
		}

		// Load string address
		builder.WriteString(fmt.Sprintf("    adrp x1, %s@PAGE\n", label))
		builder.WriteString(fmt.Sprintf("    add x1, x1, %s@PAGEOFF\n", label))
		// Load string length
		builder.WriteString(fmt.Sprintf("    mov x2, #%d\n", len(lit.Value)))
		// Write to stdout (fd=1)
		builder.WriteString("    mov x0, #1\n")
		builder.WriteString("    mov x16, #4\n")
		builder.WriteString("    svc #0x80\n")
		// Write newline
		builder.WriteString(generateNewline())

		return builder.String(), nil
	}

	// Handle integer printing (existing logic)
	code, err := GenerateExprWithContext(arg, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Convert integer to string
	builder.WriteString("    mov x0, x2\n")
	builder.WriteString("    bl int_to_string\n")
	builder.WriteString("\n")

	// Write the string to stdout
	builder.WriteString(generateWriteSyscall("x0", "x1"))
	builder.WriteString("\n")

	// Write a newline character
	builder.WriteString(generateNewline())

	return builder.String(), nil
}

// generateLegacyProgram generates code for legacy top-level statement programs
func generateLegacyProgram(program *ast.Program, builder *strings.Builder, sourceLines []string) (string, error) {
	statementsToProcess := program.Statements

	// Check if we need a .data section for strings or print statements
	hasPrint := false
	stringIndex := 0
	stringMap := make(map[*ast.LiteralExpr]string)

	var collectStrings func(ast.Expression)
	collectStrings = func(expr ast.Expression) {
		switch e := expr.(type) {
		case *ast.BinaryExpr:
			collectStrings(e.Left)
			collectStrings(e.Right)
		case *ast.LiteralExpr:
			if e.Kind == ast.LiteralTypeString {
				if _, exists := stringMap[e]; !exists {
					stringMap[e] = fmt.Sprintf("str_%d", stringIndex)
					stringIndex++
				}
			}
		case *ast.CallExpr:
			if e.Name == "print" {
				hasPrint = true
			}
			for _, arg := range e.Arguments {
				collectStrings(arg)
			}
		}
	}

	for _, stmt := range statementsToProcess {
		switch s := stmt.(type) {
		case *ast.ExprStmt:
			collectStrings(s.Expr)
		}
	}

	if len(stringMap) > 0 || hasPrint {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")
		if hasPrint {
			builder.WriteString("buffer: .space 32\n")
			builder.WriteString("newline: .byte 10\n")
		}
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

	if hasPrint {
		builder.WriteString(intToStringFunctionText())
		builder.WriteString("\n")
	}

	builder.WriteString("main:\n")

	ctx := newCodeGenContext(stringMap, sourceLines)
	varCount := countVariables(statementsToProcess)

	if varCount > 0 {
		builder.WriteString("    stp x29, x30, [sp, #-16]!\n")
		builder.WriteString("    mov x29, sp\n")
		stackSize := varCount * 16
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", stackSize))
	}

	for _, stmt := range statementsToProcess {
		// Add source line comment
		builder.WriteString(ctx.getSourceLineComment(stmt.Pos()))
		code, err := GenerateStmt(stmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	if varCount > 0 {
		builder.WriteString("    mov sp, x29\n")
		builder.WriteString("    ldp x29, x30, [sp], #16\n")
	}

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

	case *ast.CallExpr:
		// Generate function call
		code, err := GenerateCallExpr(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
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
			builder.WriteString("    str x2, [sp, #-16]!\n")

			rightCode, err := GenerateExprWithContext(e.Right, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
			builder.WriteString("    mov x1, x2\n")
			builder.WriteString("    ldr x0, [sp], #16\n")

		} else if rightIsBinary {
			// Right is complex: evaluate right first, save it, then evaluate left
			rightCode, err := GenerateExprWithContext(e.Right, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
			builder.WriteString("    str x2, [sp, #-16]!\n")

			leftCode, err := generateOperandToReg(e.Left, "x0", ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(leftCode)
			builder.WriteString("    ldr x1, [sp], #16\n")

		} else if leftIsBinary {
			// Left is complex: evaluate left first, save it, then evaluate right
			leftCode, err := GenerateExprWithContext(e.Left, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(leftCode)
			builder.WriteString("    str x2, [sp, #-16]!\n")

			rightCode, err := generateOperandToReg(e.Right, "x1", ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(rightCode)
			builder.WriteString("    ldr x0, [sp], #16\n")

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

	case *ast.CallExpr:
		// For function calls, evaluate and store result
		code, err := GenerateCallExpr(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		// Result is in x2, move to target register if needed
		if reg != "x2" {
			builder.WriteString(fmt.Sprintf("    mov %s, x2\n", reg))
		}

	default:
		return "", fmt.Errorf("unsupported operand type: %T", expr)
	}

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

// generateWriteSyscall creates assembly code to write data to stdout.
// bufferReg is the register containing the buffer address.
// lengthReg is the register containing the length.
func generateWriteSyscall(bufferReg, lengthReg string) string {
	return fmt.Sprintf("    mov x2, %s\n", lengthReg) +
		fmt.Sprintf("    mov x1, %s\n", bufferReg) +
		"    mov x0, #1\n" +
		"    mov x16, #4\n" +
		"    svc #0x80\n"
}

// generateNewline creates assembly code to write a newline to stdout.
func generateNewline() string {
	return "    adrp x1, newline@PAGE\n" +
		"    add x1, x1, newline@PAGEOFF\n" +
		"    mov x2, #1\n" +
		"    mov x0, #1\n" +
		"    mov x16, #4\n" +
		"    svc #0x80\n"
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
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

`
}

// setupConversion initializes registers for the conversion process
func setupConversion() string {
	return `    adrp x19, buffer@PAGE
    add x19, x19, buffer@PAGEOFF
    mov x20, x0
    mov x21, #0

`
}

// handleZeroCase handles the special case when the input is zero
func handleZeroCase() string {
	return `    cmp x20, #0
    bne check_negative
    mov w10, #48
    strb w10, [x19]
    mov x0, x19
    mov x1, #1
    b restore_regs

`
}

// handleNegativeNumber checks for negative numbers and converts them to positive
func handleNegativeNumber() string {
	return `check_negative:
    cmp x20, #0
    bge convert_loop_setup
    mov x21, #1
    neg x20, x20

`
}

// convertDigitsToASCII converts the number to ASCII digits (backwards)
func convertDigitsToASCII() string {
	return `convert_loop_setup:
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

`
}

// addMinusSignIfNeeded adds a minus sign if the number was negative
func addMinusSignIfNeeded() string {
	return `    cmp x21, #1
    bne finalize
    mov w10, #45
    strb w10, [x19]
    sub x19, x19, #1
    add x22, x22, #1

`
}

// functionEpilogue finalizes the result and restores registers
func functionEpilogue() string {
	return `finalize:
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

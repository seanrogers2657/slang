package as

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/semantic"
)

// TypedCodeGenerator generates ARM64 assembly from a TypedProgram
type TypedCodeGenerator struct {
	program     *semantic.TypedProgram
	sourceLines []string
}

// NewTypedCodeGenerator creates a new typed code generator
func NewTypedCodeGenerator(program *semantic.TypedProgram, sourceLines []string) *TypedCodeGenerator {
	return &TypedCodeGenerator{
		program:     program,
		sourceLines: sourceLines,
	}
}

// TypedCodeGenContext tracks state during typed code generation
type TypedCodeGenContext struct {
	variables   map[string]VariableSlot
	stackOffset int
	sourceLines []string
}

// VariableSlot tracks a variable's stack location and type
type VariableSlot struct {
	Offset int
	Type   semantic.Type
}

func newTypedCodeGenContext(sourceLines []string) *TypedCodeGenContext {
	return &TypedCodeGenContext{
		variables:   make(map[string]VariableSlot),
		stackOffset: 0,
		sourceLines: sourceLines,
	}
}

func (ctx *TypedCodeGenContext) declareVariable(name string, typ semantic.Type) int {
	ctx.stackOffset += 16 // 16-byte aligned for ARM64
	ctx.variables[name] = VariableSlot{Offset: ctx.stackOffset, Type: typ}
	return ctx.stackOffset
}

func (ctx *TypedCodeGenContext) getVariable(name string) (VariableSlot, bool) {
	slot, ok := ctx.variables[name]
	return slot, ok
}

func (ctx *TypedCodeGenContext) getSourceLineComment(pos ast.Position) string {
	if ctx.sourceLines == nil || pos.Line <= 0 || pos.Line > len(ctx.sourceLines) {
		return ""
	}
	line := strings.TrimSpace(ctx.sourceLines[pos.Line-1])
	if line == "" {
		return ""
	}
	return fmt.Sprintf("// %d: %s\n", pos.Line, line)
}

// Generate generates ARM64 assembly from the typed program
func (g *TypedCodeGenerator) Generate() (string, error) {
	builder := strings.Builder{}

	if len(g.program.Declarations) > 0 {
		return g.generateFunctionBasedProgram(&builder)
	}

	return g.generateLegacyProgram(&builder)
}

func (g *TypedCodeGenerator) generateFunctionBasedProgram(builder *strings.Builder) (string, error) {
	// Collect functions
	functions := make([]*semantic.TypedFunctionDecl, 0)
	for _, decl := range g.program.Declarations {
		if fn, ok := decl.(*semantic.TypedFunctionDecl); ok {
			functions = append(functions, fn)
		}
	}

	if len(functions) == 0 {
		return "", fmt.Errorf("no functions found")
	}

	// Check for print statements and collect float literals
	hasPrint := false
	floatLiterals := make(map[string]floatLiteralInfo)
	floatIndex := 0

	for _, fn := range functions {
		g.collectFloatLiterals(fn.Body, &floatLiterals, &floatIndex)
		if g.hasPrintStatements(fn.Body) {
			hasPrint = true
		}
	}

	// Write .data section
	if len(floatLiterals) > 0 || hasPrint {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")

		if hasPrint {
			builder.WriteString("buffer: .space 32\n")
			builder.WriteString("newline: .byte 10\n")
		}

		// Float literals in .data section
		for label, info := range floatLiterals {
			if info.isF64 {
				builder.WriteString(fmt.Sprintf("%s: .double %s\n", label, info.value))
			} else {
				builder.WriteString(fmt.Sprintf("%s: .float %s\n", label, info.value))
			}
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

	if hasPrint {
		builder.WriteString(intToStringFunctionText())
		builder.WriteString("\n")
	}

	// Generate code for each function
	for _, fn := range functions {
		code, err := g.generateFunction(fn, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

type floatLiteralInfo struct {
	value string
	isF64 bool
}

func (g *TypedCodeGenerator) collectFloatLiterals(block *semantic.TypedBlockStmt, literals *map[string]floatLiteralInfo, index *int) {
	for _, stmt := range block.Statements {
		g.collectFloatLiteralsFromStmt(stmt, literals, index)
	}
}

func (g *TypedCodeGenerator) collectFloatLiteralsFromStmt(stmt semantic.TypedStatement, literals *map[string]floatLiteralInfo, index *int) {
	switch s := stmt.(type) {
	case *semantic.TypedExprStmt:
		g.collectFloatLiteralsFromExpr(s.Expr, literals, index)
	case *semantic.TypedVarDeclStmt:
		g.collectFloatLiteralsFromExpr(s.Initializer, literals, index)
	case *semantic.TypedAssignStmt:
		g.collectFloatLiteralsFromExpr(s.Value, literals, index)
	case *semantic.TypedReturnStmt:
		if s.Value != nil {
			g.collectFloatLiteralsFromExpr(s.Value, literals, index)
		}
	}
}

func (g *TypedCodeGenerator) collectFloatLiteralsFromExpr(expr semantic.TypedExpression, literals *map[string]floatLiteralInfo, index *int) {
	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		if e.LitType == ast.LiteralTypeFloat {
			label := fmt.Sprintf("float_%d", *index)
			(*index)++
			_, isF64 := e.Type.(semantic.F64Type)
			(*literals)[label] = floatLiteralInfo{value: e.Value, isF64: isF64}
		}
	case *semantic.TypedBinaryExpr:
		g.collectFloatLiteralsFromExpr(e.Left, literals, index)
		g.collectFloatLiteralsFromExpr(e.Right, literals, index)
	case *semantic.TypedCallExpr:
		for _, arg := range e.Arguments {
			g.collectFloatLiteralsFromExpr(arg, literals, index)
		}
	}
}

func (g *TypedCodeGenerator) hasPrintStatements(block *semantic.TypedBlockStmt) bool {
	for _, stmt := range block.Statements {
		if exprStmt, ok := stmt.(*semantic.TypedExprStmt); ok {
			if g.hasPrintCall(exprStmt.Expr) {
				return true
			}
		}
	}
	return false
}

func (g *TypedCodeGenerator) hasPrintCall(expr semantic.TypedExpression) bool {
	switch e := expr.(type) {
	case *semantic.TypedCallExpr:
		if e.Name == "print" {
			return true
		}
		for _, arg := range e.Arguments {
			if g.hasPrintCall(arg) {
				return true
			}
		}
	case *semantic.TypedBinaryExpr:
		return g.hasPrintCall(e.Left) || g.hasPrintCall(e.Right)
	}
	return false
}

func (g *TypedCodeGenerator) generateFunction(fn *semantic.TypedFunctionDecl, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	builder.WriteString(".align 4\n")
	builder.WriteString(fmt.Sprintf("_%s:\n", fn.Name))

	ctx := newTypedCodeGenContext(g.sourceLines)

	// Count locals
	paramCount := len(fn.Parameters)
	varCount := g.countVariables(fn.Body.Statements)
	totalLocals := paramCount + varCount

	// Function prologue
	builder.WriteString("    stp x29, x30, [sp, #-16]!\n")
	builder.WriteString("    mov x29, sp\n")

	if totalLocals > 0 {
		stackSize := totalLocals * 16
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", stackSize))
	}

	// Store parameters
	for i, param := range fn.Parameters {
		offset := ctx.declareVariable(param.Name, param.Type)
		builder.WriteString(fmt.Sprintf("    str x%d, [x29, #-%d]\n", i, offset))
	}

	// Generate body
	for _, stmt := range fn.Body.Statements {
		builder.WriteString(ctx.getSourceLineComment(stmt.Pos()))
		code, err := g.generateStmt(stmt, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Default return for main
	if fn.Name == "main" {
		if _, isVoid := fn.ReturnType.(semantic.VoidType); isVoid {
			builder.WriteString("    mov x0, #0\n")
		}
	}

	// Function epilogue
	if totalLocals > 0 {
		builder.WriteString("    mov sp, x29\n")
	}
	builder.WriteString("    ldp x29, x30, [sp], #16\n")
	builder.WriteString("    ret\n")

	return builder.String(), nil
}

func (g *TypedCodeGenerator) countVariables(stmts []semantic.TypedStatement) int {
	count := 0
	for _, stmt := range stmts {
		if _, ok := stmt.(*semantic.TypedVarDeclStmt); ok {
			count++
		}
	}
	return count
}

func (g *TypedCodeGenerator) generateStmt(stmt semantic.TypedStatement, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	switch s := stmt.(type) {
	case *semantic.TypedExprStmt:
		return g.generateExpr(s.Expr, ctx, floatLiterals)
	case *semantic.TypedVarDeclStmt:
		return g.generateVarDecl(s, ctx, floatLiterals)
	case *semantic.TypedAssignStmt:
		return g.generateAssignStmt(s, ctx, floatLiterals)
	case *semantic.TypedReturnStmt:
		return g.generateReturnStmt(s, ctx, floatLiterals)
	default:
		return "", fmt.Errorf("unknown statement type: %T", s)
	}
}

func (g *TypedCodeGenerator) generateVarDecl(stmt *semantic.TypedVarDeclStmt, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	// Generate initializer
	code, err := g.generateExpr(stmt.Initializer, ctx, floatLiterals)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Allocate stack slot
	offset := ctx.declareVariable(stmt.Name, stmt.DeclaredType)

	// Store value based on type
	if semantic.IsFloatType(stmt.DeclaredType) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", offset))
	} else {
		builder.WriteString(fmt.Sprintf("    str x2, [x29, #-%d]\n", offset))
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateAssignStmt(stmt *semantic.TypedAssignStmt, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	code, err := g.generateExpr(stmt.Value, ctx, floatLiterals)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	slot, ok := ctx.getVariable(stmt.Name)
	if !ok {
		return "", fmt.Errorf("undefined variable: %s", stmt.Name)
	}

	if semantic.IsFloatType(slot.Type) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", slot.Offset))
	} else {
		builder.WriteString(fmt.Sprintf("    str x2, [x29, #-%d]\n", slot.Offset))
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateReturnStmt(stmt *semantic.TypedReturnStmt, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	if stmt.Value != nil {
		code, err := g.generateExpr(stmt.Value, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)

		if semantic.IsFloatType(stmt.Value.GetType()) {
			// Float return value in d0
			builder.WriteString("    fmov x0, d0\n")
		} else {
			builder.WriteString("    mov x0, x2\n")
		}
	}

	builder.WriteString("    mov sp, x29\n")
	builder.WriteString("    ldp x29, x30, [sp], #16\n")
	builder.WriteString("    ret\n")

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateExpr(expr semantic.TypedExpression, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		return g.generateLiteral(e, ctx, floatLiterals)

	case *semantic.TypedIdentifierExpr:
		slot, ok := ctx.getVariable(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		if semantic.IsFloatType(slot.Type) {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x29, #-%d]\n", slot.Offset))
		} else {
			builder.WriteString(fmt.Sprintf("    ldr x2, [x29, #-%d]\n", slot.Offset))
		}
		return builder.String(), nil

	case *semantic.TypedCallExpr:
		return g.generateCallExpr(e, ctx, floatLiterals)

	case *semantic.TypedBinaryExpr:
		return g.generateBinaryExpr(e, ctx, floatLiterals)

	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func (g *TypedCodeGenerator) generateLiteral(lit *semantic.TypedLiteralExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	if lit.LitType == ast.LiteralTypeFloat {
		// Find the label for this float literal
		for label, info := range floatLiterals {
			if info.value == lit.Value {
				_, isF64 := lit.Type.(semantic.F64Type)
				if isF64 {
					builder.WriteString(fmt.Sprintf("    adrp x8, %s@PAGE\n", label))
					builder.WriteString(fmt.Sprintf("    ldr d0, [x8, %s@PAGEOFF]\n", label))
				} else {
					builder.WriteString(fmt.Sprintf("    adrp x8, %s@PAGE\n", label))
					builder.WriteString(fmt.Sprintf("    ldr s0, [x8, %s@PAGEOFF]\n", label))
					builder.WriteString("    fcvt d0, s0\n") // promote to d0 for operations
				}
				return builder.String(), nil
			}
		}
		return "", fmt.Errorf("float literal not found in data section: %s", lit.Value)
	}

	// Integer literal - use appropriate instruction based on type
	builder.WriteString(fmt.Sprintf("    mov x2, #%s\n", lit.Value))

	// Sign extend for smaller signed types
	switch lit.Type.(type) {
	case semantic.I8Type:
		builder.WriteString("    sxtb x2, w2\n")
	case semantic.I16Type:
		builder.WriteString("    sxth x2, w2\n")
	case semantic.I32Type:
		builder.WriteString("    sxtw x2, w2\n")
	case semantic.U8Type:
		builder.WriteString("    and x2, x2, #0xFF\n")
	case semantic.U16Type:
		builder.WriteString("    and x2, x2, #0xFFFF\n")
	case semantic.U32Type:
		builder.WriteString("    mov w2, w2\n") // zero-extends to x2
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	isFloat := semantic.IsFloatType(expr.Type)

	if isFloat {
		return g.generateFloatBinaryExpr(expr, ctx, floatLiterals)
	}

	return g.generateIntBinaryExpr(expr, ctx, floatLiterals, &builder)
}

func (g *TypedCodeGenerator) generateIntBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo, builder *strings.Builder) (string, error) {
	// Check if operands are complex
	_, leftIsBinary := expr.Left.(*semantic.TypedBinaryExpr)
	_, rightIsBinary := expr.Right.(*semantic.TypedBinaryExpr)

	if leftIsBinary && rightIsBinary {
		leftCode, err := g.generateExpr(expr.Left, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)
		builder.WriteString("    str x2, [sp, #-16]!\n")

		rightCode, err := g.generateExpr(expr.Right, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
		builder.WriteString("    mov x1, x2\n")
		builder.WriteString("    ldr x0, [sp], #16\n")

	} else if rightIsBinary {
		rightCode, err := g.generateExpr(expr.Right, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
		builder.WriteString("    str x2, [sp, #-16]!\n")

		leftCode, err := g.generateOperandToReg(expr.Left, "x0", ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)
		builder.WriteString("    ldr x1, [sp], #16\n")

	} else if leftIsBinary {
		leftCode, err := g.generateExpr(expr.Left, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)
		builder.WriteString("    str x2, [sp, #-16]!\n")

		rightCode, err := g.generateOperandToReg(expr.Right, "x1", ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
		builder.WriteString("    ldr x0, [sp], #16\n")

	} else {
		leftCode, err := g.generateOperandToReg(expr.Left, "x0", ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(leftCode)

		rightCode, err := g.generateOperandToReg(expr.Right, "x1", ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)
	}

	// Generate operation based on signedness
	isSigned := true
	if numType, ok := expr.Type.(semantic.NumericType); ok {
		isSigned = numType.IsSigned()
	}

	switch expr.Op {
	case "+":
		builder.WriteString("    add x2, x0, x1\n")
	case "-":
		builder.WriteString("    sub x2, x0, x1\n")
	case "*":
		builder.WriteString("    mul x2, x0, x1\n")
	case "/":
		if isSigned {
			builder.WriteString("    sdiv x2, x0, x1\n")
		} else {
			builder.WriteString("    udiv x2, x0, x1\n")
		}
	case "%":
		if isSigned {
			builder.WriteString("    sdiv x3, x0, x1\n")
		} else {
			builder.WriteString("    udiv x3, x0, x1\n")
		}
		builder.WriteString("    msub x2, x3, x1, x0\n")
	case "==":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, eq\n")
	case "!=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, ne\n")
	case "<":
		builder.WriteString("    cmp x0, x1\n")
		if isSigned {
			builder.WriteString("    cset x2, lt\n")
		} else {
			builder.WriteString("    cset x2, lo\n")
		}
	case ">":
		builder.WriteString("    cmp x0, x1\n")
		if isSigned {
			builder.WriteString("    cset x2, gt\n")
		} else {
			builder.WriteString("    cset x2, hi\n")
		}
	case "<=":
		builder.WriteString("    cmp x0, x1\n")
		if isSigned {
			builder.WriteString("    cset x2, le\n")
		} else {
			builder.WriteString("    cset x2, ls\n")
		}
	case ">=":
		builder.WriteString("    cmp x0, x1\n")
		if isSigned {
			builder.WriteString("    cset x2, ge\n")
		} else {
			builder.WriteString("    cset x2, hs\n")
		}
	default:
		return "", fmt.Errorf("unsupported operation: %s", expr.Op)
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateFloatBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	// Evaluate left operand into d0
	leftCode, err := g.generateExpr(expr.Left, ctx, floatLiterals)
	if err != nil {
		return "", err
	}
	builder.WriteString(leftCode)
	builder.WriteString("    fmov d1, d0\n") // save left to d1

	// Evaluate right operand into d0
	rightCode, err := g.generateExpr(expr.Right, ctx, floatLiterals)
	if err != nil {
		return "", err
	}
	builder.WriteString(rightCode)
	// Now d1 = left, d0 = right

	switch expr.Op {
	case "+":
		builder.WriteString("    fadd d0, d1, d0\n")
	case "-":
		builder.WriteString("    fsub d0, d1, d0\n")
	case "*":
		builder.WriteString("    fmul d0, d1, d0\n")
	case "/":
		builder.WriteString("    fdiv d0, d1, d0\n")
	case "==":
		builder.WriteString("    fcmp d1, d0\n")
		builder.WriteString("    cset x2, eq\n")
	case "!=":
		builder.WriteString("    fcmp d1, d0\n")
		builder.WriteString("    cset x2, ne\n")
	case "<":
		builder.WriteString("    fcmp d1, d0\n")
		builder.WriteString("    cset x2, mi\n")
	case ">":
		builder.WriteString("    fcmp d1, d0\n")
		builder.WriteString("    cset x2, gt\n")
	case "<=":
		builder.WriteString("    fcmp d1, d0\n")
		builder.WriteString("    cset x2, ls\n")
	case ">=":
		builder.WriteString("    fcmp d1, d0\n")
		builder.WriteString("    cset x2, ge\n")
	default:
		return "", fmt.Errorf("unsupported float operation: %s", expr.Op)
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateOperandToReg(expr semantic.TypedExpression, reg string, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		if e.LitType == ast.LiteralTypeInteger {
			builder.WriteString(fmt.Sprintf("    mov %s, #%s\n", reg, e.Value))
		}

	case *semantic.TypedIdentifierExpr:
		slot, ok := ctx.getVariable(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		builder.WriteString(fmt.Sprintf("    ldr %s, [x29, #-%d]\n", reg, slot.Offset))

	case *semantic.TypedBinaryExpr:
		code, err := g.generateExpr(e, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			builder.WriteString(fmt.Sprintf("    mov %s, x2\n", reg))
		}

	case *semantic.TypedCallExpr:
		code, err := g.generateCallExpr(e, ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			builder.WriteString(fmt.Sprintf("    mov %s, x2\n", reg))
		}

	default:
		return "", fmt.Errorf("unsupported operand type: %T", expr)
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateCallExpr(call *semantic.TypedCallExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	// Check for built-in functions first
	if _, isBuiltin := semantic.Builtins[call.Name]; isBuiltin {
		return g.generateBuiltinCall(call, ctx, floatLiterals)
	}

	builder := strings.Builder{}

	argCount := len(call.Arguments)

	if argCount > 0 {
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", argCount*16))

		for i, arg := range call.Arguments {
			code, err := g.generateExpr(arg, ctx, floatLiterals)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
			builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", i*16))
		}

		for i := 0; i < argCount && i < 8; i++ {
			builder.WriteString(fmt.Sprintf("    ldr x%d, [sp, #%d]\n", i, i*16))
		}

		builder.WriteString(fmt.Sprintf("    add sp, sp, #%d\n", argCount*16))
	}

	builder.WriteString(fmt.Sprintf("    bl _%s\n", call.Name))
	builder.WriteString("    mov x2, x0\n")

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateBuiltinCall(call *semantic.TypedCallExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	switch call.Name {
	case "exit":
		return g.generateExitBuiltin(call, ctx, floatLiterals)
	case "print":
		return g.generatePrintBuiltin(call, ctx, floatLiterals)
	default:
		return "", fmt.Errorf("unknown built-in function: %s", call.Name)
	}
}

func (g *TypedCodeGenerator) generateExitBuiltin(call *semantic.TypedCallExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	// Generate code for the exit code argument (result in x2)
	if len(call.Arguments) > 0 {
		code, err := g.generateExpr(call.Arguments[0], ctx, floatLiterals)
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

func (g *TypedCodeGenerator) generatePrintBuiltin(call *semantic.TypedCallExpr, ctx *TypedCodeGenContext, floatLiterals map[string]floatLiteralInfo) (string, error) {
	builder := strings.Builder{}

	// Generate code for the argument (result in x2)
	if len(call.Arguments) > 0 {
		code, err := g.generateExpr(call.Arguments[0], ctx, floatLiterals)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Convert integer to string
	builder.WriteString("    mov x0, x2\n")
	builder.WriteString("    bl int_to_string\n\n")

	// Write the string to stdout
	builder.WriteString(generateWriteSyscall("x0", "x1"))
	builder.WriteString("\n")

	// Write a newline character
	builder.WriteString(generateNewline())

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateLegacyProgram(builder *strings.Builder) (string, error) {
	// Legacy programs not supported with typed codegen yet
	return "", fmt.Errorf("legacy programs not supported with typed code generator")
}

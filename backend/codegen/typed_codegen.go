package codegen

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/semantic"
)

// TypedCodeGenerator generates ARM64 assembly from a type-checked TypedProgram.
// It handles type-aware code generation including:
//   - Proper register selection (x registers for integers, d registers for floats)
//   - Signed vs unsigned operations (sdiv vs udiv, etc.)
//   - Float literals in the data section
//   - String literal handling
type TypedCodeGenerator struct {
	program     *semantic.TypedProgram
	sourceLines []string
	info        *ProgramInfo
}

// NewTypedCodeGenerator creates a new typed code generator.
func NewTypedCodeGenerator(program *semantic.TypedProgram, sourceLines []string) *TypedCodeGenerator {
	return &TypedCodeGenerator{
		program:     program,
		sourceLines: sourceLines,
	}
}

// Generate produces ARM64 assembly code from the typed program.
func (g *TypedCodeGenerator) Generate() (string, error) {
	builder := strings.Builder{}

	if len(g.program.Declarations) > 0 {
		return g.generateFunctionBasedProgram(&builder)
	}

	return g.generateLegacyProgram(&builder)
}

func (g *TypedCodeGenerator) generateFunctionBasedProgram(builder *strings.Builder) (string, error) {
	functions := make([]*semantic.TypedFunctionDecl, 0)
	for _, decl := range g.program.Declarations {
		if fn, ok := decl.(*semantic.TypedFunctionDecl); ok {
			functions = append(functions, fn)
		}
	}

	if len(functions) == 0 {
		return "", fmt.Errorf("no functions found")
	}

	// Collect literals and detect print usage
	g.info = NewProgramInfo()
	for _, fn := range functions {
		g.info.CollectFromTypedFunction(fn)
	}

	// Write .data section if needed
	if len(g.info.FloatLiterals) > 0 || len(g.info.StringLiterals) > 0 || g.info.HasPrint {
		EmitDataSection(builder, g.info.HasPrint)

		// Float literals
		for label, lit := range g.info.FloatLiterals {
			if lit.IsF64 {
				builder.WriteString(fmt.Sprintf("%s: .double %s\n", label, lit.Value))
			} else {
				builder.WriteString(fmt.Sprintf("%s: .float %s\n", label, lit.Value))
			}
		}

		// String literals
		for _, lit := range g.info.StringLiterals {
			escapedStr := EscapeStringForAsm(lit.Value)
			builder.WriteString(fmt.Sprintf("%s: .asciz \"%s\"\n", lit.Label, escapedStr))
			builder.WriteString(fmt.Sprintf("%s_len = %d\n", lit.Label, lit.Length))
		}

		builder.WriteString("\n.text\n")
	}

	EmitProgramEntry(builder)

	if g.info.HasPrint {
		builder.WriteString(intToStringFunctionText())
		builder.WriteString("\n")
	}

	for _, fn := range functions {
		code, err := g.generateFunction(fn)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateFunction(fn *semantic.TypedFunctionDecl) (string, error) {
	builder := strings.Builder{}

	EmitFunctionLabel(&builder, fn.Name)

	ctx := NewBaseContext(g.sourceLines)

	paramCount := len(fn.Parameters)
	varCount := CountTypedVariables(fn.Body.Statements)
	totalLocals := paramCount + varCount
	stackSize := totalLocals * StackAlignment

	EmitFunctionPrologue(&builder, stackSize)

	// Store parameters
	for i, param := range fn.Parameters {
		offset := ctx.DeclareVariable(param.Name, param.Type)
		EmitStoreToStack(&builder, fmt.Sprintf("x%d", i), offset)
	}

	// Generate body
	for _, stmt := range fn.Body.Statements {
		builder.WriteString(ctx.GetSourceLineComment(stmt.Pos()))
		code, err := g.generateStmt(stmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Default return for void main
	if fn.Name == "main" {
		if _, isVoid := fn.ReturnType.(semantic.VoidType); isVoid {
			EmitMoveImm(&builder, "x0", "0")
		}
	}

	EmitFunctionEpilogue(&builder, totalLocals > 0)

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateStmt(stmt semantic.TypedStatement, ctx *BaseContext) (string, error) {
	switch s := stmt.(type) {
	case *semantic.TypedExprStmt:
		return g.generateExpr(s.Expr, ctx)
	case *semantic.TypedVarDeclStmt:
		return g.generateVarDecl(s, ctx)
	case *semantic.TypedAssignStmt:
		return g.generateAssignStmt(s, ctx)
	case *semantic.TypedReturnStmt:
		return g.generateReturnStmt(s, ctx)
	default:
		return "", fmt.Errorf("unknown statement type: %T", s)
	}
}

func (g *TypedCodeGenerator) generateVarDecl(stmt *semantic.TypedVarDeclStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	code, err := g.generateExpr(stmt.Initializer, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	offset := ctx.DeclareVariable(stmt.Name, stmt.DeclaredType)

	if semantic.IsFloatType(stmt.DeclaredType) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", offset))
	} else {
		EmitStoreToStack(&builder, "x2", offset)
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateAssignStmt(stmt *semantic.TypedAssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	code, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	slot, ok := ctx.GetVariable(stmt.Name)
	if !ok {
		return "", fmt.Errorf("undefined variable: %s", stmt.Name)
	}

	if semantic.IsFloatType(slot.Type) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", slot.Offset))
	} else {
		EmitStoreToStack(&builder, "x2", slot.Offset)
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateReturnStmt(stmt *semantic.TypedReturnStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if stmt.Value != nil {
		code, err := g.generateExpr(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)

		if semantic.IsFloatType(stmt.Value.GetType()) {
			builder.WriteString("    fmov x0, d0\n")
		} else {
			EmitMoveReg(&builder, "x0", "x2")
		}
	}

	EmitReturnEpilogue(&builder)
	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateExpr(expr semantic.TypedExpression, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		return g.generateLiteral(e)

	case *semantic.TypedIdentifierExpr:
		slot, ok := ctx.GetVariable(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		if semantic.IsFloatType(slot.Type) {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x29, #-%d]\n", slot.Offset))
		} else {
			EmitLoadFromStack(&builder, "x2", slot.Offset)
		}
		return builder.String(), nil

	case *semantic.TypedCallExpr:
		return g.generateCallExpr(e, ctx)

	case *semantic.TypedBinaryExpr:
		return g.generateBinaryExpr(e, ctx)

	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func (g *TypedCodeGenerator) generateLiteral(lit *semantic.TypedLiteralExpr) (string, error) {
	builder := strings.Builder{}

	if lit.LitType == ast.LiteralTypeFloat {
		label, _, found := g.info.FindFloatLiteral(lit.Value)
		if !found {
			return "", fmt.Errorf("float literal not found in data section: %s", lit.Value)
		}

		_, isF64 := lit.Type.(semantic.F64Type)
		builder.WriteString(fmt.Sprintf("    adrp x8, %s@PAGE\n", label))
		if isF64 {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x8, %s@PAGEOFF]\n", label))
		} else {
			builder.WriteString(fmt.Sprintf("    ldr s0, [x8, %s@PAGEOFF]\n", label))
			builder.WriteString("    fcvt d0, s0\n")
		}
		return builder.String(), nil
	}

	if lit.LitType == ast.LiteralTypeString {
		// String literals are handled by print
		return "", nil
	}

	// Integer literal
	EmitMoveImm(&builder, "x2", lit.Value)

	// Sign extend for smaller types
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
		builder.WriteString("    mov w2, w2\n")
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	if semantic.IsFloatType(expr.Type) {
		return g.generateFloatBinaryExpr(expr, ctx)
	}
	return g.generateIntBinaryExpr(expr, ctx)
}

func (g *TypedCodeGenerator) generateIntBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	_, leftIsBinary := expr.Left.(*semantic.TypedBinaryExpr)
	_, rightIsBinary := expr.Right.(*semantic.TypedBinaryExpr)

	eval := &BinaryExprEvaluator{
		LeftIsComplex:  leftIsBinary,
		RightIsComplex: rightIsBinary,
		GenerateLeft: func() (string, error) {
			return g.generateExpr(expr.Left, ctx)
		},
		GenerateRight: func() (string, error) {
			return g.generateExpr(expr.Right, ctx)
		},
		GenerateLeftToReg: func(reg string) (string, error) {
			return g.generateOperandToReg(expr.Left, reg, ctx)
		},
		GenerateRightToReg: func(reg string) (string, error) {
			return g.generateOperandToReg(expr.Right, reg, ctx)
		},
	}

	setupCode, err := EmitBinaryExprSetup(eval)
	if err != nil {
		return "", err
	}
	builder.WriteString(setupCode)

	// Determine signedness
	isSigned := true
	if numType, ok := expr.Type.(semantic.NumericType); ok {
		isSigned = numType.IsSigned()
	}

	opCode, err := IntOperation(expr.Op, isSigned)
	if err != nil {
		return "", err
	}
	builder.WriteString(opCode)

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateFloatBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	setupCode, err := EmitFloatBinaryExprSetup(
		func() (string, error) { return g.generateExpr(expr.Left, ctx) },
		func() (string, error) { return g.generateExpr(expr.Right, ctx) },
	)
	if err != nil {
		return "", err
	}
	builder.WriteString(setupCode)

	opCode, err := FloatOperation(expr.Op)
	if err != nil {
		return "", err
	}
	builder.WriteString(opCode)

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateOperandToReg(expr semantic.TypedExpression, reg string, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		if e.LitType == ast.LiteralTypeInteger {
			EmitMoveImm(&builder, reg, e.Value)
		}

	case *semantic.TypedIdentifierExpr:
		slot, ok := ctx.GetVariable(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		EmitLoadFromStack(&builder, reg, slot.Offset)

	case *semantic.TypedBinaryExpr:
		code, err := g.generateExpr(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	case *semantic.TypedCallExpr:
		code, err := g.generateCallExpr(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	default:
		return "", fmt.Errorf("unsupported operand type: %T", expr)
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateCallExpr(call *semantic.TypedCallExpr, ctx *BaseContext) (string, error) {
	if _, isBuiltin := semantic.Builtins[call.Name]; isBuiltin {
		return g.generateBuiltinCall(call, ctx)
	}

	builder := strings.Builder{}

	argCount := len(call.Arguments)
	if argCount > 0 {
		code, err := EmitCallSetup(argCount, func(i int) (string, error) {
			return g.generateExpr(call.Arguments[i], ctx)
		})
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	EmitBranchLink(&builder, call.Name)
	EmitMoveReg(&builder, "x2", "x0")

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateBuiltinCall(call *semantic.TypedCallExpr, ctx *BaseContext) (string, error) {
	switch call.Name {
	case "exit":
		return g.generateExitBuiltin(call, ctx)
	case "print":
		return g.generatePrintBuiltin(call, ctx)
	default:
		return "", fmt.Errorf("unknown built-in function: %s", call.Name)
	}
}

func (g *TypedCodeGenerator) generateExitBuiltin(call *semantic.TypedCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if len(call.Arguments) > 0 {
		code, err := g.generateExpr(call.Arguments[0], ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	EmitExitSyscall(&builder)
	return builder.String(), nil
}

func (g *TypedCodeGenerator) generatePrintBuiltin(call *semantic.TypedCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if len(call.Arguments) == 0 {
		return "", nil
	}

	arg := call.Arguments[0]
	argType := arg.GetType()

	// Check if argument is a string
	if _, isString := argType.(semantic.StringType); isString {
		if lit, ok := arg.(*semantic.TypedLiteralExpr); ok {
			info, found := g.info.FindStringLiteral(lit.Value)
			if !found {
				return "", fmt.Errorf("string literal not found in data section: %s", lit.Value)
			}

			EmitLoadAddress(&builder, "x1", info.Label)
			EmitMoveImm(&builder, "x2", fmt.Sprintf("%d", info.Length))
			builder.WriteString("    mov x0, #1\n")
			builder.WriteString("    mov x16, #4\n")
			builder.WriteString("    svc #0x80\n")
			EmitNewline(&builder)
			return builder.String(), nil
		}
		return "", fmt.Errorf("only string literals are supported for print")
	}

	// Handle integer printing
	code, err := g.generateExpr(arg, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	EmitPrintInt(&builder)
	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateLegacyProgram(builder *strings.Builder) (string, error) {
	return "", fmt.Errorf("legacy programs not supported with typed code generator")
}

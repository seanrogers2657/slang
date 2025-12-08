package codegen

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// AsGenerator defines the interface for ARM64 assembly code generators.
// Implementations convert parsed programs into ARM64 assembly targeting macOS.
type AsGenerator interface {
	// Generate produces ARM64 assembly code as a string.
	// Returns an error if code generation fails.
	Generate() (string, error)
}

// NewAsGenerator creates a new AST-based code generator.
// This generator works with untyped AST programs and is primarily used
// for simple programs. For type-aware code generation, use NewTypedCodeGenerator.
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
	if len(program.Declarations) == 0 {
		return "", fmt.Errorf("no declarations found: programs must have at least one function")
	}

	builder := strings.Builder{}
	return generateFunctionBasedProgram(program, &builder, sourceLines)
}

func generateFunctionBasedProgram(program *ast.Program, builder *strings.Builder, sourceLines []string) (string, error) {
	functions := make([]*ast.FunctionDecl, 0)
	for _, decl := range program.Declarations {
		if fn, ok := decl.(*ast.FunctionDecl); ok {
			functions = append(functions, fn)
		}
	}

	if len(functions) == 0 {
		return "", fmt.Errorf("no functions found")
	}

	// Collect literals and detect print usage
	info := NewProgramInfo()
	stringMap := make(map[*ast.LiteralExpr]string)

	for _, fn := range functions {
		fnStringMap := info.CollectFromASTFunction(fn)
		for k, v := range fnStringMap {
			stringMap[k] = v
		}
	}

	// Write .data section if needed
	if len(info.StringLiterals) > 0 || info.HasPrint {
		EmitDataSection(builder, info.HasPrint)
		for _, lit := range info.StringLiterals {
			builder.WriteString(fmt.Sprintf("%s:\n", lit.Label))
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", lit.Value))
		}
		builder.WriteString("\n.text\n")
	}

	EmitProgramEntry(builder)

	if info.HasPrint {
		builder.WriteString(intToStringFunctionText())
		builder.WriteString("\n")
	}

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

	EmitFunctionLabel(&builder, fn.Name)

	ctx := NewBaseContext(sourceLines)
	ctx.SetStringMap(stringMap)

	paramCount := len(fn.Parameters)
	varCount := CountVariables(fn.Body.Statements)
	totalLocals := paramCount + varCount
	stackSize := totalLocals * StackAlignment

	EmitFunctionPrologue(&builder, stackSize)

	// Store parameters from registers to stack
	for i, param := range fn.Parameters {
		offset := ctx.DeclareVariable(param.Name, nil)
		EmitStoreToStack(&builder, fmt.Sprintf("x%d", i), offset)
	}

	// Generate code for function body
	for _, stmt := range fn.Body.Statements {
		builder.WriteString(ctx.GetSourceLineComment(stmt.Pos()))
		code, err := GenerateStmt(stmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Default return for void main
	if fn.Name == "main" && fn.ReturnType == "void" {
		EmitMoveImm(&builder, "x0", "0")
	}

	EmitFunctionEpilogue(&builder, totalLocals > 0)

	return builder.String(), nil
}

// GenerateStmt generates code for a statement
func GenerateStmt(stmt ast.Statement, ctx *BaseContext) (string, error) {
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
func GenerateReturnStmt(stmt *ast.ReturnStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if stmt.Value != nil {
		code, err := GenerateExprWithContext(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		EmitMoveReg(&builder, "x0", "x2")
	}

	EmitReturnEpilogue(&builder)
	return builder.String(), nil
}

// GenerateCallExpr generates code for a function call expression
func GenerateCallExpr(call *ast.CallExpr, ctx *BaseContext) (string, error) {
	if _, isBuiltin := semantic.Builtins[call.Name]; isBuiltin {
		return generateBuiltinCallAST(call, ctx)
	}

	builder := strings.Builder{}

	argCount := len(call.Arguments)
	if argCount > 0 {
		code, err := EmitCallSetup(argCount, func(i int) (string, error) {
			return GenerateExprWithContext(call.Arguments[i], ctx)
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

func generateBuiltinCallAST(call *ast.CallExpr, ctx *BaseContext) (string, error) {
	switch call.Name {
	case "exit":
		return generateExitBuiltinAST(call, ctx)
	case "print":
		return generatePrintBuiltinAST(call, ctx)
	default:
		return "", fmt.Errorf("unknown built-in function: %s", call.Name)
	}
}

func generateExitBuiltinAST(call *ast.CallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if len(call.Arguments) > 0 {
		code, err := GenerateExprWithContext(call.Arguments[0], ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	EmitExitSyscall(&builder)
	return builder.String(), nil
}

func generatePrintBuiltinAST(call *ast.CallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if len(call.Arguments) == 0 {
		return "", nil
	}

	arg := call.Arguments[0]

	// Check if the argument is a string literal
	if lit, ok := arg.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralTypeString {
		label, exists := ctx.GetStringLabel(lit)
		if !exists {
			return "", fmt.Errorf("string literal not found in string map: %s", lit.Value)
		}

		EmitLoadAddress(&builder, "x1", label)
		EmitMoveImm(&builder, "x2", fmt.Sprintf("%d", len(lit.Value)))
		builder.WriteString("    mov x0, #1\n")
		builder.WriteString("    mov x16, #4\n")
		builder.WriteString("    svc #0x80\n")
		EmitNewline(&builder)

		return builder.String(), nil
	}

	// Handle integer printing
	code, err := GenerateExprWithContext(arg, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	EmitPrintInt(&builder)
	return builder.String(), nil
}

// GenerateVarDecl generates code for a variable declaration.
func GenerateVarDecl(stmt *ast.VarDeclStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	code, err := GenerateExprWithContext(stmt.Initializer, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	offset := ctx.DeclareVariable(stmt.Name, nil)
	EmitStoreToStack(&builder, "x2", offset)

	return builder.String(), nil
}

// GenerateAssignStmt generates code for a variable assignment.
func GenerateAssignStmt(stmt *ast.AssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	code, err := GenerateExprWithContext(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	offset, ok := ctx.GetVariableOffset(stmt.Name)
	if !ok {
		return "", fmt.Errorf("undefined variable: %s", stmt.Name)
	}

	EmitStoreToStack(&builder, "x2", offset)
	return builder.String(), nil
}

// GenerateExprWithContext generates code for an expression with variable support
func GenerateExprWithContext(expr ast.Expression, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		if e.Kind == ast.LiteralTypeString {
			label, _ := ctx.GetStringLabel(e)
			EmitLoadAddress(&builder, "x2", label)
		} else {
			EmitMoveImm(&builder, "x2", e.Value)
		}
		return builder.String(), nil

	case *ast.IdentifierExpr:
		offset, ok := ctx.GetVariableOffset(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		EmitLoadFromStack(&builder, "x2", offset)
		return builder.String(), nil

	case *ast.CallExpr:
		return GenerateCallExpr(e, ctx)

	case *ast.BinaryExpr:
		_, leftIsBinary := e.Left.(*ast.BinaryExpr)
		_, rightIsBinary := e.Right.(*ast.BinaryExpr)

		eval := &BinaryExprEvaluator{
			LeftIsComplex:  leftIsBinary,
			RightIsComplex: rightIsBinary,
			GenerateLeft: func() (string, error) {
				return GenerateExprWithContext(e.Left, ctx)
			},
			GenerateRight: func() (string, error) {
				return GenerateExprWithContext(e.Right, ctx)
			},
			GenerateLeftToReg: func(reg string) (string, error) {
				return generateOperandToReg(e.Left, reg, ctx)
			},
			GenerateRightToReg: func(reg string) (string, error) {
				return generateOperandToReg(e.Right, reg, ctx)
			},
		}

		setupCode, err := EmitBinaryExprSetup(eval)
		if err != nil {
			return "", err
		}
		builder.WriteString(setupCode)

		opCode, err := IntOperation(e.Op, true)
		if err != nil {
			return "", err
		}
		builder.WriteString(opCode)

		return builder.String(), nil

	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func generateOperandToReg(expr ast.Expression, reg string, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		if e.Kind == ast.LiteralTypeString {
			label, _ := ctx.GetStringLabel(e)
			EmitLoadAddress(&builder, reg, label)
		} else {
			EmitMoveImm(&builder, reg, e.Value)
		}

	case *ast.IdentifierExpr:
		offset, ok := ctx.GetVariableOffset(e.Name)
		if !ok {
			return "", fmt.Errorf("undefined variable: %s", e.Name)
		}
		EmitLoadFromStack(&builder, reg, offset)

	case *ast.BinaryExpr:
		code, err := GenerateExprWithContext(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	case *ast.CallExpr:
		code, err := GenerateCallExpr(e, ctx)
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

// intToStringFunctionText generates the int-to-string conversion routine.
func intToStringFunctionText() string {
	return defaultEmitter.IntToStringFunction()
}

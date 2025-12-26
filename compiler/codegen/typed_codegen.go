package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// TypedCodeGenerator generates ARM64 assembly from a type-checked TypedProgram.
// It handles type-aware code generation including:
//   - Proper register selection (x registers for integers, d registers for floats)
//   - Signed vs unsigned operations (sdiv vs udiv, etc.)
//   - Float literals in the data section
//   - String literal handling
//   - Runtime boundary checks with panic on overflow/division-by-zero
//   - Symbol table generation for stack traces
type TypedCodeGenerator struct {
	program     *semantic.TypedProgram
	sourceLines []string
	info        *ProgramInfo
	filename    string
	symtab      *SymbolTable
	checkGen    *CheckGenerator
}

// NewTypedCodeGenerator creates a new typed code generator.
func NewTypedCodeGenerator(program *semantic.TypedProgram, sourceLines []string) *TypedCodeGenerator {
	return NewTypedCodeGeneratorWithFilename(program, sourceLines, "")
}

// NewTypedCodeGeneratorWithFilename creates a new typed code generator with source filename.
func NewTypedCodeGeneratorWithFilename(program *semantic.TypedProgram, sourceLines []string, filename string) *TypedCodeGenerator {
	return &TypedCodeGenerator{
		program:     program,
		sourceLines: sourceLines,
		filename:    filename,
		symtab:      NewSymbolTable(filename),
		checkGen:    NewCheckGenerator(filename),
	}
}

// Generate produces ARM64 assembly code from the typed program.
func (g *TypedCodeGenerator) Generate() (string, error) {
	if len(g.program.Declarations) == 0 {
		return "", fmt.Errorf("no declarations found: programs must have at least one function")
	}

	builder := strings.Builder{}
	return g.generateFunctionBasedProgram(&builder)
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

	// Register functions in symbol table for stack traces
	for _, fn := range functions {
		g.symtab.AddFunction(fn.Name, fn.FnKeyword.Line)
	}

	// Write .data section if needed
	if len(g.info.FloatLiterals) > 0 || len(g.info.StringLiterals) > 0 || g.info.HasPrint || g.info.HasBoolPrint {
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

		// Boolean string literals for print(bool)
		if g.info.HasBoolPrint {
			builder.WriteString("_bool_true: .asciz \"true\"\n")
			builder.WriteString("_bool_true_len = 4\n")
			builder.WriteString("_bool_false: .asciz \"false\"\n")
			builder.WriteString("_bool_false_len = 5\n")
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

	// Generate symbol table for stack traces
	builder.WriteString(g.symtab.GenerateDataSection())

	// Include runtime panic handler
	builder.WriteString(RuntimePanicCode())

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

	// Emit function end label for symbol table
	builder.WriteString(GenerateFunctionEndLabel(fn.Name))

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
	case *semantic.TypedFieldAssignStmt:
		return g.generateFieldAssignStmt(s, ctx)
	case *semantic.TypedIndexAssignStmt:
		return g.generateIndexAssignStmt(s, ctx)
	case *semantic.TypedReturnStmt:
		return g.generateReturnStmt(s, ctx)
	case *semantic.TypedIfStmt:
		return g.generateIfStmt(s, ctx)
	case *semantic.TypedForStmt:
		return g.generateForStmt(s, ctx)
	case *semantic.TypedBreakStmt:
		return g.generateBreakStmt(s, ctx)
	case *semantic.TypedContinueStmt:
		return g.generateContinueStmt(s, ctx)
	case *semantic.TypedWhenExpr:
		return g.generateWhenStmt(s, ctx)
	default:
		return "", fmt.Errorf("unknown statement type: %T", s)
	}
}

func (g *TypedCodeGenerator) generateVarDecl(stmt *semantic.TypedVarDeclStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Check if this is a struct type
	if structType, ok := stmt.DeclaredType.(semantic.StructType); ok {
		return g.generateStructVarDecl(stmt, structType, ctx)
	}

	// Check if this is an array type
	if arrayType, ok := stmt.DeclaredType.(semantic.ArrayType); ok {
		return g.generateArrayVarDecl(stmt, arrayType, ctx)
	}

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

// generateStructVarDecl generates code for declaring a struct variable.
// The struct literal values are generated and stored at consecutive stack locations.
func (g *TypedCodeGenerator) generateStructVarDecl(stmt *semantic.TypedVarDeclStmt, structType semantic.StructType, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get the struct literal expression
	structLit, ok := stmt.Initializer.(*semantic.TypedStructLiteralExpr)
	if !ok {
		return "", fmt.Errorf("struct variable must be initialized with struct literal")
	}

	// Calculate total size needed (including nested structs)
	totalSlots := g.countStructSlots(structType)

	// Allocate space for all fields (we allocate first slot, then additional slots)
	baseOffset := ctx.DeclareVariable(stmt.Name, stmt.DeclaredType)

	// Allocate additional slots
	for i := 1; i < totalSlots; i++ {
		ctx.stackOffset += StackAlignment
	}

	// Generate and store all fields (handles nested structs recursively)
	code, err := g.generateStructFieldsInline(structLit, baseOffset, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	return builder.String(), nil
}

// countStructSlots counts the total number of stack slots needed for a struct type
// (recursively counting nested struct fields)
func (g *TypedCodeGenerator) countStructSlots(structType semantic.StructType) int {
	count := 0
	for _, field := range structType.Fields {
		if nestedStruct, ok := field.Type.(semantic.StructType); ok {
			count += g.countStructSlots(nestedStruct)
		} else {
			count++
		}
	}
	return count
}

// generateStructFieldsInline generates code to store all struct fields at the given base offset.
// Handles nested struct literals by recursively generating their fields.
func (g *TypedCodeGenerator) generateStructFieldsInline(structLit *semantic.TypedStructLiteralExpr, baseOffset int, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}
	currentOffset := baseOffset

	for i, arg := range structLit.Args {
		fieldType := structLit.Type.Fields[i].Type

		// Check if this argument is a nested struct literal
		if nestedLit, ok := arg.(*semantic.TypedStructLiteralExpr); ok {
			// Recursively generate nested struct fields
			code, err := g.generateStructFieldsInline(nestedLit, currentOffset, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)

			// Advance offset by the nested struct size
			nestedStruct := fieldType.(semantic.StructType)
			currentOffset += g.countStructSlots(nestedStruct) * StackAlignment
		} else {
			// Generate the field value
			code, err := g.generateExpr(arg, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)

			// Store the value
			if semantic.IsFloatType(fieldType) {
				builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", currentOffset))
			} else {
				EmitStoreToStack(&builder, "x2", currentOffset)
			}

			currentOffset += StackAlignment
		}
	}

	return builder.String(), nil
}

// getElementSlotCount returns the number of 16-byte stack slots needed for one array element.
func (g *TypedCodeGenerator) getElementSlotCount(elementType semantic.Type) int {
	if structType, ok := elementType.(semantic.StructType); ok {
		return g.countStructSlots(structType)
	}
	return 1 // primitives take 1 slot
}

// generateArrayVarDecl generates code for declaring an array variable.
// Each array element is stored in consecutive 16-byte aligned stack slots.
// For struct elements, each element takes multiple slots based on struct size.
func (g *TypedCodeGenerator) generateArrayVarDecl(stmt *semantic.TypedVarDeclStmt, arrayType semantic.ArrayType, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get the array literal expression
	arrayLit, ok := stmt.Initializer.(*semantic.TypedArrayLiteralExpr)
	if !ok {
		return "", fmt.Errorf("array variable must be initialized with array literal")
	}

	// Calculate element size in slots
	elementSlots := g.getElementSlotCount(arrayType.ElementType)
	totalSlots := arrayType.Size * elementSlots

	// Allocate space for all elements (first slot is allocated by DeclareVariable)
	baseOffset := ctx.DeclareVariable(stmt.Name, stmt.DeclaredType)

	// Allocate additional slots for remaining elements
	for i := 1; i < totalSlots; i++ {
		ctx.stackOffset += StackAlignment
	}

	// Generate and store each element
	currentOffset := baseOffset
	for _, elem := range arrayLit.Elements {
		// Check if element is a struct literal
		if structLit, ok := elem.(*semantic.TypedStructLiteralExpr); ok {
			code, err := g.generateStructFieldsInline(structLit, currentOffset, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
		} else {
			// Primitive element
			code, err := g.generateExpr(elem, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)

			if semantic.IsFloatType(arrayType.ElementType) {
				builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", currentOffset))
			} else {
				EmitStoreToStack(&builder, "x2", currentOffset)
			}
		}

		currentOffset += elementSlots * StackAlignment
	}

	return builder.String(), nil
}

// generateIndexAssignStmt generates code for array index assignment (e.g., arr[0] = 5)
func (g *TypedCodeGenerator) generateIndexAssignStmt(stmt *semantic.TypedIndexAssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get base offset from array variable
	baseOffset, err := g.getArrayBaseOffset(stmt.Array, ctx)
	if err != nil {
		return "", err
	}

	// Get element size from array type
	arrayType := stmt.Array.GetType().(semantic.ArrayType)
	elementSizeBytes := g.getElementSlotCount(arrayType.ElementType) * StackAlignment

	// Optimization: for literal indices, compute offset at compile time
	// and skip runtime bounds check (already validated in semantic analysis)
	if litIndex, ok := tryGetLiteralIndex(stmt.Index); ok {
		// Generate the value expression (result in x2 or d0)
		valueCode, err := g.generateExpr(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(valueCode)

		// Compile-time offset: baseOffset + index * elementSizeBytes
		offset := baseOffset + litIndex*elementSizeBytes

		// Store directly at the computed offset
		EmitStoreToStack(&builder, "x2", offset)
		return builder.String(), nil
	}

	// Dynamic index case
	// Generate the value expression (result in x2 or d0)
	valueCode, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Save value to stack temporarily
	builder.WriteString("    str x2, [sp, #-16]!\n")

	// Generate the index expression (result in x2)
	indexCode, err := g.generateExpr(stmt.Index, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(indexCode)

	// Runtime bounds check
	builder.WriteString(g.checkGen.ArrayBoundsCheck(stmt.ArraySize, stmt.LeftBracket.Line))

	// Restore value from stack
	builder.WriteString("    ldr x5, [sp], #16\n")

	// Calculate element address (result in x4)
	// Uses optimized shifted-add for power-of-2 element sizes
	emitArrayElementAddress(&builder, baseOffset, elementSizeBytes)

	// Store element at computed address
	builder.WriteString("    str x5, [x4]\n")

	return builder.String(), nil
}

// getArrayBaseOffset returns the stack offset for an array variable
func (g *TypedCodeGenerator) getArrayBaseOffset(expr semantic.TypedExpression, ctx *BaseContext) (int, error) {
	switch e := expr.(type) {
	case *semantic.TypedIdentifierExpr:
		slot, ok := ctx.GetVariable(e.Name)
		if !ok {
			return 0, fmt.Errorf("undefined variable: %s", e.Name)
		}
		return slot.Offset, nil
	default:
		return 0, fmt.Errorf("unsupported array expression type for indexing: %T", expr)
	}
}

// log2IfPowerOf2 returns the log2 of n if n is a power of 2, otherwise -1.
func log2IfPowerOf2(n int) int {
	if n <= 0 || (n&(n-1)) != 0 {
		return -1
	}
	log := 0
	for n > 1 {
		n >>= 1
		log++
	}
	return log
}

// emitArrayElementAddress generates code to compute the address of an array element.
// Expects the index to be in x2. After this code runs:
// - x4 contains the computed address (x29 - baseOffset - index*elementSize)
// The index in x2 is preserved.
// elementSizeBytes is the size of each element in bytes (16 for primitives, N*16 for structs).
func emitArrayElementAddress(builder *strings.Builder, baseOffset int, elementSizeBytes int) {
	shift := log2IfPowerOf2(elementSizeBytes)
	if shift >= 0 {
		// Power-of-2 size: use shifted register addressing
		// x4 = x29 - baseOffset (element 0 address)
		builder.WriteString(fmt.Sprintf("    sub x4, x29, #%d\n", baseOffset))
		// x3 = -index
		builder.WriteString("    neg x3, x2\n")
		// x4 = x4 + (x3 << shift) = element address
		builder.WriteString(fmt.Sprintf("    add x4, x4, x3, lsl #%d\n", shift))
	} else {
		// Non-power-of-2: use multiply (fallback, shouldn't happen with 16-byte aligned elements)
		builder.WriteString(fmt.Sprintf("    mov x3, #%d\n", elementSizeBytes))
		builder.WriteString("    mul x3, x2, x3\n")
		builder.WriteString(fmt.Sprintf("    mov x4, #%d\n", baseOffset))
		builder.WriteString("    add x3, x4, x3\n")
		builder.WriteString("    sub x4, x29, x3\n")
	}
}

// tryGetLiteralIndex checks if the expression is a literal integer and returns
// the value along with a success flag. Used to optimize array access with
// compile-time known indices.
func tryGetLiteralIndex(expr semantic.TypedExpression) (int, bool) {
	lit, ok := expr.(*semantic.TypedLiteralExpr)
	if !ok || lit.LitType != ast.LiteralTypeInteger {
		return 0, false
	}
	val, err := strconv.ParseInt(lit.Value, 10, 64)
	if err != nil {
		return 0, false
	}
	return int(val), true
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

// generateFieldAssignStmt generates code for a field assignment (e.g., p.y = 25)
func (g *TypedCodeGenerator) generateFieldAssignStmt(stmt *semantic.TypedFieldAssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate the value expression (result in x2 or d0)
	valueCode, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Get the field offset
	fieldOffset, err := g.getFieldOffset(stmt.Object, stmt.Field, ctx)
	if err != nil {
		return "", err
	}

	// Store the value at the computed offset
	// Determine field type from the object's struct type
	fieldType := g.getFieldType(stmt.Object, stmt.Field)
	if semantic.IsFloatType(fieldType) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", fieldOffset))
	} else {
		EmitStoreToStack(&builder, "x2", fieldOffset)
	}

	return builder.String(), nil
}

// getFieldOffset computes the stack offset for accessing a field on an object.
// It handles nested field access (e.g., rect.topLeft.x).
func (g *TypedCodeGenerator) getFieldOffset(object semantic.TypedExpression, fieldName string, ctx *BaseContext) (int, error) {
	switch obj := object.(type) {
	case *semantic.TypedIdentifierExpr:
		// Direct struct variable access: p.x
		slot, ok := ctx.GetVariable(obj.Name)
		if !ok {
			return 0, fmt.Errorf("undefined variable: %s", obj.Name)
		}
		structType, ok := slot.Type.(semantic.StructType)
		if !ok {
			return 0, fmt.Errorf("variable '%s' is not a struct type", obj.Name)
		}
		fieldByteOffset := g.getFieldByteOffset(structType, fieldName)
		if fieldByteOffset < 0 {
			return 0, fmt.Errorf("struct '%s' has no field '%s'", structType.Name, fieldName)
		}
		return slot.Offset + fieldByteOffset, nil

	case *semantic.TypedFieldAccessExpr:
		// Nested field access: rect.topLeft.x
		// First get the offset of the outer field
		outerOffset, err := g.getFieldOffset(obj.Object, obj.Field, ctx)
		if err != nil {
			return 0, err
		}
		// Then add the offset of the inner field
		structType, ok := obj.Type.(semantic.StructType)
		if !ok {
			return 0, fmt.Errorf("field '%s' is not a struct type", obj.Field)
		}
		fieldByteOffset := g.getFieldByteOffset(structType, fieldName)
		if fieldByteOffset < 0 {
			return 0, fmt.Errorf("struct '%s' has no field '%s'", structType.Name, fieldName)
		}
		return outerOffset + fieldByteOffset, nil

	default:
		return 0, fmt.Errorf("unsupported object type for field access: %T", object)
	}
}

// getFieldByteOffset returns the byte offset of a field within a struct,
// accounting for nested struct sizes. Returns -1 if field not found.
func (g *TypedCodeGenerator) getFieldByteOffset(structType semantic.StructType, fieldName string) int {
	offset := 0
	for _, field := range structType.Fields {
		if field.Name == fieldName {
			return offset
		}
		// Add the size of this field
		if nestedStruct, ok := field.Type.(semantic.StructType); ok {
			offset += g.countStructSlots(nestedStruct) * StackAlignment
		} else {
			offset += StackAlignment
		}
	}
	return -1 // field not found
}

// getFieldType returns the type of a field on an object
func (g *TypedCodeGenerator) getFieldType(object semantic.TypedExpression, fieldName string) semantic.Type {
	objectType := object.GetType()
	structType, ok := objectType.(semantic.StructType)
	if !ok {
		return semantic.TypeError
	}
	fieldInfo, found := structType.GetField(fieldName)
	if !found {
		return semantic.TypeError
	}
	return fieldInfo.Type
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

func (g *TypedCodeGenerator) generateIfStmt(stmt *semantic.TypedIfStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate condition (result in x2: 0 = false, non-zero = true)
	condCode, err := g.generateExpr(stmt.Condition, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(condCode)

	// Generate labels
	elseLabel := ctx.NextLabel("if_else")
	endLabel := ctx.NextLabel("if_end")

	// Branch to else if condition is false (x2 == 0)
	builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", elseLabel))

	// Generate then branch
	for _, stmt := range stmt.ThenBranch.Statements {
		stmtCode, err := g.generateStmt(stmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(stmtCode)
	}

	// Jump over else branch (only if there is one)
	if stmt.ElseBranch != nil {
		builder.WriteString(fmt.Sprintf("    b %s\n", endLabel))
	}

	// Else label
	builder.WriteString(fmt.Sprintf("%s:\n", elseLabel))

	// Generate else branch if present
	if stmt.ElseBranch != nil {
		switch elseBranch := stmt.ElseBranch.(type) {
		case *semantic.TypedIfStmt:
			// else if: recursively generate
			elseCode, err := g.generateIfStmt(elseBranch, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(elseCode)
		case *semantic.TypedBlockStmt:
			// else block
			for _, stmt := range elseBranch.Statements {
				stmtCode, err := g.generateStmt(stmt, ctx)
				if err != nil {
					return "", err
				}
				builder.WriteString(stmtCode)
			}
		default:
			return "", fmt.Errorf("unexpected else branch type: %T", elseBranch)
		}
	}

	// End label (only needed if there was an else branch)
	if stmt.ElseBranch != nil {
		builder.WriteString(fmt.Sprintf("%s:\n", endLabel))
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateForStmt(stmt *semantic.TypedForStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate labels
	loopStartLabel := ctx.NextLabel("for_start")
	loopContinueLabel := ctx.NextLabel("for_continue")
	loopEndLabel := ctx.NextLabel("for_end")

	// Push loop labels onto stack for break/continue
	ctx.PushLoop(loopContinueLabel, loopEndLabel)
	defer ctx.PopLoop()

	// Generate initialization (if present)
	if stmt.Init != nil {
		initCode, err := g.generateStmt(stmt.Init, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(initCode)
	}

	// Loop start label
	builder.WriteString(fmt.Sprintf("%s:\n", loopStartLabel))

	// Generate condition check (if present)
	if stmt.Condition != nil {
		condCode, err := g.generateExpr(stmt.Condition, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(condCode)

		// If condition is false (x2 == 0), jump to loop end
		builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", loopEndLabel))
	}
	// If no condition, it's an infinite loop (no conditional jump)

	// Generate body
	for _, bodyStmt := range stmt.Body.Statements {
		stmtCode, err := g.generateStmt(bodyStmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(stmtCode)
	}

	// Continue label (where continue jumps to, before update)
	builder.WriteString(fmt.Sprintf("%s:\n", loopContinueLabel))

	// Generate update (if present)
	if stmt.Update != nil {
		updateCode, err := g.generateStmt(stmt.Update, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(updateCode)
	}

	// Jump back to start
	builder.WriteString(fmt.Sprintf("    b %s\n", loopStartLabel))

	// Loop end label (where break jumps to)
	builder.WriteString(fmt.Sprintf("%s:\n", loopEndLabel))

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateBreakStmt(stmt *semantic.TypedBreakStmt, ctx *BaseContext) (string, error) {
	_, breakLabel, ok := ctx.CurrentLoop()
	if !ok {
		return "", fmt.Errorf("break outside loop at line %d", stmt.Keyword.Line)
	}
	return fmt.Sprintf("    b %s\n", breakLabel), nil
}

func (g *TypedCodeGenerator) generateContinueStmt(stmt *semantic.TypedContinueStmt, ctx *BaseContext) (string, error) {
	continueLabel, _, ok := ctx.CurrentLoop()
	if !ok {
		return "", fmt.Errorf("continue outside loop at line %d", stmt.Keyword.Line)
	}
	return fmt.Sprintf("    b %s\n", continueLabel), nil
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

	case *semantic.TypedStructLiteralExpr:
		return g.generateStructLiteral(e, ctx)

	case *semantic.TypedFieldAccessExpr:
		return g.generateFieldAccess(e, ctx)

	case *semantic.TypedArrayLiteralExpr:
		return g.generateArrayLiteral(e, ctx)

	case *semantic.TypedIndexExpr:
		return g.generateIndexExpr(e, ctx)

	case *semantic.TypedLenExpr:
		return g.generateLenExpr(e, ctx)

	case *semantic.TypedBinaryExpr:
		return g.generateBinaryExpr(e, ctx)

	case *semantic.TypedUnaryExpr:
		return g.generateUnaryExpr(e, ctx)

	case *semantic.TypedIfStmt:
		// If expression: generate like a statement, result will be in x2
		return g.generateIfStmt(e, ctx)

	case *semantic.TypedWhenExpr:
		// When expression: generate like a statement, result will be in x2
		return g.generateWhenStmt(e, ctx)

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

	if lit.LitType == ast.LiteralTypeBoolean {
		// Boolean literal: true = 1, false = 0
		if lit.Value == "true" {
			EmitMoveImm(&builder, "x2", "1")
		} else {
			EmitMoveImm(&builder, "x2", "0")
		}
		return builder.String(), nil
	}

	// Integer literal
	EmitMoveImm(&builder, "x2", lit.Value)

	// Sign extend for smaller types (i64/u64 don't need extension)
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
	default:
		// i64, u64: no sign extension needed for 64-bit values
	}

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateUnaryExpr(expr *semantic.TypedUnaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate operand (result in x2)
	operandCode, err := g.generateExpr(expr.Operand, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(operandCode)

	if expr.Op == "!" {
		// Logical NOT: flip 0 <-> 1
		// x2 = (x2 == 0) ? 1 : 0
		builder.WriteString("    cmp x2, #0\n")
		builder.WriteString("    cset x2, eq\n")
		return builder.String(), nil
	}

	return "", fmt.Errorf("unknown unary operator: %s", expr.Op)
}

// isComplexOperand returns true if the operand requires register preservation
// during binary expression evaluation (i.e., it may clobber x0/x1).
func isComplexOperand(expr semantic.TypedExpression) bool {
	switch expr.(type) {
	case *semantic.TypedBinaryExpr, *semantic.TypedIfStmt, *semantic.TypedCallExpr:
		return true
	default:
		return false
	}
}

func (g *TypedCodeGenerator) generateBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	if semantic.IsFloatType(expr.Type) {
		return g.generateFloatBinaryExpr(expr, ctx)
	}
	return g.generateIntBinaryExpr(expr, ctx)
}

func (g *TypedCodeGenerator) generateIntBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	// Handle short-circuit logical operators specially
	if expr.Op == "&&" {
		return g.generateLogicalAnd(expr, ctx)
	}
	if expr.Op == "||" {
		return g.generateLogicalOr(expr, ctx)
	}

	builder := strings.Builder{}

	// Check if operands are complex (need register preservation)
	leftIsComplex := isComplexOperand(expr.Left)
	rightIsComplex := isComplexOperand(expr.Right)

	eval := &BinaryExprEvaluator{
		LeftIsComplex:  leftIsComplex,
		RightIsComplex: rightIsComplex,
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

	// Use checked operations with runtime overflow/division-by-zero detection
	opCode, err := g.checkGen.IntOperationChecked(expr.Op, isSigned, expr.OpPos.Line)
	if err != nil {
		return "", err
	}
	builder.WriteString(opCode)

	return builder.String(), nil
}

// generateLogicalAnd generates code for && with short-circuit evaluation.
// If the left operand is false (0), we skip the right operand entirely.
func (g *TypedCodeGenerator) generateLogicalAnd(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}
	endLabel := ctx.NextLabel("and_end")

	// Evaluate left operand (result in x2)
	leftCode, err := g.generateExpr(expr.Left, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(leftCode)

	// If left is false (0), short-circuit to end with result 0
	builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", endLabel))

	// Evaluate right operand (result becomes the final result in x2)
	rightCode, err := g.generateExpr(expr.Right, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(rightCode)

	// End label - x2 already has the correct result
	builder.WriteString(fmt.Sprintf("%s:\n", endLabel))

	return builder.String(), nil
}

// generateLogicalOr generates code for || with short-circuit evaluation.
// If the left operand is true (non-zero), we skip the right operand and return 1.
func (g *TypedCodeGenerator) generateLogicalOr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}
	trueLabel := ctx.NextLabel("or_true")
	endLabel := ctx.NextLabel("or_end")

	// Evaluate left operand (result in x2)
	leftCode, err := g.generateExpr(expr.Left, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(leftCode)

	// If left is true (non-zero), short-circuit to true label
	builder.WriteString(fmt.Sprintf("    cbnz x2, %s\n", trueLabel))

	// Evaluate right operand (result becomes the final result in x2)
	rightCode, err := g.generateExpr(expr.Right, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(rightCode)
	builder.WriteString(fmt.Sprintf("    b %s\n", endLabel))

	// True label - set result to 1
	builder.WriteString(fmt.Sprintf("%s:\n", trueLabel))
	builder.WriteString("    mov x2, #1\n")

	// End label
	builder.WriteString(fmt.Sprintf("%s:\n", endLabel))

	return builder.String(), nil
}

func (g *TypedCodeGenerator) generateFloatBinaryExpr(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Check if operands are complex (need register preservation)
	leftIsComplex := isComplexOperand(expr.Left)
	rightIsComplex := isComplexOperand(expr.Right)

	eval := &FloatBinaryExprEvaluator{
		LeftIsComplex:  leftIsComplex,
		RightIsComplex: rightIsComplex,
		GenerateLeft: func() (string, error) {
			return g.generateExpr(expr.Left, ctx)
		},
		GenerateRight: func() (string, error) {
			return g.generateExpr(expr.Right, ctx)
		},
	}

	setupCode, err := EmitFloatBinaryExprSetupWithComplexity(eval)
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

	case *semantic.TypedIfStmt:
		code, err := g.generateIfStmt(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	case *semantic.TypedUnaryExpr:
		code, err := g.generateUnaryExpr(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	case *semantic.TypedFieldAccessExpr:
		code, err := g.generateFieldAccess(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	case *semantic.TypedIndexExpr:
		code, err := g.generateIndexExpr(e, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		if reg != "x2" {
			EmitMoveReg(&builder, reg, "x2")
		}

	case *semantic.TypedLenExpr:
		code, err := g.generateLenExpr(e, ctx)
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

// generateStructLiteral generates code for a struct literal expression.
// This is called when a struct is used as an expression (e.g., passed as argument).
// Note: When used in variable declaration, generateStructVarDecl handles it directly.
func (g *TypedCodeGenerator) generateStructLiteral(expr *semantic.TypedStructLiteralExpr, ctx *BaseContext) (string, error) {
	// For struct literals used as expressions (not in variable declarations),
	// we need to allocate temporary stack space and return a pointer or copy.
	// For now, we return an error as structs as expressions (not var decl) aren't fully supported.
	// The main use case (val p = Point(10, 20)) is handled by generateStructVarDecl.
	return "", fmt.Errorf("struct literals as expressions are not yet supported; use in variable declaration")
}

// generateFieldAccess generates code for accessing a struct field (e.g., p.x)
func (g *TypedCodeGenerator) generateFieldAccess(expr *semantic.TypedFieldAccessExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Check if the object is an index expression (e.g., arr[0].x)
	if indexExpr, ok := expr.Object.(*semantic.TypedIndexExpr); ok {
		// Get the field offset within the struct
		structType := indexExpr.Type.(semantic.StructType)
		fieldByteOffset := g.getFieldByteOffset(structType, expr.Field)
		if fieldByteOffset < 0 {
			return "", fmt.Errorf("struct '%s' has no field '%s'", structType.Name, expr.Field)
		}

		// Optimization: for literal indices, compute entire offset at compile time
		if litIndex, ok := tryGetLiteralIndex(indexExpr.Index); ok {
			baseOffset, err := g.getArrayBaseOffset(indexExpr.Array, ctx)
			if err != nil {
				return "", err
			}
			arrayType := indexExpr.Array.GetType().(semantic.ArrayType)
			elementSizeBytes := g.getElementSlotCount(arrayType.ElementType) * StackAlignment

			// Compile-time offset: baseOffset + index * elementSizeBytes + fieldByteOffset
			offset := baseOffset + litIndex*elementSizeBytes + fieldByteOffset

			// Load directly from the computed offset
			if semantic.IsFloatType(expr.Type) {
				builder.WriteString(fmt.Sprintf("    ldr d0, [x29, #-%d]\n", offset))
			} else {
				EmitLoadFromStack(&builder, "x2", offset)
			}
			return builder.String(), nil
		}

		// Dynamic index: generate the index expression (puts element address in x4)
		indexCode, err := g.generateIndexExpr(indexExpr, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(indexCode)

		// Subtract field offset from the element address in x4
		// (fields are stored at negative offsets from the base)
		if fieldByteOffset > 0 {
			builder.WriteString(fmt.Sprintf("    sub x4, x4, #%d\n", fieldByteOffset))
		}

		// Load the value from the computed address
		if semantic.IsFloatType(expr.Type) {
			builder.WriteString("    ldr d0, [x4]\n")
		} else {
			builder.WriteString("    ldr x2, [x4]\n")
		}

		return builder.String(), nil
	}

	// Static case: struct at known stack location
	fieldOffset, err := g.getFieldOffset(expr.Object, expr.Field, ctx)
	if err != nil {
		return "", err
	}

	// Load the value from the computed offset
	if semantic.IsFloatType(expr.Type) {
		builder.WriteString(fmt.Sprintf("    ldr d0, [x29, #-%d]\n", fieldOffset))
	} else {
		EmitLoadFromStack(&builder, "x2", fieldOffset)
	}

	return builder.String(), nil
}

// generateArrayLiteral generates code for an array literal expression.
// Note: When used in variable declaration, generateArrayVarDecl handles it directly.
func (g *TypedCodeGenerator) generateArrayLiteral(expr *semantic.TypedArrayLiteralExpr, ctx *BaseContext) (string, error) {
	// Array literals as standalone expressions are not supported
	// The main use case (val arr = [1, 2, 3]) is handled by generateArrayVarDecl
	return "", fmt.Errorf("array literals as expressions are not yet supported; use in variable declaration")
}

// generateIndexExpr generates code for array index access (e.g., arr[0])
func (g *TypedCodeGenerator) generateIndexExpr(expr *semantic.TypedIndexExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get base offset from array variable
	baseOffset, err := g.getArrayBaseOffset(expr.Array, ctx)
	if err != nil {
		return "", err
	}

	// Get element size from array type
	arrayType := expr.Array.GetType().(semantic.ArrayType)
	elementSizeBytes := g.getElementSlotCount(arrayType.ElementType) * StackAlignment

	// Optimization: for literal indices, compute offset at compile time
	// and skip runtime bounds check (already validated in semantic analysis)
	if litIndex, ok := tryGetLiteralIndex(expr.Index); ok {
		// Compile-time offset: baseOffset + index * elementSizeBytes
		offset := baseOffset + litIndex*elementSizeBytes

		// For struct elements, compute address into x4 for field access
		// For primitive elements, load directly
		if _, isStruct := expr.Type.(semantic.StructType); isStruct {
			// Struct access: put element address in x4 for subsequent field access
			builder.WriteString(fmt.Sprintf("    sub x4, x29, #%d\n", offset))
		} else if semantic.IsFloatType(expr.Type) {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x29, #-%d]\n", offset))
		} else {
			EmitLoadFromStack(&builder, "x2", offset)
		}
		return builder.String(), nil
	}

	// Dynamic index: generate index expression and runtime bounds check
	indexCode, err := g.generateExpr(expr.Index, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(indexCode)

	// Runtime bounds check
	builder.WriteString(g.checkGen.ArrayBoundsCheck(expr.ArraySize, expr.LeftBracket.Line))

	// Calculate element address (result in x4)
	// Uses optimized shifted-add for power-of-2 element sizes
	emitArrayElementAddress(&builder, baseOffset, elementSizeBytes)

	// For struct elements, we don't load - the address in x4 is used by field access
	// For primitive elements, load the value
	if _, isStruct := expr.Type.(semantic.StructType); isStruct {
		// Struct access: leave address in x4 for subsequent field access
		// This case is handled specially when accessed via TypedFieldAccessExpr
	} else if semantic.IsFloatType(expr.Type) {
		builder.WriteString("    ldr d0, [x4]\n")
	} else {
		builder.WriteString("    ldr x2, [x4]\n")
	}

	return builder.String(), nil
}

// generateLenExpr generates code for len() builtin on arrays
func (g *TypedCodeGenerator) generateLenExpr(expr *semantic.TypedLenExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}
	// len() is a compile-time constant for fixed-size arrays
	EmitMoveImm(&builder, "x2", fmt.Sprintf("%d", expr.ArraySize))
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

	// Use GenerateCallWithLine to emit bl with a label for line number tracking
	builder.WriteString(g.symtab.GenerateCallWithLine(call.Name, call.NamePos.Line))
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

	// Check if argument is a boolean
	if _, isBool := argType.(semantic.BooleanType); isBool {
		// Generate boolean expression (result in x2: 0 or 1)
		code, err := g.generateExpr(arg, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)

		// Branch based on boolean value
		falseLabel := ctx.NextLabel("print_false")
		endLabel := ctx.NextLabel("print_end")

		builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", falseLabel))

		// Print "true"
		EmitLoadAddress(&builder, "x1", "_bool_true")
		builder.WriteString("    mov x2, #4\n") // length of "true"
		builder.WriteString("    mov x0, #1\n")
		builder.WriteString("    mov x16, #4\n")
		builder.WriteString("    svc #0x80\n")
		builder.WriteString(fmt.Sprintf("    b %s\n", endLabel))

		// Print "false"
		builder.WriteString(fmt.Sprintf("%s:\n", falseLabel))
		EmitLoadAddress(&builder, "x1", "_bool_false")
		builder.WriteString("    mov x2, #5\n") // length of "false"
		builder.WriteString("    mov x0, #1\n")
		builder.WriteString("    mov x16, #4\n")
		builder.WriteString("    svc #0x80\n")

		builder.WriteString(fmt.Sprintf("%s:\n", endLabel))
		EmitNewline(&builder)
		return builder.String(), nil
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

// generateWhenStmt generates ARM64 code for a when expression/statement
// Form: when { cond -> body, ... }
func (g *TypedCodeGenerator) generateWhenStmt(when *semantic.TypedWhenExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}
	endLabel := ctx.NextLabel("when_end")

	for i, wcase := range when.Cases {
		if wcase.IsElse {
			// Else case: just generate the body
			bodyCode, err := g.generateWhenCaseBody(wcase.Body, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(bodyCode)
		} else {
			// Generate condition
			condCode, err := g.generateExpr(wcase.Condition, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(condCode)

			// Branch to next case if condition is false
			nextLabel := ctx.NextLabel(fmt.Sprintf("when_case_%d", i+1))
			builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", nextLabel))

			// Generate body
			bodyCode, err := g.generateWhenCaseBody(wcase.Body, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(bodyCode)

			// Jump to end
			builder.WriteString(fmt.Sprintf("    b %s\n", endLabel))

			// Next case label
			builder.WriteString(fmt.Sprintf("%s:\n", nextLabel))
		}
	}

	// End label
	builder.WriteString(fmt.Sprintf("%s:\n", endLabel))

	return builder.String(), nil
}

// generateWhenCaseBody generates code for a when case body
func (g *TypedCodeGenerator) generateWhenCaseBody(body semantic.TypedStatement, ctx *BaseContext) (string, error) {
	switch b := body.(type) {
	case *semantic.TypedBlockStmt:
		builder := strings.Builder{}
		for _, stmt := range b.Statements {
			stmtCode, err := g.generateStmt(stmt, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(stmtCode)
		}
		return builder.String(), nil
	case *semantic.TypedExprStmt:
		return g.generateExpr(b.Expr, ctx)
	default:
		return g.generateStmt(body, ctx)
	}
}

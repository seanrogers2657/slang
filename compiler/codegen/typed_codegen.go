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

// expressionHasNullableTag checks if a typed expression representing a nullable
// primitive would have its tag loaded into x3 after generateExpr is called.
// This is true for: identifiers, function calls, field access, and index access
// that return nullable types.
func expressionHasNullableTag(expr semantic.TypedExpression) bool {
	switch e := expr.(type) {
	case *semantic.TypedIdentifierExpr:
		return semantic.IsNullable(e.Type)
	case *semantic.TypedCallExpr:
		return semantic.IsNullable(e.Type)
	case *semantic.TypedFieldAccessExpr:
		return semantic.IsNullable(e.Type)
	case *semantic.TypedIndexExpr:
		return semantic.IsNullable(e.Type)
	default:
		return false
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
		g.symtab.AddFunction(fn.Name, fn.NamePos.Line)
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

	// Count stack slots needed for parameters (nullable primitives need 2 slots)
	paramSlots := 0
	for _, param := range fn.Parameters {
		if nullableType, isNullable := param.Type.(semantic.NullableType); isNullable {
			if !semantic.IsReferenceType(nullableType.InnerType) {
				paramSlots += 2 // tag + value
			} else {
				paramSlots += 1
			}
		} else {
			paramSlots += 1
		}
	}

	varCount := CountTypedVariables(fn.Body.Statements)
	totalLocals := paramSlots + varCount
	stackSize := totalLocals * StackAlignment

	EmitFunctionPrologue(&builder, stackSize)

	// Store parameters from registers
	// Nullable params use 2 consecutive registers (tag, value)
	regIdx := 0
	for _, param := range fn.Parameters {
		offset := ctx.DeclareVariable(param.Name, param.Type)

		if nullableType, isNullable := param.Type.(semantic.NullableType); isNullable {
			if !semantic.IsReferenceType(nullableType.InnerType) {
				// Nullable primitive: tag in xN, value in xN+1
				// Store tag at offset-8, value at offset
				tagOffset := offset - 8
				builder.WriteString(fmt.Sprintf("    str x%d, [x29, #-%d]\n", regIdx, tagOffset))
				regIdx++
				EmitStoreToStack(&builder, fmt.Sprintf("x%d", regIdx), offset)
				regIdx++
			} else {
				// Nullable reference: just one register
				EmitStoreToStack(&builder, fmt.Sprintf("x%d", regIdx), offset)
				regIdx++
			}
		} else {
			// Non-nullable: single register
			EmitStoreToStack(&builder, fmt.Sprintf("x%d", regIdx), offset)
			regIdx++
		}
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
	case *semantic.TypedWhileStmt:
		return g.generateWhileStmt(s, ctx)
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

	// Check if this is a nullable type
	if nullableType, isNullable := stmt.DeclaredType.(semantic.NullableType); isNullable {
		return g.generateNullableVarDecl(stmt, nullableType, ctx)
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

// generateNullableVarDecl generates code for declaring a nullable variable.
// For nullable primitives: stores tag (8 bytes) + value (8 bytes)
// For nullable references: stores pointer (8 bytes, 0 = null)
func (g *TypedCodeGenerator) generateNullableVarDecl(stmt *semantic.TypedVarDeclStmt, nullableType semantic.NullableType, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	isRef := semantic.IsReferenceType(nullableType.InnerType)

	// Check if initializer is null literal
	isNullInit := false
	if litExpr, ok := stmt.Initializer.(*semantic.TypedLiteralExpr); ok {
		if litExpr.LitType == ast.LiteralTypeNull {
			isNullInit = true
		}
	}

	// Allocate stack space (DeclareVariable handles sizing)
	offset := ctx.DeclareVariable(stmt.Name, stmt.DeclaredType)

	if isRef {
		// Nullable reference: just store pointer (0 for null)
		if isNullInit {
			// Store null (0)
			builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", offset))
		} else {
			// Generate the expression and store the pointer
			code, err := g.generateExpr(stmt.Initializer, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
			EmitStoreToStack(&builder, "x2", offset)
		}
	} else {
		// Nullable primitive: store tag + value
		// Layout: [tag (8 bytes)][value (8 bytes)]
		// offset points to the END of the allocation, so:
		// - value is at offset
		// - tag is at offset - 8
		tagOffset := offset - 8
		valueOffset := offset

		if isNullInit {
			// Store null: tag = 0, value = undefined
			builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", tagOffset))
		} else {
			// Check if initializer is a nullable expression that already has tag in x3
			hasTagInX3 := expressionHasNullableTag(stmt.Initializer)

			// Generate the expression
			code, err := g.generateExpr(stmt.Initializer, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)

			if hasTagInX3 {
				// Tag is already in x3 from the expression
				builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
			} else {
				// Non-null value: store tag = 1
				EmitMoveImm(&builder, "x3", "1")
				builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
			}

			// Store value
			EmitStoreToStack(&builder, "x2", valueOffset)
		}
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
		} else if nullableType, ok := field.Type.(semantic.NullableType); ok {
			// Nullable primitives need 2 slots: tag + value
			if !semantic.IsReferenceType(nullableType.InnerType) {
				count += 2
			} else {
				count++
			}
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
		} else if nullableType, ok := fieldType.(semantic.NullableType); ok && !semantic.IsReferenceType(nullableType.InnerType) {
			// Nullable primitive field: store tag at currentOffset, value at currentOffset+8
			tagOffset := currentOffset
			valueOffset := currentOffset + 8

			// Check if it's a null literal
			if litExpr, ok := arg.(*semantic.TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeNull {
				// Store tag = 0 (null)
				builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", tagOffset))
				// Store value = 0
				builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", valueOffset))
			} else {
				// Generate the expression value
				code, err := g.generateExpr(arg, ctx)
				if err != nil {
					return "", err
				}
				builder.WriteString(code)

				// Store tag = 1 (non-null)
				EmitMoveImm(&builder, "x3", "1")
				builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
				// Store value
				EmitStoreToStack(&builder, "x2", valueOffset)
			}

			currentOffset += 2 * StackAlignment
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
	// Nullable primitives need 2 slots: tag + value
	if nullableType, ok := elementType.(semantic.NullableType); ok {
		if !semantic.IsReferenceType(nullableType.InnerType) {
			return 2
		}
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
		} else if nullableType, ok := arrayType.ElementType.(semantic.NullableType); ok && !semantic.IsReferenceType(nullableType.InnerType) {
			// Nullable primitive element: store tag at currentOffset, value at currentOffset+8
			tagOffset := currentOffset
			valueOffset := currentOffset + 8

			// Check if element is a null literal
			if litExpr, ok := elem.(*semantic.TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeNull {
				// Store tag = 0 (null)
				builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", tagOffset))
				// Store value = 0
				builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", valueOffset))
			} else {
				// Generate the expression
				code, err := g.generateExpr(elem, ctx)
				if err != nil {
					return "", err
				}
				builder.WriteString(code)

				// Check if element is a nullable expression that already has tag in x3
				hasTagInX3 := expressionHasNullableTag(elem)

				if hasTagInX3 {
					// Tag is already in x3
					builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
				} else {
					// Non-null value: store tag = 1
					EmitMoveImm(&builder, "x3", "1")
					builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
				}
				// Store value
				EmitStoreToStack(&builder, "x2", valueOffset)
			}
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

	// Get element type and size from array type
	arrayType := stmt.Array.GetType().(semantic.ArrayType)
	elementType := arrayType.ElementType
	elementSizeBytes := g.getElementSlotCount(elementType) * StackAlignment

	// Handle nullable elements specially
	if semantic.IsNullable(elementType) {
		return g.generateNullableIndexAssign(stmt, baseOffset, elementSizeBytes, &builder, ctx)
	}

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

// generateNullableIndexAssign handles assignment to nullable array elements.
func (g *TypedCodeGenerator) generateNullableIndexAssign(stmt *semantic.TypedIndexAssignStmt, baseOffset, elementSizeBytes int, builder *strings.Builder, ctx *BaseContext) (string, error) {
	// Check if assigning null
	isNullAssign := false
	if litExpr, ok := stmt.Value.(*semantic.TypedLiteralExpr); ok {
		if litExpr.LitType == ast.LiteralTypeNull {
			isNullAssign = true
		}
	}

	// Check if value is a nullable expression that already has tag in x3
	hasTagInX3 := expressionHasNullableTag(stmt.Value)

	// Optimization: for literal indices, compute offset at compile time
	if litIndex, ok := tryGetLiteralIndex(stmt.Index); ok {
		// Compile-time offset: baseOffset + index * elementSizeBytes
		offset := baseOffset + litIndex*elementSizeBytes
		tagOffset := offset
		valueOffset := offset + 8

		if isNullAssign {
			// Store null: tag = 0
			builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", tagOffset))
			return builder.String(), nil
		}

		// Generate the value expression
		valueCode, err := g.generateExpr(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(valueCode)

		if hasTagInX3 {
			// Tag is already in x3 from the expression
			builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
		} else {
			// Non-null value: set tag = 1
			EmitMoveImm(builder, "x3", "1")
			builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
		}

		// Store value
		EmitStoreToStack(builder, "x2", valueOffset)
		return builder.String(), nil
	}

	// Dynamic index case - more complex
	// For now, generate a simpler but correct implementation

	if isNullAssign {
		// Generate the index expression
		indexCode, err := g.generateExpr(stmt.Index, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(indexCode)

		// Runtime bounds check
		builder.WriteString(g.checkGen.ArrayBoundsCheck(stmt.ArraySize, stmt.LeftBracket.Line))

		// Calculate element address (result in x4)
		emitArrayElementAddress(builder, baseOffset, elementSizeBytes)

		// Store null: tag = 0 at [x4], value undefined
		builder.WriteString("    str xzr, [x4]\n")
		return builder.String(), nil
	}

	// Generate the value expression first
	valueCode, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Save value and tag to stack temporarily
	builder.WriteString("    str x2, [sp, #-16]!\n") // save value
	if hasTagInX3 {
		builder.WriteString("    str x3, [sp, #-16]!\n") // save tag from expression
	}

	// Generate the index expression
	indexCode, err := g.generateExpr(stmt.Index, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(indexCode)

	// Runtime bounds check
	builder.WriteString(g.checkGen.ArrayBoundsCheck(stmt.ArraySize, stmt.LeftBracket.Line))

	// Restore tag and value from stack
	if hasTagInX3 {
		builder.WriteString("    ldr x3, [sp], #16\n") // restore tag
	}
	builder.WriteString("    ldr x5, [sp], #16\n") // restore value

	// Calculate element address (result in x4)
	emitArrayElementAddress(builder, baseOffset, elementSizeBytes)

	// Store tag at [x4]
	if hasTagInX3 {
		builder.WriteString("    str x3, [x4]\n")
	} else {
		EmitMoveImm(builder, "x3", "1")
		builder.WriteString("    str x3, [x4]\n")
	}

	// Store value at [x4 + 8]
	builder.WriteString("    str x5, [x4, #8]\n")

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

	slot, ok := ctx.GetVariable(stmt.Name)
	if !ok {
		return "", fmt.Errorf("undefined variable: %s", stmt.Name)
	}

	// Check if this is a nullable primitive type
	if nullableType, isNullable := slot.Type.(semantic.NullableType); isNullable {
		if !semantic.IsReferenceType(nullableType.InnerType) {
			// Nullable primitive: need to handle tag + value
			return g.generateNullableAssign(stmt, slot, &builder, ctx)
		}
	}

	// Non-nullable or nullable reference: generate value and store
	code, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	if semantic.IsFloatType(slot.Type) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", slot.Offset))
	} else {
		EmitStoreToStack(&builder, "x2", slot.Offset)
	}

	return builder.String(), nil
}

// generateNullableAssign handles assignment to nullable primitive variables.
// Updates both the tag and value slots appropriately.
func (g *TypedCodeGenerator) generateNullableAssign(stmt *semantic.TypedAssignStmt, slot VariableInfo, builder *strings.Builder, ctx *BaseContext) (string, error) {
	tagOffset := slot.Offset - 8
	valueOffset := slot.Offset

	// Check if assigning null
	if litExpr, ok := stmt.Value.(*semantic.TypedLiteralExpr); ok {
		if litExpr.LitType == ast.LiteralTypeNull {
			// Assigning null: set tag = 0
			builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", tagOffset))
			return builder.String(), nil
		}
	}

	// Check if value is a nullable expression that already has tag in x3
	hasTagInX3 := expressionHasNullableTag(stmt.Value)

	// Generate expression
	code, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	if hasTagInX3 {
		// Tag is already in x3 from the expression
		builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
	} else {
		// Non-null value: set tag = 1
		EmitMoveImm(builder, "x3", "1")
		builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
	}

	// Store value
	EmitStoreToStack(builder, "x2", valueOffset)

	return builder.String(), nil
}

// generateFieldAssignStmt generates code for a field assignment (e.g., p.y = 25)
func (g *TypedCodeGenerator) generateFieldAssignStmt(stmt *semantic.TypedFieldAssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Determine field type from the object's struct type
	fieldType := g.getFieldType(stmt.Object, stmt.Field)

	// Get the field offset
	fieldOffset, err := g.getFieldOffset(stmt.Object, stmt.Field, ctx)
	if err != nil {
		return "", err
	}

	// Handle nullable fields specially
	if semantic.IsNullable(fieldType) {
		return g.generateNullableFieldAssign(stmt, fieldOffset, &builder, ctx)
	}

	// Generate the value expression (result in x2 or d0)
	valueCode, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Store the value at the computed offset
	if semantic.IsFloatType(fieldType) {
		builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", fieldOffset))
	} else {
		EmitStoreToStack(&builder, "x2", fieldOffset)
	}

	return builder.String(), nil
}

// generateNullableFieldAssign handles assignment to nullable struct fields.
func (g *TypedCodeGenerator) generateNullableFieldAssign(stmt *semantic.TypedFieldAssignStmt, fieldOffset int, builder *strings.Builder, ctx *BaseContext) (string, error) {
	// Nullable field layout: [tag (8 bytes)][value (8 bytes)]
	// fieldOffset points to the start of the field, so:
	// - tag is at fieldOffset
	// - value is at fieldOffset + 8
	tagOffset := fieldOffset
	valueOffset := fieldOffset + 8

	// Check if assigning null
	if litExpr, ok := stmt.Value.(*semantic.TypedLiteralExpr); ok {
		if litExpr.LitType == ast.LiteralTypeNull {
			// Assigning null: set tag = 0
			builder.WriteString(fmt.Sprintf("    str xzr, [x29, #-%d]\n", tagOffset))
			return builder.String(), nil
		}
	}

	// Check if value is a nullable expression that already has tag in x3
	hasTagInX3 := expressionHasNullableTag(stmt.Value)

	// Generate expression
	code, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	if hasTagInX3 {
		// Tag is already in x3 from the expression
		builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
	} else {
		// Non-null value: set tag = 1
		EmitMoveImm(builder, "x3", "1")
		builder.WriteString(fmt.Sprintf("    str x3, [x29, #-%d]\n", tagOffset))
	}

	// Store value
	EmitStoreToStack(builder, "x2", valueOffset)

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
// accounting for nested struct sizes and nullable primitives. Returns -1 if field not found.
func (g *TypedCodeGenerator) getFieldByteOffset(structType semantic.StructType, fieldName string) int {
	offset := 0
	for _, field := range structType.Fields {
		if field.Name == fieldName {
			return offset
		}
		// Add the size of this field
		if nestedStruct, ok := field.Type.(semantic.StructType); ok {
			offset += g.countStructSlots(nestedStruct) * StackAlignment
		} else if nullableType, ok := field.Type.(semantic.NullableType); ok {
			// Nullable primitives need 2 slots: tag + value
			if !semantic.IsReferenceType(nullableType.InnerType) {
				offset += 2 * StackAlignment
			} else {
				offset += StackAlignment
			}
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

	// Check if we're returning a nullable type
	isNullableReturn := semantic.IsNullable(stmt.ExpectedType)

	if stmt.Value != nil {
		// Check if returning null literal for nullable return type
		if isNullableReturn {
			if litExpr, ok := stmt.Value.(*semantic.TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeNull {
				// Returning null: set tag=0, value=0
				EmitMoveImm(&builder, "x0", "0") // tag = 0 (null)
				EmitMoveImm(&builder, "x1", "0") // value = 0
				EmitReturnEpilogue(&builder)
				return builder.String(), nil
			}
		}

		// Check if the value expression is itself nullable (e.g., returning a nullable variable)
		valueIsNullable := semantic.IsNullable(stmt.Value.GetType())

		// Special case: returning a nullable variable - need to load both tag and value
		if isNullableReturn && valueIsNullable {
			if ident, ok := stmt.Value.(*semantic.TypedIdentifierExpr); ok {
				slot, found := ctx.GetVariable(ident.Name)
				if found {
					// Load tag and value from nullable variable
					tagOffset := slot.Offset - 8
					valueOffset := slot.Offset
					builder.WriteString(fmt.Sprintf("    ldr x0, [x29, #-%d]\n", tagOffset))  // tag
					builder.WriteString(fmt.Sprintf("    ldr x1, [x29, #-%d]\n", valueOffset)) // value
					EmitReturnEpilogue(&builder)
					return builder.String(), nil
				}
			}
		}

		code, err := g.generateExpr(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)

		if isNullableReturn {
			// Check if value came from a nullable function call (tag in x3, value in x2)
			if callExpr, ok := stmt.Value.(*semantic.TypedCallExpr); ok && semantic.IsNullable(callExpr.Type) {
				// Pass through the tag from the call
				EmitMoveReg(&builder, "x0", "x3") // tag from call
				EmitMoveReg(&builder, "x1", "x2") // value
			} else {
				// Non-null value: tag = 1
				EmitMoveImm(&builder, "x0", "1")  // tag = 1 (has value)
				EmitMoveReg(&builder, "x1", "x2") // value in x1
			}
		} else if semantic.IsFloatType(stmt.Value.GetType()) {
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

	// Generate labels with shared ID for the same loop
	loopID := ctx.NextLabelID()
	loopStartLabel := fmt.Sprintf("_for_%d", loopID)
	loopContinueLabel := fmt.Sprintf("_for_continue_%d", loopID)
	loopEndLabel := fmt.Sprintf("_for_end_%d", loopID)

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
		// Use optimized direct conditional branch for simple comparisons
		condCode, _, err := g.generateConditionBranchIfFalse(stmt.Condition, loopEndLabel, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(condCode)
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

func (g *TypedCodeGenerator) generateWhileStmt(stmt *semantic.TypedWhileStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate labels with shared ID for the same loop
	loopID := ctx.NextLabelID()
	loopStartLabel := fmt.Sprintf("_while_%d", loopID)
	loopContinueLabel := fmt.Sprintf("_while_continue_%d", loopID)
	loopEndLabel := fmt.Sprintf("_while_end_%d", loopID)

	// Push loop labels onto stack for break/continue
	ctx.PushLoop(loopContinueLabel, loopEndLabel)
	defer ctx.PopLoop()

	// Loop start label
	builder.WriteString(fmt.Sprintf("%s:\n", loopStartLabel))

	// Generate condition check (use optimized direct conditional branch)
	condCode, _, err := g.generateConditionBranchIfFalse(stmt.Condition, loopEndLabel, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(condCode)

	// Generate body
	for _, bodyStmt := range stmt.Body.Statements {
		stmtCode, err := g.generateStmt(bodyStmt, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(stmtCode)
	}

	// Continue label (where continue jumps to)
	builder.WriteString(fmt.Sprintf("%s:\n", loopContinueLabel))

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
		// Handle nullable primitives specially - load both tag and value
		if nullableType, isNullable := slot.Type.(semantic.NullableType); isNullable && !semantic.IsReferenceType(nullableType.InnerType) {
			// Nullable primitive: tag at offset-8, value at offset
			tagOffset := slot.Offset - 8
			builder.WriteString(fmt.Sprintf("    ldr x3, [x29, #-%d]\n", tagOffset))
			EmitLoadFromStack(&builder, "x2", slot.Offset)
		} else if semantic.IsFloatType(slot.Type) {
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

	case *semantic.TypedSafeCallExpr:
		return g.generateSafeCallExpr(e, ctx)

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

	if lit.LitType == ast.LiteralTypeNull {
		// Null literal: represented as 0 (null pointer for references, tag=0 for primitives)
		// For standalone null expression, just load 0 into x2
		EmitMoveImm(&builder, "x2", "0")
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

// isComparisonOp returns true if the operator is a comparison operator.
func isComparisonOp(op string) bool {
	switch op {
	case "<", ">", "<=", ">=", "==", "!=":
		return true
	default:
		return false
	}
}

// inverseCondition returns the ARM64 condition code for branching when the
// comparison is FALSE (i.e., the inverse of the comparison).
func inverseCondition(op string, signed bool) string {
	switch op {
	case "<":
		if signed {
			return "ge" // branch if >= (signed)
		}
		return "hs" // branch if >= (unsigned: higher or same)
	case ">":
		if signed {
			return "le" // branch if <= (signed)
		}
		return "ls" // branch if <= (unsigned: lower or same)
	case "<=":
		if signed {
			return "gt" // branch if > (signed)
		}
		return "hi" // branch if > (unsigned: higher)
	case ">=":
		if signed {
			return "lt" // branch if < (signed)
		}
		return "lo" // branch if < (unsigned: lower)
	case "==":
		return "ne" // branch if not equal
	case "!=":
		return "eq" // branch if equal
	default:
		return ""
	}
}

// generateConditionBranchIfFalse generates code that branches to falseLabel if
// the condition is false. For simple comparisons, this generates optimized code
// using a direct conditional branch instead of cset+cbz.
// Returns the generated code and a boolean indicating if it was optimized.
func (g *TypedCodeGenerator) generateConditionBranchIfFalse(cond semantic.TypedExpression, falseLabel string, ctx *BaseContext) (string, bool, error) {
	// Check if condition is a simple comparison
	binExpr, ok := cond.(*semantic.TypedBinaryExpr)
	if !ok || !isComparisonOp(binExpr.Op) {
		// Fall back to general expression generation
		condCode, err := g.generateExpr(cond, ctx)
		if err != nil {
			return "", false, err
		}
		return condCode + fmt.Sprintf("    cbz x2, %s\n", falseLabel), false, nil
	}

	// Skip optimization for complex operands (need register preservation)
	if isComplexOperand(binExpr.Left) || isComplexOperand(binExpr.Right) {
		condCode, err := g.generateExpr(cond, ctx)
		if err != nil {
			return "", false, err
		}
		return condCode + fmt.Sprintf("    cbz x2, %s\n", falseLabel), false, nil
	}

	builder := strings.Builder{}

	// Generate left operand into x0
	leftCode, err := g.generateOperandToReg(binExpr.Left, "x0", ctx)
	if err != nil {
		return "", false, err
	}
	builder.WriteString(leftCode)

	// Determine signedness for condition code
	isSigned := true
	if numType, ok := binExpr.Left.GetType().(semantic.NumericType); ok {
		isSigned = numType.IsSigned()
	}

	// Check if right operand is a small literal (can use immediate)
	if lit, ok := binExpr.Right.(*semantic.TypedLiteralExpr); ok && lit.LitType == ast.LiteralTypeInteger {
		val, err := strconv.ParseInt(lit.Value, 10, 64)
		if err == nil && val >= 0 && val < 4096 {
			// Use compare with immediate
			builder.WriteString(fmt.Sprintf("    cmp x0, #%d\n", val))
			builder.WriteString(fmt.Sprintf("    b.%s %s\n", inverseCondition(binExpr.Op, isSigned), falseLabel))
			return builder.String(), true, nil
		}
	}

	// Generate right operand into x1
	rightCode, err := g.generateOperandToReg(binExpr.Right, "x1", ctx)
	if err != nil {
		return "", false, err
	}
	builder.WriteString(rightCode)

	// Generate compare and conditional branch
	builder.WriteString("    cmp x0, x1\n")
	builder.WriteString(fmt.Sprintf("    b.%s %s\n", inverseCondition(binExpr.Op, isSigned), falseLabel))

	return builder.String(), true, nil
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

	// Handle null comparison specially (x == null, x != null)
	if expr.Op == "==" || expr.Op == "!=" {
		if g.isNullComparison(expr) {
			return g.generateNullComparison(expr, ctx)
		}
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

// isNullComparison checks if a binary expression is comparing against null
func (g *TypedCodeGenerator) isNullComparison(expr *semantic.TypedBinaryExpr) bool {
	// Check if either operand is a null literal
	if litExpr, ok := expr.Left.(*semantic.TypedLiteralExpr); ok {
		if litExpr.LitType == ast.LiteralTypeNull {
			return true
		}
	}
	if litExpr, ok := expr.Right.(*semantic.TypedLiteralExpr); ok {
		if litExpr.LitType == ast.LiteralTypeNull {
			return true
		}
	}
	return false
}

// generateNullComparison generates code for comparing a nullable value against null
// For nullable primitives (tagged union): checks the tag (0 = null, 1 = has value)
// For nullable references: checks if pointer is 0
func (g *TypedCodeGenerator) generateNullComparison(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Find the non-null operand
	var nullableExpr semantic.TypedExpression
	if _, ok := expr.Left.(*semantic.TypedLiteralExpr); ok {
		nullableExpr = expr.Right
	} else {
		nullableExpr = expr.Left
	}

	// Get the nullable's type
	nullableType := nullableExpr.GetType()
	innerType, isNullable := semantic.UnwrapNullable(nullableType)
	if !isNullable {
		// This shouldn't happen if semantic analysis is correct
		return "", fmt.Errorf("expected nullable type in null comparison")
	}

	isRef := semantic.IsReferenceType(innerType)

	// Handle identifier expression (the common case)
	if ident, ok := nullableExpr.(*semantic.TypedIdentifierExpr); ok {
		slot, found := ctx.GetVariable(ident.Name)
		if !found {
			return "", fmt.Errorf("undefined variable: %s", ident.Name)
		}

		if isRef {
			// Reference type: check if pointer is 0
			EmitLoadFromStack(&builder, "x2", slot.Offset)
			builder.WriteString("    cmp x2, #0\n")
		} else {
			// Primitive type: check tag (at offset - 8 from end of allocation)
			// The offset stored is the END of the allocation, tag is at offset - 8
			tagOffset := slot.Offset - 8
			builder.WriteString(fmt.Sprintf("    ldr x2, [x29, #-%d]\n", tagOffset))
			builder.WriteString("    cmp x2, #0\n")
		}

		// Set result based on comparison (== null means tag==0, != null means tag!=0)
		if expr.Op == "==" {
			builder.WriteString("    cset x2, eq\n")
		} else { // !=
			builder.WriteString("    cset x2, ne\n")
		}

		return builder.String(), nil
	}

	// For non-identifier expressions with primitive nullable types, we cannot
	// easily access the tag since generateExpr returns the value.
	// For now, only reference types are supported for complex expressions.
	if !isRef {
		return "", fmt.Errorf("null comparison on complex expressions is only supported for reference types, not primitive nullables; assign to a variable first")
	}

	// For reference types, generate the expression and compare pointer to 0
	code, err := g.generateExpr(nullableExpr, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// x2 contains the pointer value (0 = null)
	builder.WriteString("    cmp x2, #0\n")
	if expr.Op == "==" {
		builder.WriteString("    cset x2, eq\n")
	} else { // !=
		builder.WriteString("    cset x2, ne\n")
	}

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

	// Handle nullable primitive fields specially - load both tag and value
	if nullableType, ok := expr.Type.(semantic.NullableType); ok && !semantic.IsReferenceType(nullableType.InnerType) {
		// Nullable primitive field: tag at fieldOffset, value at fieldOffset+8
		tagOffset := fieldOffset
		valueOffset := fieldOffset + 8
		builder.WriteString(fmt.Sprintf("    ldr x3, [x29, #-%d]\n", tagOffset))
		EmitLoadFromStack(&builder, "x2", valueOffset)
		return builder.String(), nil
	}

	// Load the value from the computed offset
	if semantic.IsFloatType(expr.Type) {
		builder.WriteString(fmt.Sprintf("    ldr d0, [x29, #-%d]\n", fieldOffset))
	} else {
		EmitLoadFromStack(&builder, "x2", fieldOffset)
	}

	return builder.String(), nil
}

// generateSafeCallExpr generates code for safe call expression (e.g., person?.address)
// If object is null, returns null; otherwise returns the field value
func (g *TypedCodeGenerator) generateSafeCallExpr(expr *semantic.TypedSafeCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate unique labels for branching
	nullLabel := ctx.NextLabel("safe_null")
	doneLabel := ctx.NextLabel("safe_done")

	// Check if the inner type is a reference type (uses null pointer) or primitive (uses tagged union)
	isRef := semantic.IsReferenceType(expr.InnerType)

	// Generate the object expression
	objCode, err := g.generateExpr(expr.Object, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(objCode)

	if isRef {
		// Reference type: object is a pointer, null = 0
		// x2 contains the pointer (or 0 for null)
		builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", nullLabel))

		// Not null: load the field from the struct
		builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", expr.FieldOffset))
		builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

		// Null path: x2 is already 0
		builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
		// x2 already contains 0 (null)

		builder.WriteString(fmt.Sprintf("%s:\n", doneLabel))
	} else {
		// Primitive nullable: tagged union layout (from stack end):
		//   - tag at offset - 8 (8 bytes before value)
		//   - value at offset (the slot.Offset points here)
		// For simplicity, we only handle the common case where object is an identifier
		if ident, ok := expr.Object.(*semantic.TypedIdentifierExpr); ok {
			slot, found := ctx.GetVariable(ident.Name)
			if !found {
				return "", fmt.Errorf("undefined variable: %s", ident.Name)
			}

			// Load tag from stack (tag is at offset - 8)
			tagOffset := slot.Offset - 8
			builder.WriteString(fmt.Sprintf("    ldr x3, [x29, #-%d]\n", tagOffset))
			builder.WriteString(fmt.Sprintf("    cbz x3, %s\n", nullLabel))

			// Not null: load the struct pointer from value slot
			builder.WriteString(fmt.Sprintf("    ldr x2, [x29, #-%d]\n", slot.Offset))
			// Load the field from the struct
			builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", expr.FieldOffset))
			builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

			// Null path
			builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
			EmitMoveImm(&builder, "x2", "0")

			builder.WriteString(fmt.Sprintf("%s:\n", doneLabel))
		} else {
			return "", fmt.Errorf("safe call on complex expressions is only supported for reference types, not primitive nullables; assign to a variable first")
		}
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
		// For nullable primitives, load both tag and value
		// For other primitives, load directly
		if _, isStruct := expr.Type.(semantic.StructType); isStruct {
			// Struct access: put element address in x4 for subsequent field access
			builder.WriteString(fmt.Sprintf("    sub x4, x29, #%d\n", offset))
		} else if nullableType, ok := expr.Type.(semantic.NullableType); ok && !semantic.IsReferenceType(nullableType.InnerType) {
			// Nullable primitive element: tag at offset, value at offset+8
			tagOffset := offset
			valueOffset := offset + 8
			builder.WriteString(fmt.Sprintf("    ldr x3, [x29, #-%d]\n", tagOffset))
			EmitLoadFromStack(&builder, "x2", valueOffset)
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
	// For nullable primitives, load both tag and value
	// For other primitives, load the value
	if _, isStruct := expr.Type.(semantic.StructType); isStruct {
		// Struct access: leave address in x4 for subsequent field access
		// This case is handled specially when accessed via TypedFieldAccessExpr
	} else if nullableType, ok := expr.Type.(semantic.NullableType); ok && !semantic.IsReferenceType(nullableType.InnerType) {
		// Nullable primitive element: tag at [x4], value at [x4, #-8] (x4 points to tag)
		// Since x4 = base - (elementOffset), we load tag from [x4] and value from [x4, #-8]
		builder.WriteString("    ldr x3, [x4]\n")
		builder.WriteString("    ldr x2, [x4, #-8]\n")
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
		// Calculate total register slots needed (nullable args need 2 slots: tag + value)
		totalSlots := 0
		for _, arg := range call.Arguments {
			if semantic.IsNullable(arg.GetType()) {
				totalSlots += 2 // tag + value
			} else {
				totalSlots += 1
			}
		}

		// Allocate space for arguments on stack
		stackSpace := totalSlots * 16
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", stackSpace))

		// Evaluate each argument and store on stack
		slotIdx := 0
		for i, arg := range call.Arguments {
			argType := arg.GetType()
			isNullable := semantic.IsNullable(argType)

			if isNullable {
				// Nullable argument: need to load both tag and value
				if ident, ok := arg.(*semantic.TypedIdentifierExpr); ok {
					slot, found := ctx.GetVariable(ident.Name)
					if found {
						// Load tag and value from nullable variable
						tagOffset := slot.Offset - 8
						valueOffset := slot.Offset
						// Store tag at slotIdx
						builder.WriteString(fmt.Sprintf("    ldr x2, [x29, #-%d]\n", tagOffset))
						builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", slotIdx*16))
						// Store value at slotIdx+1
						builder.WriteString(fmt.Sprintf("    ldr x2, [x29, #-%d]\n", valueOffset))
						builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", (slotIdx+1)*16))
						slotIdx += 2
						continue
					}
				}
				// For other nullable expressions (e.g., function calls), generate normally
				// The expression should put tag in x3, value in x2
				code, err := g.generateExpr(arg, ctx)
				if err != nil {
					return "", err
				}
				builder.WriteString(code)
				// Check if it was a nullable call (tag in x3)
				if callExpr, ok := arg.(*semantic.TypedCallExpr); ok && semantic.IsNullable(callExpr.Type) {
					builder.WriteString(fmt.Sprintf("    str x3, [sp, #%d]\n", slotIdx*16))     // tag
					builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", (slotIdx+1)*16)) // value
				} else {
					// Null literal or other - tag = 0 or 1 based on context
					if litExpr, ok := arg.(*semantic.TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeNull {
						builder.WriteString(fmt.Sprintf("    str xzr, [sp, #%d]\n", slotIdx*16))    // tag = 0
						builder.WriteString(fmt.Sprintf("    str xzr, [sp, #%d]\n", (slotIdx+1)*16)) // value = 0
					} else {
						// Non-null value being passed as nullable
						EmitMoveImm(&builder, "x3", "1")
						builder.WriteString(fmt.Sprintf("    str x3, [sp, #%d]\n", slotIdx*16))    // tag = 1
						builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", (slotIdx+1)*16)) // value
					}
				}
				slotIdx += 2
			} else {
				// Non-nullable argument: single slot
				code, err := g.generateExpr(call.Arguments[i], ctx)
				if err != nil {
					return "", err
				}
				builder.WriteString(code)
				builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", slotIdx*16))
				slotIdx++
			}
		}

		// Load arguments from stack into registers x0-x7
		for i := 0; i < totalSlots && i < 8; i++ {
			builder.WriteString(fmt.Sprintf("    ldr x%d, [sp, #%d]\n", i, i*16))
		}

		// Restore stack pointer
		builder.WriteString(fmt.Sprintf("    add sp, sp, #%d\n", stackSpace))
	}

	// Use GenerateCallWithLine to emit bl with a label for line number tracking
	builder.WriteString(g.symtab.GenerateCallWithLine(call.Name, call.NamePos.Line))

	// Handle nullable return types: x0 = tag, x1 = value
	// For nullable returns, we keep x0 as tag and move x1 to x2 (standard result register)
	// The caller (VarDecl/Assign) will detect nullable type and store both appropriately
	if semantic.IsNullable(call.Type) {
		// x0 stays as tag, move value from x1 to x2
		EmitMoveReg(&builder, "x2", "x1")
		// Also preserve tag in x3 for the caller to use when storing
		EmitMoveReg(&builder, "x3", "x0")
	} else {
		EmitMoveReg(&builder, "x2", "x0")
	}

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

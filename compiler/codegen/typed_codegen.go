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
	program            *semantic.TypedProgram
	sourceLines        []string
	info               *ProgramInfo
	filename           string
	symtab             *SymbolTable
	checkGen           *CheckGenerator
	globalLabelCounter int // shared label counter across all methods
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
// This is true for: identifiers, function calls, field access, index access,
// safe field access, and safe method calls that return nullable types.
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
	case *semantic.TypedSafeCallExpr:
		// Safe call always returns nullable, and sets tag in x3
		return true
	case *semantic.TypedMethodCallExpr:
		// Safe navigation method call sets tag in x3
		if e.SafeNavigation {
			return true
		}
		// Regular method call returning nullable needs tag checking
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
	classes := make([]*semantic.TypedClassDecl, 0)
	objects := make([]*semantic.TypedObjectDecl, 0)

	for _, decl := range g.program.Declarations {
		switch d := decl.(type) {
		case *semantic.TypedFunctionDecl:
			functions = append(functions, d)
		case *semantic.TypedClassDecl:
			classes = append(classes, d)
		case *semantic.TypedObjectDecl:
			objects = append(objects, d)
		}
	}

	if len(functions) == 0 {
		return "", fmt.Errorf("no functions found")
	}

	// Collect literals and detect print usage from functions
	g.info = NewProgramInfo()
	for _, fn := range functions {
		g.info.CollectFromTypedFunction(fn)
	}

	// Collect literals from class methods
	for _, class := range classes {
		for _, method := range class.Methods {
			g.info.CollectFromTypedMethod(method)
		}
	}

	// Collect literals from object methods
	for _, obj := range objects {
		for _, method := range obj.Methods {
			g.info.CollectFromTypedMethod(method)
		}
	}

	// Register functions in symbol table for stack traces
	for _, fn := range functions {
		g.symtab.AddFunction(fn.Name, fn.NamePos.Line)
	}

	// Register class methods in symbol table
	for _, class := range classes {
		for _, method := range class.Methods {
			methodInfo := methodInfoFromTypedMethod(method)
			mangledName := mangleMethodNameWithInfo(class.Name, methodInfo)
			g.symtab.AddFunction(mangledName, method.NamePos.Line)
		}
	}

	// Register object methods in symbol table
	for _, obj := range objects {
		for _, method := range obj.Methods {
			methodInfo := methodInfoFromTypedMethod(method)
			mangledName := mangleMethodNameWithInfo(obj.Name, methodInfo)
			g.symtab.AddFunction(mangledName, method.NamePos.Line)
		}
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

	// Generate class methods
	for _, class := range classes {
		for _, method := range class.Methods {
			code, err := g.generateMethodDecl(class.Name, method)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
			builder.WriteString("\n")
		}
	}

	// Generate object methods
	for _, obj := range objects {
		for _, method := range obj.Methods {
			code, err := g.generateMethodDecl(obj.Name, method)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
			builder.WriteString("\n")
		}
	}

	// Generate symbol table for stack traces
	builder.WriteString(g.symtab.GenerateDataSection())

	// Include runtime: heap allocator and panic handler
	builder.WriteString(RuntimeHeapCode())
	builder.WriteString(RuntimePanicCode())

	return builder.String(), nil
}

// countParamSlots counts the number of stack slots needed for a set of parameters.
// Nullable primitives need 2 slots (tag + value), others need 1.
func countParamSlots(params []semantic.TypedParameter) int {
	slots := 0
	for _, param := range params {
		if nullableType, isNullable := param.Type.(semantic.NullableType); isNullable {
			if !semantic.IsReferenceType(nullableType.InnerType) {
				slots += 2 // tag + value
			} else {
				slots++
			}
		} else {
			slots++
		}
	}
	return slots
}

// storeParamsFromRegisters generates code to store parameters from registers to stack.
// Returns the final register index used.
func storeParamsFromRegisters(builder *strings.Builder, params []semantic.TypedParameter, ctx *BaseContext) int {
	regIdx := 0
	for _, param := range params {
		offset := ctx.DeclareVariable(param.Name, param.Type)

		if nullableType, isNullable := param.Type.(semantic.NullableType); isNullable {
			if !semantic.IsReferenceType(nullableType.InnerType) {
				// Nullable primitive: tag in xN, value in xN+1
				// Store tag at offset-8, value at offset
				tagOffset := offset - 8
				builder.WriteString(fmt.Sprintf("    str x%d, [x29, #-%d]\n", regIdx, tagOffset))
				regIdx++
				EmitStoreToStack(builder, fmt.Sprintf("x%d", regIdx), offset)
				regIdx++
			} else {
				// Nullable reference: just one register
				EmitStoreToStack(builder, fmt.Sprintf("x%d", regIdx), offset)
				regIdx++
			}
		} else {
			// Non-nullable: single register
			EmitStoreToStack(builder, fmt.Sprintf("x%d", regIdx), offset)
			regIdx++
		}
	}
	return regIdx
}

// generateBodyStatements generates code for a slice of statements.
func (g *TypedCodeGenerator) generateBodyStatements(builder *strings.Builder, stmts []semantic.TypedStatement, ctx *BaseContext) error {
	for _, stmt := range stmts {
		builder.WriteString(ctx.GetSourceLineComment(stmt.Pos()))
		code, err := g.generateStmt(stmt, ctx)
		if err != nil {
			return err
		}
		builder.WriteString(code)
	}
	return nil
}

func (g *TypedCodeGenerator) generateFunction(fn *semantic.TypedFunctionDecl) (string, error) {
	builder := strings.Builder{}

	EmitFunctionLabel(&builder, fn.Name)

	ctx := NewBaseContext(g.sourceLines)

	paramSlots := countParamSlots(fn.Parameters)
	varCount := CountTypedVariables(fn.Body.Statements)
	totalLocals := paramSlots + varCount
	stackSize := totalLocals * StackAlignment

	EmitFunctionPrologue(&builder, stackSize)

	// Store parameters from registers
	storeParamsFromRegisters(&builder, fn.Parameters, ctx)

	// Generate body
	if err := g.generateBodyStatements(&builder, fn.Body.Statements, ctx); err != nil {
		return "", err
	}

	// Cleanup owned pointers before function exit
	// (only for functions without explicit return, as return statements handle their own cleanup)
	builder.WriteString(g.generateOwnedVarCleanup(ctx, ""))

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

// mangleMethodName generates a unique assembly label for a class/object method.
// Format: _ClassName_methodName or _ClassName_methodName_ParamTypes for overloads.
// If methodInfo is provided and the method has parameters (other than self),
// parameter types are included to distinguish overloaded methods.
func mangleMethodName(className, methodName string) string {
	return fmt.Sprintf("_%s_%s", className, methodName)
}

// mangleMethodNameWithInfo generates a unique assembly label for a class/object method
// with full type information for overload disambiguation.
// Format: _ClassName_methodName_Type1_Type2_...
func mangleMethodNameWithInfo(className string, methodInfo *semantic.MethodInfo) string {
	if methodInfo == nil {
		return fmt.Sprintf("_%s_unknown", className)
	}

	// For non-overloaded methods or methods without additional parameters,
	// just use the simple name
	baseName := fmt.Sprintf("_%s_%s", className, methodInfo.Name)

	// Determine parameter offset (skip self for instance methods)
	paramOffset := 0
	if !methodInfo.IsStatic && len(methodInfo.ParamTypes) > 0 {
		paramOffset = 1
	}

	// If no parameters after self, use simple name
	if len(methodInfo.ParamTypes)-paramOffset == 0 {
		return baseName
	}

	// Build mangled name with parameter types
	var paramSuffix strings.Builder
	for i := paramOffset; i < len(methodInfo.ParamTypes); i++ {
		paramSuffix.WriteString("_")
		paramSuffix.WriteString(mangleTypeName(methodInfo.ParamTypes[i]))
	}

	return baseName + paramSuffix.String()
}

// mangleTypeName converts a type to a string suitable for name mangling.
func mangleTypeName(t semantic.Type) string {
	switch ty := t.(type) {
	case semantic.I8Type:
		return "i8"
	case semantic.I16Type:
		return "i16"
	case semantic.I32Type:
		return "i32"
	case semantic.I64Type:
		return "i64"
	case semantic.I128Type:
		return "i128"
	case semantic.U8Type:
		return "u8"
	case semantic.U16Type:
		return "u16"
	case semantic.U32Type:
		return "u32"
	case semantic.U64Type:
		return "u64"
	case semantic.U128Type:
		return "u128"
	case semantic.F32Type:
		return "f32"
	case semantic.F64Type:
		return "f64"
	case semantic.BooleanType:
		return "bool"
	case semantic.StringType:
		return "str"
	case semantic.VoidType:
		return "void"
	case semantic.NullableType:
		return mangleTypeName(ty.InnerType) + "opt"
	case semantic.OwnedPointerType:
		return "ptr" + mangleTypeName(ty.ElementType)
	case semantic.RefPointerType:
		return "ref" + mangleTypeName(ty.ElementType)
	case semantic.MutRefPointerType:
		return "mut" + mangleTypeName(ty.ElementType)
	case semantic.StructType:
		return ty.Name
	case semantic.ClassType:
		return ty.Name
	case semantic.ArrayType:
		return fmt.Sprintf("arr%s%d", mangleTypeName(ty.ElementType), ty.Size)
	default:
		return "unknown"
	}
}

// methodInfoFromTypedMethod constructs a MethodInfo from a TypedMethodDecl.
// Used for name mangling and overload disambiguation.
func methodInfoFromTypedMethod(method *semantic.TypedMethodDecl) *semantic.MethodInfo {
	paramTypes := make([]semantic.Type, len(method.Parameters))
	paramNames := make([]string, len(method.Parameters))
	for i, param := range method.Parameters {
		paramTypes[i] = param.Type
		paramNames[i] = param.Name
	}
	return &semantic.MethodInfo{
		Name:       method.Name,
		ParamTypes: paramTypes,
		ParamNames: paramNames,
		ReturnType: method.ReturnType,
		IsStatic:   method.IsStatic,
	}
}

// generateMethodDecl generates ARM64 assembly for a class/object method.
// Instance methods have 'self' as the first parameter (in x0).
// Static methods don't have 'self'.
// Methods returning ClassType use x8 for the return destination (caller passes address).
func (g *TypedCodeGenerator) generateMethodDecl(className string, method *semantic.TypedMethodDecl) (string, error) {
	builder := strings.Builder{}

	methodInfo := methodInfoFromTypedMethod(method)
	mangledName := mangleMethodNameWithInfo(className, methodInfo)
	EmitFunctionLabel(&builder, mangledName)

	ctx := NewBaseContextWithSharedCounter(g.sourceLines, &g.globalLabelCounter)

	// Check if this method returns a class by value
	_, returnsClass := method.ReturnType.(semantic.ClassType)

	paramSlots := countParamSlots(method.Parameters)

	// If returning class by value, need one extra slot to save x8
	extraSlots := 0
	if returnsClass {
		extraSlots = 1
	}

	varCount := CountTypedVariables(method.Body.Statements)
	totalLocals := paramSlots + varCount + extraSlots
	stackSize := totalLocals * StackAlignment

	EmitFunctionPrologue(&builder, stackSize)

	// Store parameters from registers
	storeParamsFromRegisters(&builder, method.Parameters, ctx)

	// If returning class by value, save x8 (caller's destination address) to stack
	if returnsClass {
		x8Offset := ctx.stackOffset + StackAlignment
		ctx.stackOffset = x8Offset
		builder.WriteString(fmt.Sprintf("    str x8, [x29, #-%d]\n", x8Offset))
		ctx.SetClassReturnType(method.ReturnType, x8Offset)
	}

	// Generate body
	if err := g.generateBodyStatements(&builder, method.Body.Statements, ctx); err != nil {
		return "", err
	}

	// Cleanup owned pointers before method exit
	builder.WriteString(g.generateOwnedVarCleanup(ctx, ""))

	EmitFunctionEpilogue(&builder, totalLocals > 0)

	// Emit method end label for symbol table
	builder.WriteString(GenerateFunctionEndLabel(mangledName))

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

	// Check if this is a class type
	if classType, ok := stmt.DeclaredType.(semantic.ClassType); ok {
		return g.generateClassVarDecl(stmt, classType, ctx)
	}

	// Check if this is an array type
	if arrayType, ok := stmt.DeclaredType.(semantic.ArrayType); ok {
		return g.generateArrayVarDecl(stmt, arrayType, ctx)
	}

	// Check if this is a nullable type
	if nullableType, isNullable := stmt.DeclaredType.(semantic.NullableType); isNullable {
		return g.generateNullableVarDecl(stmt, nullableType, ctx)
	}

	// Check if this is an owned pointer type
	if ownedType, isOwned := stmt.DeclaredType.(semantic.OwnedPointerType); isOwned {
		return g.generateOwnedPointerVarDecl(stmt, ownedType, ctx)
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

// generateOwnedPointerVarDecl generates code for declaring an owned pointer variable.
// Registers the variable for cleanup at scope exit.
func (g *TypedCodeGenerator) generateOwnedPointerVarDecl(stmt *semantic.TypedVarDeclStmt, ownedType semantic.OwnedPointerType, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate the initializer (Heap.new() call) - result pointer in x2
	code, err := g.generateExpr(stmt.Initializer, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Declare the variable and store the pointer
	offset := ctx.DeclareVariable(stmt.Name, stmt.DeclaredType)
	EmitStoreToStack(&builder, "x2", offset)

	// Calculate allocation size for cleanup
	allocSize := g.calculateHeapAllocSize(ownedType.ElementType)

	// Register for cleanup at scope exit
	ctx.RegisterOwnedVar(stmt.Name, offset, allocSize, ownedType.ElementType)

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

// generateClassVarDecl generates code for declaring a class variable on the stack.
// Handles both class literal initializers and method calls returning class by value.
func (g *TypedCodeGenerator) generateClassVarDecl(stmt *semantic.TypedVarDeclStmt, classType semantic.ClassType, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Calculate total size needed
	totalSlots := g.countClassSlots(classType)

	// Allocate space for all fields (we allocate first slot, then additional slots)
	baseOffset := ctx.DeclareVariable(stmt.Name, stmt.DeclaredType)

	// Allocate additional slots
	for i := 1; i < totalSlots; i++ {
		ctx.stackOffset += StackAlignment
	}

	// Check if initializer is a class literal - generate fields directly
	if classLit, ok := stmt.Initializer.(*semantic.TypedClassLiteralExpr); ok {
		code, err := g.generateClassFieldsInline(classLit, baseOffset, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
		return builder.String(), nil
	}

	// Check if initializer is a method call returning class by value
	if methodCall, ok := stmt.Initializer.(*semantic.TypedMethodCallExpr); ok {
		if _, isClass := methodCall.Type.(semantic.ClassType); isClass {
			// Set x8 to point to the variable's stack space
			builder.WriteString(fmt.Sprintf("    sub x8, x29, #%d\n", baseOffset))

			// Generate the method call (it will write to [x8])
			code, err := g.generateMethodCallWithDestination(methodCall, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)
			return builder.String(), nil
		}
	}

	return "", fmt.Errorf("class variable must be initialized with class literal or method returning class")
}

// countClassSlots counts the total number of 16-byte stack slots needed for a class type.
// Classes use compact 8-byte field layout (matching heap allocation) to ensure method
// compatibility between stack and heap allocated instances.
func (g *TypedCodeGenerator) countClassSlots(classType semantic.ClassType) int {
	// Calculate total bytes needed (8 bytes per field, matching heap layout)
	totalBytes := len(classType.Fields) * 8
	// Round up to 16-byte alignment
	return (totalBytes + StackAlignment - 1) / StackAlignment
}

// generateClassFieldsInline generates code to store all class fields at the given base offset.
// Uses 8-byte field offsets (matching heap layout) for compatibility with methods.
// Fields are stored so that field i is at address (x29 - baseOffset + i*8), allowing
// method code to access fields at positive offsets from the base pointer.
func (g *TypedCodeGenerator) generateClassFieldsInline(classLit *semantic.TypedClassLiteralExpr, baseOffset int, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Stack layout (growing downward):
	// Base pointer points to x29 - baseOffset
	// Field 0 at [x29 - baseOffset] = [base + 0]
	// Field 1 at [x29 - baseOffset + 8] = [base + 8]
	// So the stack offset for field i is: baseOffset - i*8
	for i, arg := range classLit.Args {
		fieldStackOffset := baseOffset - i*8

		// Generate the field value
		code, err := g.generateExpr(arg, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)

		// Store the value at the computed offset
		fieldType := classLit.Type.Fields[i].Type
		if semantic.IsFloatType(fieldType) {
			builder.WriteString(fmt.Sprintf("    str d0, [x29, #-%d]\n", fieldStackOffset))
		} else {
			builder.WriteString(fmt.Sprintf("    str x2, [x29, #-%d]\n", fieldStackOffset))
		}
	}

	return builder.String(), nil
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

	// Check if this is an owned pointer type - need to free old value first
	if ownedType, isOwned := slot.Type.(semantic.OwnedPointerType); isOwned {
		return g.generateOwnedPointerAssign(stmt, slot, ownedType, ctx)
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

// generateOwnedPointerAssign handles assignment to owned pointer variables.
// Frees the old value before storing the new one.
func (g *TypedCodeGenerator) generateOwnedPointerAssign(stmt *semantic.TypedAssignStmt, slot VariableInfo, ownedType semantic.OwnedPointerType, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Calculate allocation size for munmap
	allocSize := g.calculateHeapAllocSize(ownedType.ElementType)

	// Free the old value first
	builder.WriteString("    // free old owned pointer before reassignment\n")
	builder.WriteString(g.emitMunmap(slot.Offset, allocSize))

	// Generate the new value (result pointer in x2)
	code, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Store the new pointer
	EmitStoreToStack(&builder, "x2", slot.Offset)

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

	// Check if the object is an owned pointer (Own<T>) - requires heap dereference
	if semantic.IsOwnedPointer(stmt.Object.GetType()) {
		return g.generateFieldAssignThroughOwnedPointer(stmt, ctx)
	}

	// Check if the object is a reference pointer (Ref<T> or MutRef<T>) - requires heap dereference
	if semantic.IsAnyRefPointer(stmt.Object.GetType()) {
		return g.generateFieldAssignThroughRefPointer(stmt, ctx)
	}

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

// generateFieldAssignThroughOwnedPointer generates code for field assignment through Own<T>.
// Loads the pointer from stack, then stores the value to the heap.
func (g *TypedCodeGenerator) generateFieldAssignThroughOwnedPointer(stmt *semantic.TypedFieldAssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get the struct/class type from the owned pointer
	ownedType := stmt.Object.GetType().(semantic.OwnedPointerType)
	var fields []semantic.StructFieldInfo
	if structType, ok := ownedType.ElementType.(semantic.StructType); ok {
		fields = structType.Fields
	} else if classType, ok := ownedType.ElementType.(semantic.ClassType); ok {
		fields = classType.Fields
	} else {
		return "", fmt.Errorf("field assignment through owned pointer requires struct or class type, got %s", ownedType.ElementType.String())
	}

	// Calculate the field byte offset within the struct/class (on heap, each field is 8 bytes)
	fieldByteOffset := 0
	var fieldType semantic.Type
	for _, field := range fields {
		if field.Name == stmt.Field {
			fieldType = field.Type
			break
		}
		fieldByteOffset += 8 // Each field is 8-byte aligned on heap
	}

	// Step 1: Generate the value expression (result in x2 or d0)
	valueCode, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Step 2: Save the value temporarily (we need x2 for loading the pointer)
	if semantic.IsFloatType(fieldType) {
		builder.WriteString("    str d0, [sp, #-16]!\n")
	} else {
		builder.WriteString("    str x2, [sp, #-16]!\n")
	}

	// Step 3: Generate the object expression (loads the pointer into x2)
	objCode, err := g.generateExpr(stmt.Object, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(objCode)

	// x2 now contains the heap pointer, move it to x4
	builder.WriteString("    mov x4, x2\n")

	// Step 4: Restore the value
	if semantic.IsFloatType(fieldType) {
		builder.WriteString("    ldr d0, [sp], #16\n")
		// Store the value at [x4 + fieldByteOffset]
		if fieldByteOffset == 0 {
			builder.WriteString("    str d0, [x4]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    str d0, [x4, #%d]\n", fieldByteOffset))
		}
	} else {
		builder.WriteString("    ldr x2, [sp], #16\n")
		// Store the value at [x4 + fieldByteOffset]
		if fieldByteOffset == 0 {
			builder.WriteString("    str x2, [x4]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    str x2, [x4, #%d]\n", fieldByteOffset))
		}
	}

	return builder.String(), nil
}

// generateFieldAssignThroughRefPointer generates code for field assignment through Ref<T>/MutRef<T>.
// Loads the pointer from stack, then stores the value to the heap.
// Very similar to generateFieldAssignThroughOwnedPointer since both are heap pointers.
func (g *TypedCodeGenerator) generateFieldAssignThroughRefPointer(stmt *semantic.TypedFieldAssignStmt, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get the struct type from the ref pointer (either Ref<T> or MutRef<T>)
	objectType := stmt.Object.GetType()
	var elementType semantic.Type
	if refType, ok := objectType.(semantic.RefPointerType); ok {
		elementType = refType.ElementType
	} else if mutRefType, ok := objectType.(semantic.MutRefPointerType); ok {
		elementType = mutRefType.ElementType
	} else {
		return "", fmt.Errorf("expected Ref<T> or MutRef<T>, got %s", objectType.String())
	}

	// Get fields from either StructType or ClassType
	var fields []semantic.StructFieldInfo
	if structType, ok := elementType.(semantic.StructType); ok {
		fields = structType.Fields
	} else if classType, ok := elementType.(semantic.ClassType); ok {
		fields = classType.Fields
	} else {
		return "", fmt.Errorf("field assignment through ref pointer requires struct or class type, got %s", elementType.String())
	}

	// Calculate the field byte offset within the struct/class (on heap, each field is 8 bytes)
	fieldByteOffset := 0
	var fieldType semantic.Type
	for _, field := range fields {
		if field.Name == stmt.Field {
			fieldType = field.Type
			break
		}
		fieldByteOffset += 8 // Each field is 8-byte aligned on heap
	}

	// Step 1: Generate the value expression (result in x2 or d0)
	valueCode, err := g.generateExpr(stmt.Value, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Step 2: Save the value temporarily (we need x2 for loading the pointer)
	if semantic.IsFloatType(fieldType) {
		builder.WriteString("    str d0, [sp, #-16]!\n")
	} else {
		builder.WriteString("    str x2, [sp, #-16]!\n")
	}

	// Step 3: Generate the object expression (loads the pointer into x2)
	objCode, err := g.generateExpr(stmt.Object, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(objCode)

	// x2 now contains the heap pointer, move it to x4
	builder.WriteString("    mov x4, x2\n")

	// Step 4: Restore the value
	if semantic.IsFloatType(fieldType) {
		builder.WriteString("    ldr d0, [sp], #16\n")
		// Store the value at [x4 + fieldByteOffset]
		if fieldByteOffset == 0 {
			builder.WriteString("    str d0, [x4]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    str d0, [x4, #%d]\n", fieldByteOffset))
		}
	} else {
		builder.WriteString("    ldr x2, [sp], #16\n")
		// Store the value at [x4 + fieldByteOffset]
		if fieldByteOffset == 0 {
			builder.WriteString("    str x2, [x4]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    str x2, [x4, #%d]\n", fieldByteOffset))
		}
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

	// Check if we're returning a class by value
	if classType, x8Offset, ok := ctx.GetClassReturnType(); ok && stmt.Value != nil {
		return g.generateClassReturn(stmt, classType.(semantic.ClassType), x8Offset, ctx)
	}

	// Determine if we're returning an owned pointer variable (ownership transfer)
	// In that case, we skip cleanup for that variable
	skipVar := ""
	if stmt.Value != nil {
		if ident, ok := stmt.Value.(*semantic.TypedIdentifierExpr); ok {
			if semantic.IsOwnedPointer(ident.Type) {
				skipVar = ident.Name
			}
		}
	}

	// Check if we're returning a nullable type
	isNullableReturn := semantic.IsNullable(stmt.ExpectedType)

	if stmt.Value != nil {
		// Check if returning null literal for nullable return type
		if isNullableReturn {
			if litExpr, ok := stmt.Value.(*semantic.TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeNull {
				// Returning null: set tag=0, value=0
				EmitMoveImm(&builder, "x0", "0") // tag = 0 (null)
				EmitMoveImm(&builder, "x1", "0") // value = 0
				// Cleanup owned pointers before return
				builder.WriteString(g.generateOwnedVarCleanup(ctx, skipVar))
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
					// Cleanup owned pointers before return
					builder.WriteString(g.generateOwnedVarCleanup(ctx, skipVar))
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

	// Cleanup owned pointers before return
	builder.WriteString(g.generateOwnedVarCleanup(ctx, skipVar))

	EmitReturnEpilogue(&builder)
	return builder.String(), nil
}

// generateClassReturn handles returning a class by value.
// It writes the class fields to the destination address passed in x8.
func (g *TypedCodeGenerator) generateClassReturn(stmt *semantic.TypedReturnStmt, classType semantic.ClassType, x8Offset int, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Load destination address from saved x8
	builder.WriteString(fmt.Sprintf("    ldr x8, [x29, #-%d]\n", x8Offset))

	// Check if returning a class literal - can write fields directly
	if classLit, ok := stmt.Value.(*semantic.TypedClassLiteralExpr); ok {
		// Write each field to the destination
		for i, arg := range classLit.Args {
			fieldOffset := i * 8 // 8-byte field spacing

			// Generate the field value
			code, err := g.generateExpr(arg, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(code)

			// Store to destination + field offset
			builder.WriteString(fmt.Sprintf("    str x2, [x8, #%d]\n", fieldOffset))
		}
	} else {
		// Returning a variable or expression - need to copy
		// Generate the expression (should produce address of class in x2)
		code, err := g.generateExpr(stmt.Value, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)

		// x2 now contains address of source class
		// Copy all fields from source to destination
		for i := range classType.Fields {
			fieldOffset := i * 8
			// Load from source
			builder.WriteString(fmt.Sprintf("    ldr x3, [x2, #%d]\n", fieldOffset))
			// Store to destination
			builder.WriteString(fmt.Sprintf("    str x3, [x8, #%d]\n", fieldOffset))
		}
	}

	// Cleanup owned pointers before return
	builder.WriteString(g.generateOwnedVarCleanup(ctx, ""))

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

	case *semantic.TypedClassLiteralExpr:
		return g.generateClassLiteral(e, ctx)

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

	case *semantic.TypedMethodCallExpr:
		return g.generateMethodCallExpr(e, ctx)

	case *semantic.TypedSelfExpr:
		return g.generateSelfExpr(e, ctx)

	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// generateSelfExpr generates code for a 'self' expression in a method body.
// Self is stored on the stack like other parameters.
func (g *TypedCodeGenerator) generateSelfExpr(expr *semantic.TypedSelfExpr, ctx *BaseContext) (string, error) {
	slot, ok := ctx.GetVariable("self")
	if !ok {
		return "", fmt.Errorf("'self' not found in scope - are you inside a method?")
	}
	builder := strings.Builder{}
	EmitLoadFromStack(&builder, "x2", slot.Offset)
	return builder.String(), nil
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
	case *semantic.TypedBinaryExpr, *semantic.TypedIfStmt, *semantic.TypedCallExpr, *semantic.TypedMethodCallExpr:
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
	if expr.Op == "?:" {
		return g.generateElvis(expr, ctx)
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

// generateElvis generates code for ?: with short-circuit evaluation.
// If the left operand is non-null, use its unwrapped value; otherwise evaluate right operand.
func (g *TypedCodeGenerator) generateElvis(expr *semantic.TypedBinaryExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}
	endLabel := ctx.NextLabel("elvis_end")

	// Get the nullable type to determine if it's a reference or primitive
	leftType := expr.Left.GetType()
	innerType, isNullable := semantic.UnwrapNullable(leftType)
	if !isNullable {
		return "", fmt.Errorf("elvis operator requires nullable left operand, got %s", leftType.String())
	}

	isRef := semantic.IsReferenceType(innerType)

	// Handle identifier expression (the common case)
	if ident, ok := expr.Left.(*semantic.TypedIdentifierExpr); ok {
		slot, found := ctx.GetVariable(ident.Name)
		if !found {
			return "", fmt.Errorf("undefined variable: %s", ident.Name)
		}

		if isRef {
			// Reference type: load pointer, check if non-null
			EmitLoadFromStack(&builder, "x2", slot.Offset)
			builder.WriteString(fmt.Sprintf("    cbnz x2, %s\n", endLabel))
		} else {
			// Primitive type: check tag at offset - 8, load value into x2
			tagOffset := slot.Offset - 8
			builder.WriteString(fmt.Sprintf("    ldr x3, [x29, #-%d]\n", tagOffset))
			EmitLoadFromStack(&builder, "x2", slot.Offset)
			builder.WriteString(fmt.Sprintf("    cbnz x3, %s\n", endLabel))
		}

		// Null path: evaluate right operand (overwrites x2)
		rightCode, err := g.generateExpr(expr.Right, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(rightCode)

		builder.WriteString(fmt.Sprintf("%s:\n", endLabel))
		return builder.String(), nil
	}

	// For complex expressions (function calls, field access, etc.)
	// Generate the expression first
	leftCode, err := g.generateExpr(expr.Left, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(leftCode)

	if isRef {
		// Reference type: x2 is the pointer, check if non-null
		builder.WriteString(fmt.Sprintf("    cbnz x2, %s\n", endLabel))
	} else {
		// For primitive nullable, tag is in x3, value is in x2
		builder.WriteString(fmt.Sprintf("    cbnz x3, %s\n", endLabel))
	}

	// Null path: evaluate right operand (overwrites x2)
	rightCode, err := g.generateExpr(expr.Right, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(rightCode)

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

	case *semantic.TypedMethodCallExpr:
		code, err := g.generateMethodCallExpr(e, ctx)
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
	builder := strings.Builder{}

	// Calculate total slots needed
	totalSlots := g.countStructSlots(expr.Type)

	// Allocate temporary stack space
	baseOffset := ctx.stackOffset + StackAlignment
	ctx.stackOffset = baseOffset + (totalSlots-1)*StackAlignment

	// Generate and store all fields
	code, err := g.generateStructFieldsInline(expr, baseOffset, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Store the stack address of the struct in x2 for use by field access or method calls
	// Calculate address: x29 - baseOffset
	builder.WriteString(fmt.Sprintf("    sub x2, x29, #%d\n", baseOffset))

	return builder.String(), nil
}

// generateClassLiteral generates code for a class literal expression.
// This is called when a class is used as an expression (e.g., for method calls on temporaries).
func (g *TypedCodeGenerator) generateClassLiteral(expr *semantic.TypedClassLiteralExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Calculate total slots needed
	totalSlots := g.countClassSlots(expr.Type)

	// Allocate temporary stack space
	baseOffset := ctx.stackOffset + StackAlignment
	ctx.stackOffset = baseOffset + (totalSlots-1)*StackAlignment

	// Generate and store all fields
	code, err := g.generateClassFieldsInline(expr, baseOffset, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(code)

	// Store the stack address of the class instance in x2 for use by method calls
	// Calculate address: x29 - baseOffset
	builder.WriteString(fmt.Sprintf("    sub x2, x29, #%d\n", baseOffset))

	return builder.String(), nil
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

	// Check if the object is an owned pointer (Own<T>) - requires heap dereference
	if semantic.IsOwnedPointer(expr.Object.GetType()) {
		return g.generateFieldAccessThroughOwnedPointer(expr, ctx)
	}

	// Check if the object is a reference pointer (Ref<T> or MutRef<T>) - requires heap dereference
	if semantic.IsAnyRefPointer(expr.Object.GetType()) {
		return g.generateFieldAccessThroughRefPointer(expr, ctx)
	}

	// Check if this is a nested field access through a reference pointer (e.g., self.topLeft.x)
	// This happens when expr.Object is a TypedFieldAccessExpr that ultimately derives from a pointer
	if rootPtr, fields := g.getFieldAccessChainThroughPointer(expr); rootPtr != nil {
		return g.generateChainedFieldAccess(rootPtr, fields, expr.Type, ctx)
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

// generateFieldAccessThroughOwnedPointer generates code for field access through Own<T>.
// Loads the pointer from stack, then loads the field from the heap.
func (g *TypedCodeGenerator) generateFieldAccessThroughOwnedPointer(expr *semantic.TypedFieldAccessExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get the struct/class type from the owned pointer
	ownedType := expr.Object.GetType().(semantic.OwnedPointerType)
	var fields []semantic.StructFieldInfo
	if structType, ok := ownedType.ElementType.(semantic.StructType); ok {
		fields = structType.Fields
	} else if classType, ok := ownedType.ElementType.(semantic.ClassType); ok {
		fields = classType.Fields
	} else {
		return "", fmt.Errorf("field access through owned pointer requires struct or class type, got %s", ownedType.ElementType.String())
	}

	// Calculate the field byte offset within the struct/class (on heap, each field is 8 bytes)
	fieldByteOffset := 0
	for _, field := range fields {
		if field.Name == expr.Field {
			break
		}
		fieldByteOffset += 8 // Each field is 8-byte aligned on heap
	}

	// Generate the object expression (loads the pointer into x2)
	objCode, err := g.generateExpr(expr.Object, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(objCode)

	// x2 now contains the heap pointer
	// Load the field value from [x2 + fieldByteOffset]
	if semantic.IsFloatType(expr.Type) {
		if fieldByteOffset == 0 {
			builder.WriteString("    ldr d0, [x2]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x2, #%d]\n", fieldByteOffset))
		}
	} else {
		if fieldByteOffset == 0 {
			builder.WriteString("    ldr x2, [x2]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", fieldByteOffset))
		}
	}

	return builder.String(), nil
}

// generateFieldAccessThroughRefPointer generates code for field access through Ref<T>/MutRef<T>.
// Loads the pointer from stack, then loads the field from the heap.
// Very similar to generateFieldAccessThroughOwnedPointer since both are heap pointers.
func (g *TypedCodeGenerator) generateFieldAccessThroughRefPointer(expr *semantic.TypedFieldAccessExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Get the struct type from the ref pointer (either Ref<T> or MutRef<T>)
	objectType := expr.Object.GetType()
	var elementType semantic.Type
	if refType, ok := objectType.(semantic.RefPointerType); ok {
		elementType = refType.ElementType
	} else if mutRefType, ok := objectType.(semantic.MutRefPointerType); ok {
		elementType = mutRefType.ElementType
	} else {
		return "", fmt.Errorf("expected Ref<T> or MutRef<T>, got %s", objectType.String())
	}

	// Get fields from either StructType or ClassType
	var fields []semantic.StructFieldInfo
	if structType, ok := elementType.(semantic.StructType); ok {
		fields = structType.Fields
	} else if classType, ok := elementType.(semantic.ClassType); ok {
		fields = classType.Fields
	} else {
		return "", fmt.Errorf("field access through ref pointer requires struct or class type, got %s", elementType.String())
	}

	// Calculate the field byte offset within the struct/class (on heap, each field is 8 bytes)
	fieldByteOffset := 0
	for _, field := range fields {
		if field.Name == expr.Field {
			break
		}
		fieldByteOffset += 8 // Each field is 8-byte aligned on heap
	}

	// Generate the object expression (loads the pointer into x2)
	objCode, err := g.generateExpr(expr.Object, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(objCode)

	// x2 now contains the heap pointer
	// Load the field value from [x2 + fieldByteOffset]
	if semantic.IsFloatType(expr.Type) {
		if fieldByteOffset == 0 {
			builder.WriteString("    ldr d0, [x2]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x2, #%d]\n", fieldByteOffset))
		}
	} else {
		if fieldByteOffset == 0 {
			builder.WriteString("    ldr x2, [x2]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", fieldByteOffset))
		}
	}

	return builder.String(), nil
}

// getFieldAccessChainThroughPointer checks if a field access expression is part of a chain
// that ultimately accesses fields through a reference pointer (e.g., self.topLeft.x).
// Returns the root pointer expression and the list of field names in the chain, or nil if not applicable.
func (g *TypedCodeGenerator) getFieldAccessChainThroughPointer(expr *semantic.TypedFieldAccessExpr) (semantic.TypedExpression, []string) {
	var fields []string
	fields = append(fields, expr.Field)

	current := expr.Object
	for {
		switch obj := current.(type) {
		case *semantic.TypedFieldAccessExpr:
			// Add field to the chain and continue up
			fields = append([]string{obj.Field}, fields...)
			current = obj.Object

		case *semantic.TypedSelfExpr:
			// Found self - this is a pointer type
			return obj, fields

		case *semantic.TypedIdentifierExpr:
			// Check if this identifier is a reference pointer
			if semantic.IsAnyRefPointer(obj.Type) || semantic.IsOwnedPointer(obj.Type) {
				return obj, fields
			}
			// Not a pointer, return nil to fall back to static access
			return nil, nil

		default:
			// Unknown object type, return nil
			return nil, nil
		}
	}
}

// generateChainedFieldAccess generates code for accessing nested fields through a pointer.
// rootPtr is the pointer expression (e.g., self or a pointer variable)
// fields is the list of field names to access (e.g., ["topLeft", "x"])
func (g *TypedCodeGenerator) generateChainedFieldAccess(rootPtr semantic.TypedExpression, fields []string, resultType semantic.Type, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate code to load the root pointer into x2
	rootCode, err := g.generateExpr(rootPtr, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(rootCode)

	// Get the type of the struct/class pointed to by the root
	rootType := rootPtr.GetType()
	var currentFields []semantic.StructFieldInfo

	switch rt := rootType.(type) {
	case semantic.RefPointerType:
		if st, ok := rt.ElementType.(semantic.StructType); ok {
			currentFields = st.Fields
		} else if ct, ok := rt.ElementType.(semantic.ClassType); ok {
			currentFields = ct.Fields
		}
	case semantic.MutRefPointerType:
		if st, ok := rt.ElementType.(semantic.StructType); ok {
			currentFields = st.Fields
		} else if ct, ok := rt.ElementType.(semantic.ClassType); ok {
			currentFields = ct.Fields
		}
	case semantic.OwnedPointerType:
		if st, ok := rt.ElementType.(semantic.StructType); ok {
			currentFields = st.Fields
		} else if ct, ok := rt.ElementType.(semantic.ClassType); ok {
			currentFields = ct.Fields
		}
	default:
		return "", fmt.Errorf("expected pointer type for chained field access, got %s", rootType.String())
	}

	// Calculate the combined offset for all fields in the chain
	totalOffset := 0
	for i, fieldName := range fields {
		found := false
		offset := 0
		var nextFields []semantic.StructFieldInfo

		for _, f := range currentFields {
			if f.Name == fieldName {
				found = true
				// For the last field, we're done
				// For intermediate fields, get the nested struct's fields
				if i < len(fields)-1 {
					if st, ok := f.Type.(semantic.StructType); ok {
						nextFields = st.Fields
					} else if ct, ok := f.Type.(semantic.ClassType); ok {
						nextFields = ct.Fields
					}
				}
				break
			}
			offset += 8 // Each field is 8-byte aligned on heap
		}

		if !found {
			return "", fmt.Errorf("field '%s' not found in struct", fieldName)
		}

		totalOffset += offset
		currentFields = nextFields
	}

	// Load the field value from [x2 + totalOffset]
	if semantic.IsFloatType(resultType) {
		if totalOffset == 0 {
			builder.WriteString("    ldr d0, [x2]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    ldr d0, [x2, #%d]\n", totalOffset))
		}
	} else {
		if totalOffset == 0 {
			builder.WriteString("    ldr x2, [x2]\n")
		} else {
			builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", totalOffset))
		}
	}

	return builder.String(), nil
}

// generateSafeCallExpr generates code for safe call expression (e.g., person?.address)
// If object is null, returns null; otherwise returns the field value
// Result: x2 = value, x3 = tag (1 = not null, 0 = null) for primitive nullable results
func (g *TypedCodeGenerator) generateSafeCallExpr(expr *semantic.TypedSafeCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate unique labels for branching
	nullLabel := ctx.NextLabel("safe_null")
	doneLabel := ctx.NextLabel("safe_done")

	// Check if the inner type is a reference type (uses null pointer) or primitive (uses tagged union)
	isRef := semantic.IsReferenceType(expr.InnerType)

	// Check if the RESULT type is a primitive nullable (needs tag in x3)
	resultIsRefNullable := false
	if nullableResult, ok := expr.Type.(semantic.NullableType); ok {
		resultIsRefNullable = semantic.IsReferenceType(nullableResult.InnerType)
	}

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
		if !resultIsRefNullable {
			// Result is primitive nullable - set tag = 1 (not null)
			EmitMoveImm(&builder, "x3", "1")
		}
		builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

		// Null path: x2 is already 0
		builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
		// x2 already contains 0 (null)
		if !resultIsRefNullable {
			// Result is primitive nullable - set tag = 0 (null)
			EmitMoveImm(&builder, "x3", "0")
		}

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
			// x3 already has tag = 1 from loading above
			builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

			// Null path
			builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
			EmitMoveImm(&builder, "x2", "0")
			// x3 already has tag = 0 from loading above

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
	case "sleep":
		return g.generateSleepBuiltin(call, ctx)
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

func (g *TypedCodeGenerator) generateSleepBuiltin(call *semantic.TypedCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Generate the nanoseconds argument - result in x2
	if len(call.Arguments) > 0 {
		code, err := g.generateExpr(call.Arguments[0], ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	builder.WriteString("    // sleep builtin - x2 has nanoseconds\n")

	// Allocate timeval struct on stack (16 bytes: tv_sec + tv_usec)
	builder.WriteString("    sub sp, sp, #16\n")

	// Save nanoseconds to a safe register
	builder.WriteString("    mov x10, x2\n")

	// Load 1,000,000,000 for division to get seconds
	// 1,000,000,000 = 0x3B9ACA00
	builder.WriteString("    mov x11, #0xCA00\n")
	builder.WriteString("    movk x11, #0x3B9A, lsl #16\n")

	// tv_sec = ns / 1,000,000,000
	builder.WriteString("    sdiv x12, x10, x11\n")
	builder.WriteString("    str x12, [sp]\n")

	// remainder = ns % 1,000,000,000 (ns - sec * 1B)
	builder.WriteString("    msub x13, x12, x11, x10\n")

	// tv_usec = remainder / 1000 (convert ns remainder to microseconds)
	builder.WriteString("    mov x11, #1000\n")
	builder.WriteString("    sdiv x13, x13, x11\n")
	builder.WriteString("    str x13, [sp, #8]\n")

	// Call select(0, NULL, NULL, NULL, &timeval)
	builder.WriteString("    mov x0, #0\n")
	builder.WriteString("    mov x1, #0\n")
	builder.WriteString("    mov x2, #0\n")
	builder.WriteString("    mov x3, #0\n")
	builder.WriteString("    mov x4, sp\n")
	builder.WriteString("    mov x16, #93\n") // SYS_select
	builder.WriteString("    svc #0x80\n")

	// Clean up stack
	builder.WriteString("    add sp, sp, #16\n")

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

// ============================================================================
// Heap Allocation (Own<T> pointers)
// ============================================================================

// generateMethodCallExpr generates code for method call expressions.
// Supports:
// - Heap.new(expr) - allocates memory on the heap and returns Own<T>
// - p.copy() - creates a deep copy of an owned pointer (Phase 9)
// - ClassName.method(args) - static method call on a class
// - ObjectName.method(args) - method call on a singleton object
// - instance.method(args) - instance method call on a class instance
func (g *TypedCodeGenerator) generateMethodCallExpr(expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	// Check if this is a Heap.new() call
	if ident, ok := expr.Object.(*semantic.TypedIdentifierExpr); ok {
		if ident.Name == "Heap" && expr.Method == "new" {
			return g.generateHeapNew(expr, ctx)
		}
	}

	// Check if this is a .copy() call on an owned pointer
	if expr.Method == "copy" {
		objectType := expr.Object.GetType()
		if _, isOwned := objectType.(semantic.OwnedPointerType); isOwned {
			return g.generateCopy(expr, ctx)
		}
	}

	// Check if this is a static method call on a class or object
	if ident, ok := expr.Object.(*semantic.TypedIdentifierExpr); ok {
		// Check if this identifier is a variable in scope (instance method call)
		// vs. a class/object type name (static method call)
		_, isVariable := ctx.GetVariable(ident.Name)
		if !isVariable {
			// Not a variable - this is a static method call on the class/object type
			switch t := ident.Type.(type) {
			case semantic.ClassType:
				return g.generateClassStaticMethodCall(t.Name, expr, ctx)
			case semantic.ObjectType:
				return g.generateObjectMethodCall(t.Name, expr, ctx)
			}
		}
		// If it's a variable, fall through to instance method call below
	}

	// Check if this is an instance method call
	objectType := expr.Object.GetType()
	if className := g.getClassNameFromType(objectType); className != "" {
		return g.generateClassInstanceMethodCall(className, expr, ctx)
	}

	return "", fmt.Errorf("unsupported method call: %s.%s", expr.Object.GetType().String(), expr.Method)
}

// getClassNameFromType extracts the class name from a type that contains a ClassType.
// Returns empty string if the type is not a class or pointer to class.
// Also handles nullable types by unwrapping them first.
func (g *TypedCodeGenerator) getClassNameFromType(t semantic.Type) string {
	// Handle nullable types by unwrapping them
	if nullableType, ok := t.(semantic.NullableType); ok {
		t = nullableType.InnerType
	}

	switch typ := t.(type) {
	case semantic.ClassType:
		return typ.Name
	case semantic.OwnedPointerType:
		if ct, ok := typ.ElementType.(semantic.ClassType); ok {
			return ct.Name
		}
	case semantic.RefPointerType:
		if ct, ok := typ.ElementType.(semantic.ClassType); ok {
			return ct.Name
		}
	case semantic.MutRefPointerType:
		if ct, ok := typ.ElementType.(semantic.ClassType); ok {
			return ct.Name
		}
	}
	return ""
}

// generateClassStaticMethodCall generates code for a static method call on a class.
// Static methods don't have a receiver, so arguments go directly to x0, x1, ...
func (g *TypedCodeGenerator) generateClassStaticMethodCall(className string, expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Evaluate arguments and place in registers x0, x1, x2, ...
	for i, arg := range expr.Arguments {
		argCode, err := g.generateExpr(arg, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(argCode)
		// Result is in x2, move to argument register
		builder.WriteString(fmt.Sprintf("    mov x%d, x2\n", i))
	}

	// Call the method - use ResolvedMethod for overload disambiguation
	var mangledName string
	if expr.ResolvedMethod != nil {
		mangledName = mangleMethodNameWithInfo(className, expr.ResolvedMethod)
	} else {
		mangledName = mangleMethodName(className, expr.Method)
	}
	builder.WriteString(fmt.Sprintf("    bl _%s\n", mangledName))

	// Handle nullable return types: method returns x0 = tag, x1 = value
	// For nullable returns, move value from x1 to x2 and tag from x0 to x3
	if semantic.IsNullable(expr.Type) && !semantic.IsReferenceType(expr.Type.(semantic.NullableType).InnerType) {
		EmitMoveReg(&builder, "x2", "x1")
		EmitMoveReg(&builder, "x3", "x0")
	} else {
		// Non-nullable or reference-type nullable: just move x0 to x2
		builder.WriteString("    mov x2, x0\n")
	}

	return builder.String(), nil
}

// generateMethodCallWithDestination generates a method call where x8 already contains
// the destination address for a class return value. This is used when the caller has
// pre-allocated space for the return value.
func (g *TypedCodeGenerator) generateMethodCallWithDestination(expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// Check if this is a static method call on a class
	if ident, ok := expr.Object.(*semantic.TypedIdentifierExpr); ok {
		_, isVariable := ctx.GetVariable(ident.Name)
		if !isVariable {
			if classType, isClass := ident.Type.(semantic.ClassType); isClass {
				// Static method call - evaluate arguments and place in registers
				for i, arg := range expr.Arguments {
					argCode, err := g.generateExpr(arg, ctx)
					if err != nil {
						return "", err
					}
					builder.WriteString(argCode)
					builder.WriteString(fmt.Sprintf("    mov x%d, x2\n", i))
				}

				// Call the method (x8 is already set by caller)
				var mangledName string
				if expr.ResolvedMethod != nil {
					mangledName = mangleMethodNameWithInfo(classType.Name, expr.ResolvedMethod)
				} else {
					mangledName = mangleMethodName(classType.Name, expr.Method)
				}
				builder.WriteString(fmt.Sprintf("    bl _%s\n", mangledName))

				return builder.String(), nil
			}
		}
	}

	// Instance method call
	className := g.getClassNameFromType(expr.Object.GetType())
	if className == "" {
		return "", fmt.Errorf("cannot determine class name for method call")
	}

	// Save arguments to stack first
	argCount := len(expr.Arguments)
	if argCount > 0 {
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", argCount*16))
		for i, arg := range expr.Arguments {
			argCode, err := g.generateExpr(arg, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(argCode)
			builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", i*16))
		}
	}

	// Evaluate receiver - for stack-allocated class, compute address
	if ident, isIdent := expr.Object.(*semantic.TypedIdentifierExpr); isIdent {
		if _, isClass := ident.Type.(semantic.ClassType); isClass {
			slot, ok := ctx.GetVariable(ident.Name)
			if !ok {
				return "", fmt.Errorf("undefined variable: %s", ident.Name)
			}
			builder.WriteString(fmt.Sprintf("    sub x2, x29, #%d\n", slot.Offset))
		} else {
			receiverCode, err := g.generateExpr(expr.Object, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(receiverCode)
		}
	} else {
		receiverCode, err := g.generateExpr(expr.Object, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(receiverCode)
	}

	// Move receiver to x0
	builder.WriteString("    mov x0, x2\n")

	// Restore arguments from stack
	if argCount > 0 {
		for i := 0; i < argCount; i++ {
			builder.WriteString(fmt.Sprintf("    ldr x%d, [sp, #%d]\n", i+1, i*16))
		}
		builder.WriteString(fmt.Sprintf("    add sp, sp, #%d\n", argCount*16))
	}

	// Call the method (x8 is already set by caller)
	var mangledName string
	if expr.ResolvedMethod != nil {
		mangledName = mangleMethodNameWithInfo(className, expr.ResolvedMethod)
	} else {
		mangledName = mangleMethodName(className, expr.Method)
	}
	builder.WriteString(fmt.Sprintf("    bl _%s\n", mangledName))

	return builder.String(), nil
}

// generateObjectMethodCall generates code for a method call on a singleton object.
// Object methods are always static, so they work the same as class static methods.
func (g *TypedCodeGenerator) generateObjectMethodCall(objectName string, expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	// Object methods work exactly like class static methods
	return g.generateClassStaticMethodCall(objectName, expr, ctx)
}

// generateClassInstanceMethodCall generates code for an instance method call.
// Instance methods have 'self' as the first argument (in x0), other args in x1, x2, ...
// Supports safe navigation (?.method()) where receiver is nullable.
func (g *TypedCodeGenerator) generateClassInstanceMethodCall(className string, expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	// For safe navigation, we need labels for null checking
	var nullLabel, doneLabel string
	if expr.SafeNavigation {
		nullLabel = ctx.NextLabel("safe_null")
		doneLabel = ctx.NextLabel("safe_done")
	}

	// First, evaluate all arguments and save them to the stack
	// We need to do this before evaluating the receiver since both may use x2
	argCount := len(expr.Arguments)
	if argCount > 0 {
		// Pre-allocate stack space for arguments
		builder.WriteString(fmt.Sprintf("    sub sp, sp, #%d\n", argCount*16))
		for i, arg := range expr.Arguments {
			argCode, err := g.generateExpr(arg, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(argCode)
			// Save argument to pre-allocated stack slot
			builder.WriteString(fmt.Sprintf("    str x2, [sp, #%d]\n", i*16))
		}
	}

	// Evaluate the receiver (instance) - result in x2
	// For stack-allocated class/struct variables, we need the address, not the value
	if ident, isIdent := expr.Object.(*semantic.TypedIdentifierExpr); isIdent {
		if _, isClass := ident.Type.(semantic.ClassType); isClass {
			// Stack-allocated class: compute address
			slot, ok := ctx.GetVariable(ident.Name)
			if !ok {
				return "", fmt.Errorf("undefined variable: %s", ident.Name)
			}
			builder.WriteString(fmt.Sprintf("    sub x2, x29, #%d\n", slot.Offset))
		} else if _, isStruct := ident.Type.(semantic.StructType); isStruct {
			// Stack-allocated struct: compute address
			slot, ok := ctx.GetVariable(ident.Name)
			if !ok {
				return "", fmt.Errorf("undefined variable: %s", ident.Name)
			}
			builder.WriteString(fmt.Sprintf("    sub x2, x29, #%d\n", slot.Offset))
		} else {
			// Pointer type (heap-allocated): load the pointer value
			receiverCode, err := g.generateExpr(expr.Object, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(receiverCode)
		}
	} else if classLit, isClassLit := expr.Object.(*semantic.TypedClassLiteralExpr); isClassLit {
		// Class literal as receiver (e.g., Point{ 3, 4 }.method())
		// generateClassLiteral allocates stack space and puts address in x2
		receiverCode, err := g.generateClassLiteral(classLit, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(receiverCode)
	} else {
		// Other expressions (e.g., pointer dereference, field access)
		receiverCode, err := g.generateExpr(expr.Object, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(receiverCode)
	}

	// For safe navigation, check if receiver is null before proceeding
	if expr.SafeNavigation {
		// x2 contains the receiver (pointer). If null (0), skip the call
		builder.WriteString(fmt.Sprintf("    cbz x2, %s\n", nullLabel))
	}

	// Move receiver to x0 (first argument for instance methods)
	builder.WriteString("    mov x0, x2\n")

	// Restore arguments from stack to registers x1, x2, ...
	if argCount > 0 {
		for i := 0; i < argCount; i++ {
			builder.WriteString(fmt.Sprintf("    ldr x%d, [sp, #%d]\n", i+1, i*16))
		}
		// Restore stack pointer
		builder.WriteString(fmt.Sprintf("    add sp, sp, #%d\n", argCount*16))
	}

	// Call the method - use ResolvedMethod for overload disambiguation
	var mangledName string
	if expr.ResolvedMethod != nil {
		mangledName = mangleMethodNameWithInfo(className, expr.ResolvedMethod)
	} else {
		mangledName = mangleMethodName(className, expr.Method)
	}
	builder.WriteString(fmt.Sprintf("    bl _%s\n", mangledName))

	// Handle nullable return types: method returns x0 = tag, x1 = value
	// For nullable returns, move value from x1 to x2 and tag from x0 to x3
	methodReturnsNullable := semantic.IsNullable(expr.Type) && !semantic.IsReferenceType(expr.Type.(semantic.NullableType).InnerType)
	if methodReturnsNullable && !expr.SafeNavigation {
		// Method itself returns nullable (not via safe navigation)
		EmitMoveReg(&builder, "x2", "x1")
		EmitMoveReg(&builder, "x3", "x0")
	} else {
		// Non-nullable or safe navigation handles it below
		builder.WriteString("    mov x2, x0\n")
	}

	// For safe navigation, jump over the null path
	if expr.SafeNavigation {
		// Check if result is a primitive nullable (needs tag in x3)
		resultIsPrimitiveNullable := false
		if nullableResult, ok := expr.Type.(semantic.NullableType); ok {
			resultIsPrimitiveNullable = !semantic.IsReferenceType(nullableResult.InnerType)
		}

		// Non-null path: set tag to 1 if result is primitive nullable
		if resultIsPrimitiveNullable {
			EmitMoveImm(&builder, "x3", "1")
		}
		builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

		// Null path: receiver was null, so result is null (0)
		builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
		// If we pre-allocated stack space for args, we need to clean it up
		if argCount > 0 {
			builder.WriteString(fmt.Sprintf("    add sp, sp, #%d\n", argCount*16))
		}
		// Set result to null (0)
		EmitMoveImm(&builder, "x2", "0")
		// Set tag to 0 if result is primitive nullable
		if resultIsPrimitiveNullable {
			EmitMoveImm(&builder, "x3", "0")
		}

		// Done label
		builder.WriteString(fmt.Sprintf("%s:\n", doneLabel))
	}

	return builder.String(), nil
}

// generateHeapNew generates code for Heap.new(expr).
// This allocates memory using mmap and stores the value at the allocated address.
// Result: pointer in x2
func (g *TypedCodeGenerator) generateHeapNew(expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	if len(expr.Arguments) != 1 {
		return "", fmt.Errorf("Heap.new() expects exactly 1 argument, got %d", len(expr.Arguments))
	}

	arg := expr.Arguments[0]
	argType := arg.GetType()

	// Calculate allocation size based on the type
	allocSize := g.calculateHeapAllocSize(argType)

	// For struct types, we need to handle them specially
	if structType, isStruct := argType.(semantic.StructType); isStruct {
		return g.generateHeapNewStruct(expr, structType, allocSize, ctx)
	}

	// For class types, handle them similarly to structs
	if classType, isClass := argType.(semantic.ClassType); isClass {
		return g.generateHeapNewClass(expr, classType, allocSize, ctx)
	}

	// For primitive types: generate value, allocate, store, return pointer

	// Step 1: Generate the value expression (result in x2 or d0)
	valueCode, err := g.generateExpr(arg, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(valueCode)

	// Step 2: Save the value to stack temporarily (we need x0-x5 for mmap)
	if semantic.IsFloatType(argType) {
		builder.WriteString("    str d0, [sp, #-16]!\n")
	} else {
		builder.WriteString("    str x2, [sp, #-16]!\n")
	}

	// Step 3: Call mmap to allocate memory
	builder.WriteString(g.emitMmapAlloc(allocSize))

	// x0 now contains the allocated pointer (or error if negative)
	// Move pointer to x4 for safekeeping
	builder.WriteString("    mov x4, x0\n")

	// Step 4: Restore the value from stack
	if semantic.IsFloatType(argType) {
		builder.WriteString("    ldr d0, [sp], #16\n")
		// Store float value at allocated address
		builder.WriteString("    str d0, [x4]\n")
	} else {
		builder.WriteString("    ldr x2, [sp], #16\n")
		// Store value at allocated address
		builder.WriteString("    str x2, [x4]\n")
	}

	// Step 5: Return the pointer in x2
	builder.WriteString("    mov x2, x4\n")

	return builder.String(), nil
}

// generateHeapNewStruct generates code for Heap.new(StructLiteral).
// Allocates memory for the struct and stores all fields.
func (g *TypedCodeGenerator) generateHeapNewStruct(expr *semantic.TypedMethodCallExpr, structType semantic.StructType, allocSize int, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	arg := expr.Arguments[0]
	structLit, ok := arg.(*semantic.TypedStructLiteralExpr)
	if !ok {
		return "", fmt.Errorf("Heap.new() with struct type requires struct literal, got %T", arg)
	}

	// Step 1: Call mmap to allocate memory first
	builder.WriteString(g.emitMmapAlloc(allocSize))

	// x0 now contains the allocated pointer
	// Save it to a callee-saved register or stack
	builder.WriteString("    str x0, [sp, #-16]!\n") // save pointer

	// Step 2: Generate and store each field
	currentOffset := 0
	for i, fieldArg := range structLit.Args {
		fieldType := structType.Fields[i].Type

		// Check if this field is an owned pointer being moved into the struct
		// If so, mark it as moved so it won't be freed at scope exit
		if ident, isIdent := fieldArg.(*semantic.TypedIdentifierExpr); isIdent {
			if semantic.IsOwnedPointer(ident.Type) || semantic.IsNullableOwnedPointer(ident.Type) {
				ctx.MarkOwnedVarMoved(ident.Name)
			}
		}

		// Generate the field value
		fieldCode, err := g.generateExpr(fieldArg, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(fieldCode)

		// Load the base pointer back (it might have been clobbered)
		builder.WriteString("    ldr x4, [sp]\n")

		// Store the field value at the correct offset
		if semantic.IsFloatType(fieldType) {
			builder.WriteString(fmt.Sprintf("    str d0, [x4, #%d]\n", currentOffset))
			currentOffset += 8
		} else {
			builder.WriteString(fmt.Sprintf("    str x2, [x4, #%d]\n", currentOffset))
			currentOffset += 8 // All fields are 8-byte aligned on heap
		}
	}

	// Step 3: Restore the pointer and return it in x2
	builder.WriteString("    ldr x2, [sp], #16\n")

	return builder.String(), nil
}

// generateHeapNewClass generates code for Heap.new(ClassLiteral).
// Allocates memory for the class instance and stores all fields.
func (g *TypedCodeGenerator) generateHeapNewClass(expr *semantic.TypedMethodCallExpr, classType semantic.ClassType, allocSize int, ctx *BaseContext) (string, error) {
	builder := strings.Builder{}

	arg := expr.Arguments[0]
	classLit, ok := arg.(*semantic.TypedClassLiteralExpr)
	if !ok {
		return "", fmt.Errorf("Heap.new() with class type requires class literal, got %T", arg)
	}

	// Step 1: Call mmap to allocate memory first
	builder.WriteString(g.emitMmapAlloc(allocSize))

	// x0 now contains the allocated pointer
	// Save it to a callee-saved register or stack
	builder.WriteString("    str x0, [sp, #-16]!\n") // save pointer

	// Step 2: Generate and store each field
	currentOffset := 0
	for i, fieldArg := range classLit.Args {
		fieldType := classType.Fields[i].Type

		// Check if this field is an owned pointer being moved into the class
		// If so, mark it as moved so it won't be freed at scope exit
		if ident, isIdent := fieldArg.(*semantic.TypedIdentifierExpr); isIdent {
			if semantic.IsOwnedPointer(ident.Type) || semantic.IsNullableOwnedPointer(ident.Type) {
				ctx.MarkOwnedVarMoved(ident.Name)
			}
		}

		// Generate the field value
		fieldCode, err := g.generateExpr(fieldArg, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(fieldCode)

		// Load the base pointer back (it might have been clobbered)
		builder.WriteString("    ldr x4, [sp]\n")

		// Store the field value at the correct offset
		if semantic.IsFloatType(fieldType) {
			builder.WriteString(fmt.Sprintf("    str d0, [x4, #%d]\n", currentOffset))
			currentOffset += 8
		} else {
			builder.WriteString(fmt.Sprintf("    str x2, [x4, #%d]\n", currentOffset))
			currentOffset += 8 // All fields are 8-byte aligned on heap
		}
	}

	// Step 3: Restore the pointer and return it in x2
	builder.WriteString("    ldr x2, [sp], #16\n")

	return builder.String(), nil
}

// calculateHeapAllocSize returns the number of bytes to allocate for a type on the heap.
// For structs, each field uses 8 bytes (pointer-aligned).
// Minimum allocation is 16 bytes for page alignment purposes.
func (g *TypedCodeGenerator) calculateHeapAllocSize(t semantic.Type) int {
	switch tt := t.(type) {
	case semantic.StructType:
		// Each field is 8 bytes on heap (pointer-aligned)
		size := len(tt.Fields) * 8
		// Minimum allocation size
		if size < 16 {
			return 16
		}
		// Round up to 16-byte alignment
		return (size + 15) & ^15
	case semantic.ClassType:
		// Each field is 8 bytes on heap (pointer-aligned)
		size := len(tt.Fields) * 8
		// Minimum allocation size
		if size < 16 {
			return 16
		}
		// Round up to 16-byte alignment
		return (size + 15) & ^15
	default:
		// Primitives: allocate 16 bytes (minimum for alignment)
		return 16
	}
}

// emitMmapAlloc generates ARM64 code to allocate memory using the bump allocator.
// Returns the assembly code as a string.
// After execution, x0 contains the allocated pointer.
func (g *TypedCodeGenerator) emitMmapAlloc(size int) string {
	var builder strings.Builder

	// Call the bump allocator runtime function
	// Input: x0 = size
	// Output: x0 = allocated pointer
	builder.WriteString(fmt.Sprintf("    // allocate %d bytes\n", size))
	builder.WriteString(fmt.Sprintf("    mov x0, #%d\n", size))
	builder.WriteString("    bl _sl_alloc\n")
	builder.WriteString("    // x0 now contains allocated pointer\n")

	return builder.String()
}

// emitMunmap generates ARM64 code to deallocate memory using the bump allocator free list.
// ptrOffset is the stack offset where the pointer is stored.
// size is the allocation size to free.
func (g *TypedCodeGenerator) emitMunmap(ptrOffset int, size int) string {
	var builder strings.Builder

	// Call the bump allocator free function
	// Input: x0 = pointer, x1 = size
	builder.WriteString(fmt.Sprintf("    // free %d bytes\n", size))
	builder.WriteString(fmt.Sprintf("    ldr x0, [x29, #-%d]\n", ptrOffset)) // load pointer
	builder.WriteString(fmt.Sprintf("    mov x1, #%d\n", size))               // size
	builder.WriteString("    bl _sl_free\n")

	return builder.String()
}

// generateOwnedVarCleanup generates munmap calls for all owned pointer variables.
// Variables are freed in reverse declaration order (LIFO).
// skipVar is the name of a variable to skip (e.g., when returning it).
func (g *TypedCodeGenerator) generateOwnedVarCleanup(ctx *BaseContext, skipVar string) string {
	var builder strings.Builder

	ownedVars := ctx.GetOwnedVars()
	if len(ownedVars) == 0 {
		return ""
	}

	// Count how many variables we'll actually clean up
	cleanupCount := 0
	for _, v := range ownedVars {
		if v.Name != "" && v.Name != skipVar {
			cleanupCount++
		}
	}
	if cleanupCount == 0 {
		return ""
	}

	builder.WriteString("    // cleanup owned pointers\n")
	// Save x0 (return value) before cleanup - munmap uses x0 for address
	builder.WriteString("    mov x9, x0\n")

	// Free in reverse order (LIFO)
	for i := len(ownedVars) - 1; i >= 0; i-- {
		v := ownedVars[i]
		// Skip empty names (moved variables) and the skip variable
		if v.Name == "" || v.Name == skipVar {
			continue
		}
		builder.WriteString(g.emitMunmap(v.Offset, v.AllocSize))
	}

	// Restore x0 (return value) after cleanup
	builder.WriteString("    mov x0, x9\n")

	return builder.String()
}

// ============================================================================
// Deep Copy (Own<T>.copy())
// ============================================================================

// generateCopy generates code for p.copy() on an owned pointer.
// This creates a new allocation and deep copies the value.
// Result: new pointer in x2
func (g *TypedCodeGenerator) generateCopy(expr *semantic.TypedMethodCallExpr, ctx *BaseContext) (string, error) {
	var builder strings.Builder

	// Get the owned pointer type
	ownedType, ok := expr.Object.GetType().(semantic.OwnedPointerType)
	if !ok {
		return "", fmt.Errorf(".copy() called on non-owned type: %s", expr.Object.GetType().String())
	}

	elementType := ownedType.ElementType

	// Step 1: Generate the source pointer expression (result in x2)
	sourceCode, err := g.generateExpr(expr.Object, ctx)
	if err != nil {
		return "", err
	}
	builder.WriteString(sourceCode)

	// Step 2: Save source pointer to stack (we need registers for mmap)
	builder.WriteString("    // copy: save source pointer\n")
	builder.WriteString("    str x2, [sp, #-16]!\n")

	// Step 3: Calculate allocation size and allocate new memory
	allocSize := g.calculateHeapAllocSize(elementType)
	builder.WriteString(g.emitMmapAlloc(allocSize))

	// x0 now contains the new pointer
	// Save new pointer to x4
	builder.WriteString("    mov x4, x0\n")

	// Step 4: Restore source pointer to x5
	builder.WriteString("    ldr x5, [sp], #16\n")

	// Step 5: Copy the data based on element type
	if structType, isStruct := elementType.(semantic.StructType); isStruct {
		// Copy struct fields
		copyCode, err := g.generateCopyStructFields(structType, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(copyCode)
	} else {
		// Copy primitive value (single 8-byte value)
		builder.WriteString("    // copy: copy primitive value\n")
		if semantic.IsFloatType(elementType) {
			builder.WriteString("    ldr d0, [x5]\n")
			builder.WriteString("    str d0, [x4]\n")
		} else {
			builder.WriteString("    ldr x6, [x5]\n")
			builder.WriteString("    str x6, [x4]\n")
		}
	}

	// Step 6: Return new pointer in x2
	builder.WriteString("    mov x2, x4\n")

	return builder.String(), nil
}

// generateCopyStructFields generates code to copy struct fields from x5 (source) to x4 (dest).
// For fields containing Own<T>, it recursively deep copies them.
func (g *TypedCodeGenerator) generateCopyStructFields(structType semantic.StructType, ctx *BaseContext) (string, error) {
	var builder strings.Builder

	builder.WriteString("    // copy: copy struct fields\n")

	currentOffset := 0
	for _, field := range structType.Fields {
		// Check if this field contains an owned pointer that needs deep copy
		if ownedField, isOwned := field.Type.(semantic.OwnedPointerType); isOwned {
			// Deep copy: recursively copy the owned pointer
			deepCopyCode, err := g.generateDeepCopyOwnedField(ownedField, currentOffset, ctx)
			if err != nil {
				return "", err
			}
			builder.WriteString(deepCopyCode)
		} else if nullableType, isNullable := field.Type.(semantic.NullableType); isNullable {
			// Check if nullable wraps an owned pointer: Own<T>?
			if ownedField, isOwned := nullableType.InnerType.(semantic.OwnedPointerType); isOwned {
				// Deep copy nullable owned pointer (null-safe)
				deepCopyCode, err := g.generateDeepCopyNullableOwnedField(ownedField, currentOffset, ctx)
				if err != nil {
					return "", err
				}
				builder.WriteString(deepCopyCode)
			} else {
				// Shallow copy: other nullable types
				builder.WriteString(fmt.Sprintf("    ldr x6, [x5, #%d]\n", currentOffset))
				builder.WriteString(fmt.Sprintf("    str x6, [x4, #%d]\n", currentOffset))
			}
		} else if semantic.IsFloatType(field.Type) {
			// Shallow copy: copy float value directly
			builder.WriteString(fmt.Sprintf("    ldr d0, [x5, #%d]\n", currentOffset))
			builder.WriteString(fmt.Sprintf("    str d0, [x4, #%d]\n", currentOffset))
		} else {
			// Shallow copy: copy integer/boolean/pointer value directly
			builder.WriteString(fmt.Sprintf("    ldr x6, [x5, #%d]\n", currentOffset))
			builder.WriteString(fmt.Sprintf("    str x6, [x4, #%d]\n", currentOffset))
		}
		currentOffset += 8 // all fields are 8-byte aligned
	}

	return builder.String(), nil
}

// generateDeepCopyOwnedField generates code to deep copy an Own<T> field.
// Source struct is at x5, dest struct is at x4.
// The field is at the given offset in both structs.
func (g *TypedCodeGenerator) generateDeepCopyOwnedField(ownedType semantic.OwnedPointerType, fieldOffset int, ctx *BaseContext) (string, error) {
	var builder strings.Builder

	elementType := ownedType.ElementType
	allocSize := g.calculateHeapAllocSize(elementType)

	builder.WriteString(fmt.Sprintf("    // deep copy: Own<%s> field at offset %d\n", elementType.String(), fieldOffset))

	// Save x4 (dest struct ptr) and x5 (source struct ptr) to stack
	builder.WriteString("    stp x4, x5, [sp, #-16]!\n")

	// Load the source owned pointer (the pointer stored in the source struct field)
	builder.WriteString(fmt.Sprintf("    ldr x5, [x5, #%d]\n", fieldOffset))
	// Save source inner pointer to stack
	builder.WriteString("    str x5, [sp, #-16]!\n")

	// Allocate new memory for the nested value
	builder.WriteString(g.emitMmapAlloc(allocSize))

	// x0 = new nested pointer
	builder.WriteString("    mov x4, x0\n")

	// Restore source inner pointer to x5
	builder.WriteString("    ldr x5, [sp], #16\n")

	// Copy the nested value
	if nestedStruct, isStruct := elementType.(semantic.StructType); isStruct {
		copyCode, err := g.generateCopyStructFields(nestedStruct, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(copyCode)
	} else {
		// Copy primitive
		if semantic.IsFloatType(elementType) {
			builder.WriteString("    ldr d0, [x5]\n")
			builder.WriteString("    str d0, [x4]\n")
		} else {
			builder.WriteString("    ldr x6, [x5]\n")
			builder.WriteString("    str x6, [x4]\n")
		}
	}

	// x4 now contains the new nested pointer
	// Save it temporarily
	builder.WriteString("    mov x6, x4\n")

	// Restore dest struct ptr (x4) and source struct ptr (x5)
	builder.WriteString("    ldp x4, x5, [sp], #16\n")

	// Store the new nested pointer into the dest struct field
	builder.WriteString(fmt.Sprintf("    str x6, [x4, #%d]\n", fieldOffset))

	return builder.String(), nil
}

// generateDeepCopyNullableOwnedField generates code to deep copy an Own<T>? field.
// Source struct is at x5, dest struct is at x4.
// If the source field is null, copies null. Otherwise deep copies the owned pointer.
func (g *TypedCodeGenerator) generateDeepCopyNullableOwnedField(ownedType semantic.OwnedPointerType, fieldOffset int, ctx *BaseContext) (string, error) {
	var builder strings.Builder

	elementType := ownedType.ElementType
	allocSize := g.calculateHeapAllocSize(elementType)

	// Generate unique labels for this copy operation
	labelID := ctx.NextLabelID()
	nullLabel := fmt.Sprintf("_copy_null_%d", labelID)
	doneLabel := fmt.Sprintf("_copy_done_%d", labelID)

	builder.WriteString(fmt.Sprintf("    // deep copy: Own<%s>? field at offset %d\n", elementType.String(), fieldOffset))

	// Load source field value
	builder.WriteString(fmt.Sprintf("    ldr x6, [x5, #%d]\n", fieldOffset))

	// Check if null
	builder.WriteString(fmt.Sprintf("    cbz x6, %s\n", nullLabel))

	// Not null - deep copy the owned pointer
	// Save x4 (dest struct ptr) and x5 (source struct ptr) to stack
	builder.WriteString("    stp x4, x5, [sp, #-16]!\n")

	// x6 contains the source owned pointer, move to x5 for recursive copy
	builder.WriteString("    mov x5, x6\n")
	// Save source inner pointer to stack
	builder.WriteString("    str x5, [sp, #-16]!\n")

	// Allocate new memory for the nested value
	builder.WriteString(g.emitMmapAlloc(allocSize))

	// x0 = new nested pointer
	builder.WriteString("    mov x4, x0\n")

	// Restore source inner pointer to x5
	builder.WriteString("    ldr x5, [sp], #16\n")

	// Copy the nested value
	if nestedStruct, isStruct := elementType.(semantic.StructType); isStruct {
		copyCode, err := g.generateCopyStructFields(nestedStruct, ctx)
		if err != nil {
			return "", err
		}
		builder.WriteString(copyCode)
	} else {
		// Copy primitive
		if semantic.IsFloatType(elementType) {
			builder.WriteString("    ldr d0, [x5]\n")
			builder.WriteString("    str d0, [x4]\n")
		} else {
			builder.WriteString("    ldr x6, [x5]\n")
			builder.WriteString("    str x6, [x4]\n")
		}
	}

	// x4 now contains the new nested pointer
	// Save it temporarily
	builder.WriteString("    mov x6, x4\n")

	// Restore dest struct ptr (x4) and source struct ptr (x5)
	builder.WriteString("    ldp x4, x5, [sp], #16\n")

	// Store the new nested pointer into the dest struct field
	builder.WriteString(fmt.Sprintf("    str x6, [x4, #%d]\n", fieldOffset))

	// Jump to done
	builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

	// Null case - just store null
	builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
	builder.WriteString(fmt.Sprintf("    str xzr, [x4, #%d]\n", fieldOffset))

	builder.WriteString(fmt.Sprintf("%s:\n", doneLabel))

	return builder.String(), nil
}

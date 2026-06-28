package ir

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// binaryOpMap maps operator strings to IR opcodes for standard binary operations.
var binaryOpMap = map[string]Op{
	"+":  OpAdd,
	"-":  OpSub,
	"*":  OpMul,
	"/":  OpDiv,
	"%":  OpMod,
	"==": OpEq,
	"!=": OpNe,
	"<":  OpLt,
	"<=": OpLe,
	">":  OpGt,
	">=": OpGe,
}

// ownedVar tracks an owned pointer variable for automatic cleanup
type ownedVar struct {
	name    string
	semType semantic.Type
}

// Generator converts a TypedProgram into IR.
type Generator struct {
	prog *Program

	// Current function being generated
	fn *Function

	// Current block being generated
	block *Block

	// SSA construction
	ssa *SSABuilder

	// Type mapping from semantic types to IR types
	typeCache map[semantic.Type]Type

	// Loop control flow targets
	breakTarget    *Block
	continueTarget *Block
	// loopScopeDepth is the index of the innermost loop body's scope in
	// ownedVarScopes. break/continue free every scope from the current innermost
	// one down to (and including) this depth before jumping out of the body.
	loopScopeDepth int

	// Scope tracking for owned pointer cleanup
	// Each scope level contains owned pointers declared at that level
	ownedVarScopes [][]ownedVar

	// Track value-type-nullable variables whose heap slot has been aliased into
	// another binding/array. These must not be freed at scope exit, or the new
	// owner would be double-freed. Non-copyable types (*T, classes) can never be
	// aliased — semantic rejects that — so this only ever holds
	// copyable-but-heap-backed value nullables.
	aliasedHeapVars map[string]bool

	// Top-level statements (val/var) to inject at the start of main.
	// Each statement carries the package prefix to apply during generation.
	topLevelStmts []PrefixedStmt

	// Package prefix for variable name mangling (e.g., "math__" for non-root packages)
	packagePrefix string

	// Global variables — names that should use OpLoadGlobal/OpStoreGlobal
	globalVars map[string]Type

	// funcSemanticParams maps a (mangled) function name to its parameters'
	// semantic types. The IR collapses owned/borrow pointer kinds into a
	// single PtrType, so own-vs-borrow at call sites must be decided from
	// the original semantic param types.
	funcSemanticParams map[string][]semantic.Type
}

// GeneratorConfig holds options for IR generation.
type GeneratorConfig struct {
	// PackagePrefix is prepended to variable names (e.g., "math__" for non-root packages).
	PackagePrefix string

	// TopLevelStmts are all packages' top-level statements to inject at the start of main.
	// Each carries its own prefix. Only used by the root package generator.
	TopLevelStmts []PrefixedStmt

	// GlobalVars is a set of mangled variable names that should use .data section
	// access (OpLoadGlobal/OpStoreGlobal) instead of SSA variables.
	GlobalVars map[string]bool
}

// NewGenerator creates an IR generator with the given configuration.
func NewGenerator(config GeneratorConfig) *Generator {
	g := &Generator{
		prog:          NewProgram(),
		ssa:           NewSSABuilder(),
		typeCache:     make(map[semantic.Type]Type),
		globalVars:    make(map[string]Type),
		packagePrefix: config.PackagePrefix,
		topLevelStmts: config.TopLevelStmts,
	}

	// Register globals for OpLoadGlobal/OpStoreGlobal during generation.
	// Only emit .data labels for globals that belong to this package.
	for name := range config.GlobalVars {
		g.globalVars[name] = TypeS64
		if (config.PackagePrefix == "" && !strings.Contains(name, "__")) ||
			(config.PackagePrefix != "" && strings.HasPrefix(name, config.PackagePrefix)) {
			g.prog.Globals = append(g.prog.Globals, &Global{Name: name, Type: TypeS64})
		}
	}

	return g
}

// pushScope creates a new scope for tracking owned pointers.
func (g *Generator) pushScope() {
	g.ownedVarScopes = append(g.ownedVarScopes, nil)
}

// popScope cleans up owned pointers in the current scope and removes it.
func (g *Generator) popScope() {
	if len(g.ownedVarScopes) == 0 {
		return
	}
	// Emit cleanup for all owned pointers in this scope
	g.emitScopeCleanup()
	// Remove the scope
	g.ownedVarScopes = g.ownedVarScopes[:len(g.ownedVarScopes)-1]
}

// trackOwnedVar registers a variable for cleanup when scope exits. Covers
// every kind of variable that owns heap storage: owned pointers, value-type
// nullables, and struct/class/array values whose literal expressions allocate
// a heap region at construction time.
func (g *Generator) trackOwnedVar(name string, semType semantic.Type) {
	if len(g.ownedVarScopes) == 0 {
		return
	}
	if !varOwnsHeap(semType) {
		return
	}
	lastIdx := len(g.ownedVarScopes) - 1
	g.ownedVarScopes[lastIdx] = append(g.ownedVarScopes[lastIdx], ownedVar{name, semType})
}

// emitScopeCleanup emits free operations for all owned pointers in the current scope.
func (g *Generator) emitScopeCleanup() {
	if len(g.ownedVarScopes) == 0 || g.block == nil {
		return
	}
	lastIdx := len(g.ownedVarScopes) - 1
	for _, ov := range g.ownedVarScopes[lastIdx] {
		// Skip variables whose heap slot was aliased into another owner.
		if g.aliasedHeapVars[ov.name] {
			continue
		}
		g.emitFreeIfOwned(ov.name, ov.semType)
	}
}

// emitLoopExitCleanup frees owned locals from the current innermost scope down
// to and including the loop body scope (loopScopeDepth). break and continue jump
// out of the loop body without reaching the body's fall-through cleanup, so they
// must emit the equivalent frees themselves to keep the heap balanced.
func (g *Generator) emitLoopExitCleanup() {
	if g.block == nil {
		return
	}
	for i := len(g.ownedVarScopes) - 1; i >= g.loopScopeDepth; i-- {
		for _, ov := range g.ownedVarScopes[i] {
			// Skip variables whose heap slot was aliased into another owner.
			if g.aliasedHeapVars[ov.name] {
				continue
			}
			g.emitFreeIfOwned(ov.name, ov.semType)
		}
	}
}

// emitAllScopesCleanup emits cleanup for all scopes (for function return).
// excludeVar names a local whose heap backing is being returned by value, so
// it must not be freed here (ownership transfers to the caller).
func (g *Generator) emitAllScopesCleanup(excludeVar string) {
	if g.block == nil {
		return
	}
	// Free owned pointers from all scopes, innermost first
	for i := len(g.ownedVarScopes) - 1; i >= 0; i-- {
		for _, ov := range g.ownedVarScopes[i] {
			// Skip the returned local and any whose slot was aliased away.
			if ov.name == excludeVar || g.aliasedHeapVars[ov.name] {
				continue
			}
			g.emitFreeIfOwned(ov.name, ov.semType)
		}
	}
}

// markHeapAliased records that a value-type-nullable variable's heap slot has been
// aliased into another binding/array, so it must not be freed at scope exit.
func (g *Generator) markHeapAliased(name string) {
	if g.aliasedHeapVars == nil {
		g.aliasedHeapVars = make(map[string]bool)
	}
	g.aliasedHeapVars[name] = true
}

// Generate converts a TypedProgram to IR.
// PrefixedStmt pairs a top-level statement with its package prefix for IR generation.
type PrefixedStmt struct {
	Stmt   semantic.TypedStatement
	Prefix string // e.g., "config__" for non-root, "" for root
}

// Generate converts a TypedProgram to IR using default configuration.
// This is the simple entry point for single-file programs.
func Generate(typed *semantic.TypedProgram) (*Program, error) {
	g := NewGenerator(GeneratorConfig{})
	return g.GenerateProgram(typed)
}

// GenerateProgram generates IR for an entire program.
func (g *Generator) GenerateProgram(typed *semantic.TypedProgram) (*Program, error) {
	// If no prefixed stmts were set externally (single-file via Generate()),
	// wrap the program's own statements with the current prefix
	if len(g.topLevelStmts) == 0 && len(typed.Statements) > 0 {
		for _, stmt := range typed.Statements {
			g.topLevelStmts = append(g.topLevelStmts, PrefixedStmt{Stmt: stmt, Prefix: g.packagePrefix})
		}
	}

	// First pass: register all struct types
	for _, decl := range typed.Declarations {
		if sd, ok := decl.(*semantic.TypedStructDecl); ok {
			g.registerStruct(sd)
		}
	}

	// Second pass: generate all functions
	for _, decl := range typed.Declarations {
		switch d := decl.(type) {
		case *semantic.TypedFunctionDecl:
			if err := g.generateFunction(d); err != nil {
				return nil, err
			}
		case *semantic.TypedClassDecl:
			if err := g.generateClass(d); err != nil {
				return nil, err
			}
		case *semantic.TypedObjectDecl:
			if err := g.generateObject(d); err != nil {
				return nil, err
			}
		}
	}

	return g.prog, nil
}

// registerStruct adds a struct type to the program.
func (g *Generator) registerStruct(sd *semantic.TypedStructDecl) {
	st := g.convertStructType(&sd.StructType)
	g.prog.AddStruct(st)
}

// generateFunction generates IR for a function declaration.
func (g *Generator) generateFunction(fd *semantic.TypedFunctionDecl) error {
	// Convert return type
	retType := g.convertSSAType(fd.ReturnType)

	// Create function
	g.fn = g.prog.NewFunction(fd.Name, retType)

	// Reset SSA state for this function
	g.ssa.Reset()
	g.ssa.SetFunction(g.fn)

	// Reset owned pointer scope tracking for this function
	g.ownedVarScopes = nil
	g.aliasedHeapVars = make(map[string]bool)
	g.pushScope()

	// Create entry block
	g.block = g.fn.NewBlock(BlockPlain)

	// Generate parameters
	semParams := make([]semantic.Type, 0, len(fd.Parameters))
	for _, param := range fd.Parameters {
		paramType := g.convertSSAType(param.Type)
		paramVal := g.fn.NewParam(paramType)

		// Record parameter as initial definition of the variable
		g.writeVariable(param.Name, g.block, paramVal)

		// An owned-pointer parameter would be owned by the callee and freed at
		// its scope exit. The scope-frees-it model forbids *T params (semantic
		// rejects them), so this branch is defensive only. Value-type nullable
		// parameters (s64?, bool?, ...) are borrowed: the caller retains
		// ownership of the heap slot, so the callee must not free it.
		if elemType, _ := g.getOwnedPointerInfo(param.Type); elemType != nil {
			g.trackOwnedVar(param.Name, param.Type)
		}
		semParams = append(semParams, param.Type)
	}
	if g.funcSemanticParams == nil {
		g.funcSemanticParams = make(map[string][]semantic.Type)
	}
	g.funcSemanticParams[g.fn.Name] = semParams

	// For main, inject top-level statements before the body
	if fd.Name == "main" && len(g.topLevelStmts) > 0 {
		for _, ps := range g.topLevelStmts {
			savedPrefix := g.packagePrefix
			g.packagePrefix = ps.Prefix
			if err := g.generateStatement(ps.Stmt); err != nil {
				return err
			}
			g.packagePrefix = savedPrefix
		}
	}

	// Generate function body
	if fd.Body != nil {
		if err := g.generateBlock(fd.Body); err != nil {
			return err
		}
	}

	// Seal all blocks (we're done adding predecessors)
	for _, b := range g.fn.Blocks {
		g.sealBlock(b)
	}

	// If the function is void and doesn't end with a return, add one
	if g.fn.IsVoid() && g.block != nil && g.block.Kind == BlockPlain {
		// Clean up all owned pointers before implicit return
		g.emitAllScopesCleanup("")
		g.addReturn(nil)
	}

	return nil
}

// generateClass generates IR for a class declaration (methods become functions).
func (g *Generator) generateClass(cd *semantic.TypedClassDecl) error {
	// Register the class type as a struct
	st := g.convertClassType(&cd.ClassType)
	g.prog.AddStruct(st)

	// Generate each method as a function with mangled name
	for _, method := range cd.Methods {
		if err := g.generateMethod(cd.Name, method); err != nil {
			return err
		}
	}

	return nil
}

// generateObject generates IR for an object declaration (methods become functions).
func (g *Generator) generateObject(od *semantic.TypedObjectDecl) error {
	// Generate each method as a function with mangled name
	for _, method := range od.Methods {
		if err := g.generateMethod(od.Name, method); err != nil {
			return err
		}
	}

	return nil
}

// mangleParamSuffix builds a label-safe suffix encoding parameter types, so
// methods overloaded by type (same arity, different parameter types) get
// distinct mangled names. The definition and call sites must both derive it
// from the method's declared parameter types so the labels agree.
func mangleParamSuffix(types []semantic.Type) string {
	var b strings.Builder
	for _, t := range types {
		b.WriteByte('_')
		for _, r := range t.String() {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
				b.WriteRune(r)
			case r == '&':
				b.WriteByte('R') // reference
			case r == '*':
				b.WriteByte('P') // owned pointer
			case r == '?':
				b.WriteByte('Q') // nullable
			case r == '.':
				b.WriteByte('D') // package separator in a type's String()
			case r == '[':
				b.WriteByte('L')
			case r == ']':
				b.WriteByte('J')
				// Other characters (spaces, commas, <, >) are dropped; the same
				// dropping happens on both sides so the labels still agree.
			}
		}
	}
	return b.String()
}

// generateMethod generates IR for a method declaration.
func (g *Generator) generateMethod(className string, md *semantic.TypedMethodDecl) error {
	// Mangle name: ClassName_methodName_paramCount_<paramTypes> (the type
	// suffix distinguishes overloads that share an arity).
	paramTypes := make([]semantic.Type, len(md.Parameters))
	for i, p := range md.Parameters {
		paramTypes[i] = p.Type
	}
	mangledName := fmt.Sprintf("%s_%s_%d%s", className, md.Name, len(md.Parameters), mangleParamSuffix(paramTypes))

	// Convert return type
	retType := g.convertSSAType(md.ReturnType)

	// Create function
	g.fn = g.prog.NewFunction(mangledName, retType)

	// Reset SSA state
	g.ssa.Reset()
	g.ssa.SetFunction(g.fn)

	// Create entry block
	g.block = g.fn.NewBlock(BlockPlain)

	// Generate parameters (including self for instance methods)
	for _, param := range md.Parameters {
		paramType := g.convertSSAType(param.Type)
		paramVal := g.fn.NewParam(paramType)
		g.writeVariable(param.Name, g.block, paramVal)
	}

	// Generate method body
	if md.Body != nil {
		if err := g.generateBlock(md.Body); err != nil {
			return err
		}
	}

	// Seal all blocks
	for _, b := range g.fn.Blocks {
		g.sealBlock(b)
	}

	// Add void return if needed
	if g.fn.IsVoid() && g.block != nil && g.block.Kind == BlockPlain {
		g.addReturn(nil)
	}

	return nil
}

// generateBlock generates IR for a block statement.
func (g *Generator) generateBlock(bs *semantic.TypedBlockStmt) error {
	for _, stmt := range bs.Statements {
		if err := g.generateStatement(stmt); err != nil {
			return err
		}
		// Stop if we've terminated the block
		if g.block == nil || g.block.Kind != BlockPlain {
			break
		}
	}
	return nil
}

// generateScopedBlock generates a block in its own owned-pointer scope, so heap
// allocations created inside the block are freed at the block's end rather than
// living until the enclosing function returns. This is the same scoping the
// loop bodies use, applied to if/else branches and bare { } blocks.
func (g *Generator) generateScopedBlock(bs *semantic.TypedBlockStmt) error {
	g.pushScope()
	err := g.generateBlock(bs)
	// Free this scope's allocations at the block's end, but only if control
	// still reaches here (a return/break already emitted its own cleanup).
	if err == nil && g.block != nil && g.block.Kind == BlockPlain {
		g.emitScopeCleanup()
	}
	g.ownedVarScopes = g.ownedVarScopes[:len(g.ownedVarScopes)-1]
	return err
}

// generateStatement generates IR for a statement.
func (g *Generator) generateStatement(stmt semantic.TypedStatement) error {
	switch s := stmt.(type) {
	case *semantic.TypedExprStmt:
		v, err := g.generateExpr(s.Expr)
		if err != nil {
			return err
		}
		// A bare string temporary (e.g. an interpolated string used as a
		// statement) is discarded; free it so its heap buffer isn't leaked.
		if isOwnedStringTemp(s.Expr) {
			g.builder().StrFree(v)
		}
		return nil

	case *semantic.TypedVarDeclStmt:
		return g.generateVarDecl(s)

	case *semantic.TypedAssignStmt:
		return g.generateAssign(s)

	case *semantic.TypedFieldAssignStmt:
		return g.generateFieldAssign(s)

	case *semantic.TypedIndexAssignStmt:
		return g.generateIndexAssign(s)

	case *semantic.TypedReturnStmt:
		return g.generateReturn(s)

	case *semantic.TypedIfStmt:
		// If it has a ResultType, treat as expression (for nested if expressions)
		if s.ResultType != nil {
			_, err := g.generateIfExpr(s)
			return err
		}
		return g.generateIf(s)

	case *semantic.TypedWhileStmt:
		return g.generateWhile(s)

	case *semantic.TypedForStmt:
		return g.generateFor(s)

	case *semantic.TypedBreakStmt:
		return g.generateBreak(s)

	case *semantic.TypedContinueStmt:
		return g.generateContinue(s)

	case *semantic.TypedBlockStmt:
		return g.generateScopedBlock(s)

	case *semantic.TypedWhenExpr:
		_, err := g.generateWhen(s)
		return err

	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// generateVarDecl generates IR for a variable declaration.
func (g *Generator) generateVarDecl(vd *semantic.TypedVarDeclStmt) error {
	declType := g.convertSSAType(vd.DeclaredType)

	// Handle null literal specially
	if isNullLiteral(vd.Initializer) {
		g.writeVariable(vd.Name, g.block, g.block.NewValue(OpWrapNull, declType))
		g.trackOwnedVar(vd.Name, vd.DeclaredType)
		return nil
	}

	// Generate initializer and wrap if needed
	val, err := g.generateExpr(vd.Initializer)
	if err != nil {
		return err
	}

	// Value-type nullables read from a container alias the container's slot.
	// Copy so the new binding owns its own heap slot.
	if shouldCopyOnReturn(vd.Initializer) {
		val = g.copyNullableValue(val, vd.Initializer.GetType())
	}

	// A bound string takes value semantics: copy if it borrows storage owned
	// elsewhere so the binding owns an independent buffer.
	val = g.maybeCopyString(val, vd.Initializer)
	val = g.maybeCopyVec(val, vd.Initializer)

	// Aggregates take value semantics too: deep-copy a copyable aggregate
	// read from an existing binding. A non-copyable aggregate passes its
	// allocation through unchanged (aliasing one is rejected by semantic).
	val = g.bindAggregateValue(val, vd.Initializer)

	g.writeVariable(vd.Name, g.block, g.wrapIfNeeded(val, declType))
	g.trackOwnedVar(vd.Name, vd.DeclaredType)
	return nil
}

// shouldCopyOnReturn reports whether a returned expression must be copied
// to give the caller an unaliased heap slot. Identifier reads transfer the
// local's backing allocation to the caller (the local is excluded from
// scope-exit cleanup); container reads (arr[i], obj.field) alias and need a
// true copy.
func shouldCopyOnReturn(expr semantic.TypedExpression) bool {
	if nullableValueInner(expr.GetType()) == nil {
		return false
	}
	switch expr.(type) {
	case *semantic.TypedFieldAccessExpr,
		*semantic.TypedIndexExpr:
		return true
	}
	return false
}

// shouldCopyOnReturnFromCall is the variant used by generateReturn: a
// returned identifier of value-nullable type is also an alias of a caller-
// reachable slot (the parameter), so the function must copy.
func shouldCopyOnReturnFromCall(expr semantic.TypedExpression) bool {
	if shouldCopyOnReturn(expr) {
		return true
	}
	if nullableValueInner(expr.GetType()) == nil {
		return false
	}
	if _, ok := expr.(*semantic.TypedIdentifierExpr); ok {
		return true
	}
	return false
}

// copyNullableValue allocates a fresh heap slot for a value-type nullable
// and copies the source value into it. Null is preserved as null. The
// returned IR value is a pointer that the caller alone owns.
func (g *Generator) copyNullableValue(src *Value, semType semantic.Type) *Value {
	inner := nullableValueInner(semType)
	if inner == nil {
		return src
	}

	// If null, return null directly. Otherwise, allocate a slot, copy the
	// underlying value, and return the new pointer. Implemented via phi.
	notNullBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)
	nullBlock := g.block

	isNull := g.block.NewValue(OpIsNull, TypeBool, src)
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(mergeBlock)    // null -> skip
	g.block.AddSucc(notNullBlock)  // not null -> copy
	g.sealBlock(notNullBlock)

	g.block = notNullBlock
	innerIRType := g.convertType(inner)
	unwrapped := g.block.NewValue(OpUnwrap, innerIRType, src)
	wrapped := g.block.NewValue(OpWrap, &NullableType{Elem: innerIRType}, unwrapped)
	notNullEnd := g.block
	notNullEnd.AddSucc(mergeBlock)

	g.sealBlock(mergeBlock)
	g.block = mergeBlock
	phi := g.block.NewPhiValue(&NullableType{Elem: innerIRType})
	phi.PhiArgs = []*PhiArg{
		{From: nullBlock, Value: src},
		{From: notNullEnd, Value: wrapped},
	}
	return phi
}

// generateAssign generates IR for a variable assignment.
func (g *Generator) generateAssign(as *semantic.TypedAssignStmt) error {
	varType := g.convertSSAType(as.VarType)

	if isNullLiteral(as.Value) {
		g.emitFreeIfOwned(as.Name, as.VarType)
		delete(g.aliasedHeapVars, as.Name)
		g.writeVariable(as.Name, g.block, g.block.NewValue(OpWrapNull, varType))
		return nil
	}

	// If the RHS is an identifier whose heap slot is aliased into as.Name
	// (value-type nullables: copyable, so semantic permits the alias), mark the
	// source moved so its scope exit doesn't double-free the slot the
	// destination now owns. Strings and copyable aggregates use copy semantics
	// (handled below), so the source remains valid and must not be marked.
	if ident, ok := as.Value.(*semantic.TypedIdentifierExpr); ok {
		if ident.Name != as.Name && varOwnsHeap(ident.Type) && !isStringType(ident.Type) &&
			!g.aggregateIsCopyable(ident.Type) {
			g.markHeapAliased(ident.Name)
		}
	}

	// Generate the new value first so any read of the old variable inside the
	// RHS resolves before we free it.
	val, err := g.generateExpr(as.Value)
	if err != nil {
		return err
	}

	// Value-type nullables read from a container alias the container's slot.
	// Copy the value so the new binding owns its own heap slot.
	if shouldCopyOnReturn(as.Value) {
		val = g.copyNullableValue(val, as.Value.GetType())
	}

	// Strings take value semantics: copy a borrowed source so the destination
	// owns an independent buffer.
	val = g.maybeCopyString(val, as.Value)
	val = g.maybeCopyVec(val, as.Value)

	// Aggregates take value semantics too: deep-copy a copyable aggregate
	// read from an existing binding. A non-copyable aggregate passes its
	// allocation through unchanged (aliasing one is rejected by semantic).
	val = g.bindAggregateValue(val, as.Value)

	// Free whatever the variable currently owns before overwriting. The new
	// value gives the variable fresh ownership, so clear any aliased flag too.
	g.emitFreeIfOwned(as.Name, as.VarType)
	delete(g.aliasedHeapVars, as.Name)
	g.writeVariable(as.Name, g.block, g.wrapIfNeeded(val, varType))
	return nil
}

// generateFieldAssign generates IR for a field assignment.
func (g *Generator) generateFieldAssign(fa *semantic.TypedFieldAssignStmt) error {
	obj, err := g.generateExpr(fa.Object)
	if err != nil {
		return err
	}

	fieldSemType := g.types().FieldSemanticType(fa.Object.GetType(), fa.Field)
	fieldIRType := g.convertType(fieldSemType)
	offset := g.getFieldOffset(fa.Object.GetType(), fa.Field)

	var val *Value
	if isNullLiteral(fa.Value) {
		val = g.block.NewValue(OpWrapNull, fieldIRType)
	} else {
		val, err = g.generateExpr(fa.Value)
		if err != nil {
			return err
		}
		// Borrowed string values are copied so the field owns its own buffer.
		val = g.maybeCopyString(val, fa.Value)
		val = g.maybeCopyVec(val, fa.Value)
		val = g.wrapIfNeeded(val, fieldIRType)
	}

	fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: fieldIRType})
	fieldPtr.AddArg(obj)
	fieldPtr.AuxInt = int64(offset)

	// Free whatever the field currently owns before overwriting. Skips when
	// the field type does not own heap storage. String fields own a heap
	// buffer separate from the struct, so free them here too.
	if fieldSemType != nil && (fieldOwnsHeap(fieldSemType) || isStringType(fieldSemType)) {
		oldVal := g.block.NewValue(OpLoad, fieldIRType, fieldPtr)
		g.emitFreeOwnedValue(oldVal, fieldSemType)
	}

	store := g.block.NewValue(OpStore, nil)
	store.AddArg(fieldPtr)
	store.AddArg(val)

	return nil
}

// fieldOwnsHeap reports whether a field's declared semantic type owns heap
// storage that must be released when the field is overwritten. Struct/class/
// array fields are intentionally excluded — those are embedded inline in the
// enclosing allocation, not separately allocated.
func fieldOwnsHeap(t semantic.Type) bool {
	if nullableValueInner(t) != nil {
		return true
	}
	if isNullableOwnedType(t) {
		return true
	}
	switch t.(type) {
	case *semantic.OwnedPointerType, semantic.OwnedPointerType:
		return true
	}
	return false
}

// varOwnsHeap reports whether a variable of this type owns a heap allocation
// that must be released when the variable goes out of scope. This is broader
// than fieldOwnsHeap: struct/class/array literals (`val p = Point{...}`,
// `val a = [1,2,3]`) all allocate heap behind the scenes, and the binding
// owns that allocation.
func varOwnsHeap(t semantic.Type) bool {
	if fieldOwnsHeap(t) {
		return true
	}
	switch t.(type) {
	case *semantic.StructType, semantic.StructType:
		return true
	case *semantic.ClassType, semantic.ClassType:
		return true
	case *semantic.ArrayType, semantic.ArrayType:
		return true
	case semantic.StringType, *semantic.StringType:
		// Strings are heap-owned value types: a binding owns its heap buffer
		// (interpolation results, copies) and frees it at scope exit. Constant
		// strings live in .data and free as a no-op (see _sl_str_free).
		return true
	case semantic.VecType, *semantic.VecType:
		// vec is a heap-owned value type just like string: a binding owns its
		// header+data and frees them at scope exit, with copy-on-store.
		return true
	}
	return false
}

// isStringType reports whether t is the string type.
func isStringType(t semantic.Type) bool {
	switch t.(type) {
	case semantic.StringType, *semantic.StringType:
		return true
	}
	return false
}

// isVecType reports whether t is the vec type.
func isVecType(t semantic.Type) bool {
	switch t.(type) {
	case semantic.VecType, *semantic.VecType:
		return true
	}
	return false
}

// vecIsBorrow reports whether a vec-typed expression refers to storage owned
// elsewhere (a variable, field, or array element), so storing it requires a deep
// copy. A fresh vec (vec() / a call result) is not a borrow.
func vecIsBorrow(expr semantic.TypedExpression) bool {
	if !isVecType(expr.GetType()) {
		return false
	}
	switch expr.(type) {
	case *semantic.TypedIdentifierExpr,
		*semantic.TypedFieldAccessExpr,
		*semantic.TypedIndexExpr:
		return true
	}
	return false
}

// maybeCopyVec returns a deep copy of val when expr is a borrowed vec, otherwise
// val unchanged. Parallels maybeCopyString.
func (g *Generator) maybeCopyVec(val *Value, expr semantic.TypedExpression) *Value {
	if vecIsBorrow(expr) {
		return g.builder().VecCopy(val)
	}
	return val
}

// stringIsBorrow reports whether a string-typed expression refers to storage
// owned elsewhere (a variable, field, or array element). Storing such a value
// requires a deep copy so the destination owns an independent buffer and the
// two owners don't double-free. Fresh strings (interpolation) and ownership-
// transferring results (call/method returns) are NOT borrows.
func stringIsBorrow(expr semantic.TypedExpression) bool {
	if !isStringType(expr.GetType()) {
		return false
	}
	switch expr.(type) {
	case *semantic.TypedIdentifierExpr,
		*semantic.TypedFieldAccessExpr,
		*semantic.TypedIndexExpr:
		return true
	}
	return false
}

// isOwnedStringTemp reports whether a string-typed expression produces a fresh
// heap string that no binding owns, so it must be freed after being consumed as
// a temporary (e.g. a print argument or a bare expression statement). Calls that
// return a constant string are still safe to "free" — _sl_str_free no-ops on
// non-heap pointers.
func isOwnedStringTemp(expr semantic.TypedExpression) bool {
	if !isStringType(expr.GetType()) {
		return false
	}
	switch expr.(type) {
	case *semantic.TypedInterpolatedStringExpr,
		*semantic.TypedCallExpr,
		*semantic.TypedMethodCallExpr:
		return true
	}
	return false
}

// maybeCopyString returns a deep copy of val when expr is a borrowed string,
// otherwise val unchanged. Used wherever a string is stored into a binding,
// field, element, or returned, to preserve value (copy) semantics.
func (g *Generator) maybeCopyString(val *Value, expr semantic.TypedExpression) *Value {
	if stringIsBorrow(expr) {
		return g.builder().StrCopy(val)
	}
	return val
}

// generateIndexAssign generates IR for an array index assignment.
func (g *Generator) generateIndexAssign(ia *semantic.TypedIndexAssignStmt) error {
	arr, err := g.generateExpr(ia.Array)
	if err != nil {
		return err
	}

	idx, err := g.generateExpr(ia.Index)
	if err != nil {
		return err
	}

	elemSemType := arrayElementSemType(ia.Array.GetType())
	var elemIRType Type
	if elemSemType != nil {
		elemIRType = g.convertType(elemSemType)
	}

	var val *Value
	if isNullLiteral(ia.Value) {
		val = g.block.NewValue(OpWrapNull, elemIRType)
	} else {
		val, err = g.generateExpr(ia.Value)
		if err != nil {
			return err
		}
		// Borrowed string values are copied so the element owns its own buffer.
		val = g.maybeCopyString(val, ia.Value)
		val = g.maybeCopyVec(val, ia.Value)
		if elemIRType != nil {
			val = g.wrapIfNeeded(val, elemIRType)
		}
	}

	if elemIRType == nil {
		elemIRType = val.Type
	}

	elemPtr := g.block.NewValue(OpIndexPtr, &PtrType{Elem: elemIRType})
	elemPtr.AddArg(arr)
	elemPtr.AddArg(idx)

	// Free whatever the element currently owns before overwriting. String
	// elements own a heap buffer, so free them here too.
	if elemSemType != nil && (fieldOwnsHeap(elemSemType) || isStringType(elemSemType)) {
		oldVal := g.block.NewValue(OpLoad, elemIRType, elemPtr)
		g.emitFreeOwnedValue(oldVal, elemSemType)
	}

	store := g.block.NewValue(OpStore, nil)
	store.AddArg(elemPtr)
	store.AddArg(val)

	return nil
}

// arrayElementSemType returns the element type of an array, or nil if the
// type is not an array.
func arrayElementSemType(t semantic.Type) semantic.Type {
	switch ty := t.(type) {
	case *semantic.ArrayType:
		return ty.ElementType
	case semantic.ArrayType:
		return ty.ElementType
	}
	return nil
}

// generateReturn generates IR for a return statement.
func (g *Generator) generateReturn(rs *semantic.TypedReturnStmt) error {
	var retVal *Value
	var excludeVar string // Local whose heap backing is returned by value (skip its free)

	if rs.Value != nil {
		retType := g.fn.ReturnType

		// Handle null literal specially
		if isNullLiteral(rs.Value) {
			retVal = g.block.NewValue(OpWrapNull, retType)
		} else {
			// Returning a struct/class/array local by value transfers its
			// backing allocation to the caller, so it must be excluded from
			// scope-exit cleanup. Strings are copied (below) and value-type
			// nullables get a fresh slot, so both still need their free.
			// (Owned pointers *T can never be returned — semantic rejects it.)
			if ident, ok := rs.Value.(*semantic.TypedIdentifierExpr); ok {
				if varOwnsHeap(ident.Type) && !isStringType(ident.Type) &&
					!isVecType(ident.Type) && nullableValueInner(ident.Type) == nil {
					excludeVar = ident.Name
				}
			}

			var err error
			retVal, err = g.generateExpr(rs.Value)
			if err != nil {
				return err
			}

			// A returned borrowed string must be copied so the caller receives
			// an owned buffer and the local/parameter isn't double-freed by
			// scope cleanup. Done before wrapping so the copy is the raw string.
			retVal = g.maybeCopyString(retVal, rs.Value)
			retVal = g.maybeCopyVec(retVal, rs.Value)

			retVal = g.wrapIfNeeded(retVal, retType)

			// Value-type nullables have copy semantics: aliasing the caller's
			// slot would cause a double-free. Allocate a fresh slot and copy
			// the contents (preserving the null state). This converts an
			// alias-shaped pointer into an owned-shaped pointer.
			if shouldCopyOnReturnFromCall(rs.Value) {
				retVal = g.copyNullableValue(retVal, rs.Value.GetType())
			}
		}
	}

	// Clean up all owned pointers before returning (except the one being returned)
	g.emitAllScopesCleanup(excludeVar)

	g.addReturn(retVal)
	return nil
}

// addReturn adds a return terminator to the current block.
func (g *Generator) addReturn(val *Value) {
	ret := g.block.NewValue(OpReturn, nil)
	if val != nil {
		ret.AddArg(val)
	}
	g.block.Kind = BlockReturn
	g.block = nil // Block is terminated
}

// generateIf generates IR for an if statement.
func (g *Generator) generateIf(is *semantic.TypedIfStmt) error {
	// Generate condition
	cond, err := g.generateExpr(is.Condition)
	if err != nil {
		return err
	}

	// Create blocks
	thenBlock := g.fn.NewBlock(BlockPlain)
	var elseBlock *Block
	if is.ElseBranch != nil {
		elseBlock = g.fn.NewBlock(BlockPlain)
	}
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Set up conditional branch
	g.block.Kind = BlockIf
	g.block.Control = cond
	g.block.AddSucc(thenBlock)
	if elseBlock != nil {
		g.block.AddSucc(elseBlock)
	} else {
		g.block.AddSucc(mergeBlock)
	}

	// Seal thenBlock - its only predecessor is the current (condition) block
	g.sealBlock(thenBlock)

	// Generate then branch in its own scope so block-local allocations are
	// freed at the branch's end.
	g.block = thenBlock
	if err := g.generateScopedBlock(is.ThenBranch); err != nil {
		return err
	}
	// Jump to merge if not terminated
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(mergeBlock)
	}

	// Generate else branch if present
	if is.ElseBranch != nil {
		// Seal elseBlock - its only predecessor is the condition block
		g.sealBlock(elseBlock)

		g.block = elseBlock
		if err := g.generateStatement(is.ElseBranch); err != nil {
			return err
		}
		// Jump to merge if not terminated
		if g.block != nil && g.block.Kind == BlockPlain {
			g.block.AddSucc(mergeBlock)
		}
	}

	// Seal mergeBlock - all predecessors are now known
	g.sealBlock(mergeBlock)

	// Continue in merge block
	g.block = mergeBlock

	return nil
}

// generateWhile generates IR for a while loop.
func (g *Generator) generateWhile(ws *semantic.TypedWhileStmt) error {
	// Create blocks
	headerBlock := g.fn.NewBlock(BlockPlain)
	bodyBlock := g.fn.NewBlock(BlockPlain)
	exitBlock := g.fn.NewBlock(BlockPlain)

	// Save loop context for break/continue
	prevBreakTarget := g.breakTarget
	prevContinueTarget := g.continueTarget
	prevLoopScopeDepth := g.loopScopeDepth
	g.breakTarget = exitBlock
	g.continueTarget = headerBlock

	// Jump to header
	g.block.AddSucc(headerBlock)
	g.block = headerBlock

	// Generate condition
	cond, err := g.generateExpr(ws.Condition)
	if err != nil {
		return err
	}

	// Conditional branch
	g.block.Kind = BlockIf
	g.block.Control = cond
	g.block.AddSucc(bodyBlock)
	g.block.AddSucc(exitBlock)

	// Seal bodyBlock now - its only predecessor is headerBlock
	g.sealBlock(bodyBlock)

	// Generate body within its own scope so per-iteration owned variables
	// are freed before the next iteration.
	g.block = bodyBlock
	g.pushScope()
	g.loopScopeDepth = len(g.ownedVarScopes) - 1
	if err := g.generateBlock(ws.Body); err != nil {
		return err
	}
	if g.block != nil && g.block.Kind == BlockPlain {
		g.emitScopeCleanup()
	}
	g.ownedVarScopes = g.ownedVarScopes[:len(g.ownedVarScopes)-1]
	// Jump back to header if not terminated
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(headerBlock)
	}

	// Seal headerBlock now - all predecessors are known (entry + body end)
	g.sealBlock(headerBlock)

	// Seal exitBlock - its only predecessor is headerBlock
	g.sealBlock(exitBlock)

	// Restore loop context
	g.breakTarget = prevBreakTarget
	g.continueTarget = prevContinueTarget
	g.loopScopeDepth = prevLoopScopeDepth

	// Continue in exit block
	g.block = exitBlock

	return nil
}


// generateFor generates IR for a for loop.
func (g *Generator) generateFor(fs *semantic.TypedForStmt) error {
	// Generate init (if present)
	if fs.Init != nil {
		if err := g.generateStatement(fs.Init); err != nil {
			return err
		}
	}

	// Create blocks
	headerBlock := g.fn.NewBlock(BlockPlain)
	bodyBlock := g.fn.NewBlock(BlockPlain)
	updateBlock := g.fn.NewBlock(BlockPlain)
	exitBlock := g.fn.NewBlock(BlockPlain)

	// Save loop context for break/continue
	prevBreakTarget := g.breakTarget
	prevContinueTarget := g.continueTarget
	prevLoopScopeDepth := g.loopScopeDepth
	g.breakTarget = exitBlock
	g.continueTarget = updateBlock

	// Jump to header
	g.block.AddSucc(headerBlock)
	g.block = headerBlock

	// Generate condition (if present)
	if fs.Condition != nil {
		cond, err := g.generateExpr(fs.Condition)
		if err != nil {
			return err
		}

		// Conditional branch
		g.block.Kind = BlockIf
		g.block.Control = cond
		g.block.AddSucc(bodyBlock)
		g.block.AddSucc(exitBlock)
	} else {
		// No condition means always true
		g.block.AddSucc(bodyBlock)
	}

	// Seal bodyBlock now - its only predecessor is headerBlock
	g.sealBlock(bodyBlock)

	// Generate body within its own scope so per-iteration owned variables
	// (val p = new T{...}) are freed before the next iteration.
	g.block = bodyBlock
	g.pushScope()
	g.loopScopeDepth = len(g.ownedVarScopes) - 1
	if err := g.generateBlock(fs.Body); err != nil {
		return err
	}
	if g.block != nil && g.block.Kind == BlockPlain {
		g.emitScopeCleanup()
	}
	g.ownedVarScopes = g.ownedVarScopes[:len(g.ownedVarScopes)-1]
	// Jump to update if not terminated
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(updateBlock)
	}

	// Seal updateBlock now - after body is generated (continue statements may add predecessors)
	g.sealBlock(updateBlock)

	// Generate update (in update block)
	g.block = updateBlock
	if fs.Update != nil {
		if err := g.generateStatement(fs.Update); err != nil {
			return err
		}
	}
	// Jump back to header
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(headerBlock)
	}

	// Seal headerBlock now - all its predecessors are known (entry + updateBlock)
	g.sealBlock(headerBlock)

	// Seal exitBlock - after body is generated (break statements may add predecessors)
	g.sealBlock(exitBlock)

	// Restore loop context
	g.breakTarget = prevBreakTarget
	g.continueTarget = prevContinueTarget
	g.loopScopeDepth = prevLoopScopeDepth

	// Continue in exit block
	g.block = exitBlock

	return nil
}

// generateBreak generates IR for a break statement.
func (g *Generator) generateBreak(_ *semantic.TypedBreakStmt) error {
	if g.breakTarget == nil {
		return fmt.Errorf("break outside of loop")
	}
	// Free owned locals allocated in the loop body before leaving it.
	g.emitLoopExitCleanup()
	g.block.AddSucc(g.breakTarget)
	g.block = nil // Block is terminated
	return nil
}

// generateContinue generates IR for a continue statement.
func (g *Generator) generateContinue(_ *semantic.TypedContinueStmt) error {
	if g.continueTarget == nil {
		return fmt.Errorf("continue outside of loop")
	}
	// Free owned locals allocated in the loop body before restarting the loop.
	g.emitLoopExitCleanup()
	g.block.AddSucc(g.continueTarget)
	g.block = nil // Block is terminated
	return nil
}

// generateWhen generates IR for a when expression.
func (g *Generator) generateWhen(we *semantic.TypedWhenExpr) (*Value, error) {
	// Statement-position when has no result type; only build a result phi
	// for expression-position when. convertSSAType maps nil to TypeVoid, so
	// check the semantic type before converting.
	var resultType Type
	if we.ResultType != nil {
		if _, isVoid := we.ResultType.(semantic.VoidType); !isVoid {
			resultType = g.convertSSAType(we.ResultType)
		}
	}
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Track phi arguments if this is an expression (has result type)
	var phiArgs []*PhiArg

	for i, c := range we.Cases {
		if c.IsElse {
			// Else case: generate body and jump to merge
			val, err := g.generateWhenCaseBody(c.Body, resultType)
			if err != nil {
				return nil, err
			}

			// If expression, collect phi arg
			if resultType != nil && g.block != nil && val != nil {
				phiArgs = append(phiArgs, &PhiArg{From: g.block, Value: val})
			}

			// Jump to merge
			if g.block != nil && g.block.Kind == BlockPlain {
				g.block.AddSucc(mergeBlock)
			}
		} else {
			// Regular case: condition -> then/next
			cond, err := g.generateExpr(c.Condition)
			if err != nil {
				return nil, err
			}

			thenBlock := g.fn.NewBlock(BlockPlain)
			var nextBlock *Block
			if i < len(we.Cases)-1 {
				nextBlock = g.fn.NewBlock(BlockPlain)
			} else {
				nextBlock = mergeBlock
			}

			// Conditional branch
			g.block.Kind = BlockIf
			g.block.Control = cond
			g.block.AddSucc(thenBlock)
			g.block.AddSucc(nextBlock)

			// Seal then block (its only predecessor is the current conditional block)
			g.sealBlock(thenBlock)

			// Seal next block (its only predecessor is the current conditional
			// block) — unless it is the merge block, which gains more
			// predecessors from case bodies and is sealed after the loop.
			if nextBlock != mergeBlock {
				g.sealBlock(nextBlock)
			}

			// Generate then block
			g.block = thenBlock
			val, err := g.generateWhenCaseBody(c.Body, resultType)
			if err != nil {
				return nil, err
			}

			// If expression, collect phi arg
			if resultType != nil && g.block != nil && val != nil {
				phiArgs = append(phiArgs, &PhiArg{From: g.block, Value: val})
			}

			// Jump to merge
			if g.block != nil && g.block.Kind == BlockPlain {
				g.block.AddSucc(mergeBlock)
			}

			// Move to next condition block
			g.block = nextBlock
		}
	}

	// Seal merge block (all predecessors should now be connected)
	g.sealBlock(mergeBlock)

	// Continue in merge block
	g.block = mergeBlock

	// Create phi node if this is an expression
	if resultType != nil && len(phiArgs) > 0 {
		phi := g.block.NewPhiValue(resultType)
		phi.PhiArgs = phiArgs
		return phi, nil
	}

	return nil, nil
}

// getLastValue returns the last value added to the current block.
func (g *Generator) getLastValue() *Value {
	if g.block == nil || len(g.block.Values) == 0 {
		return nil
	}
	return g.block.Values[len(g.block.Values)-1]
}

// generateWhenCaseBody generates a when case body and returns the result value
// for expression-position when. For bare identifiers/expressions, this directly
// evaluates the expression rather than relying on getLastValue(), which fails
// when the expression doesn't emit a new value into the current block.
func (g *Generator) generateWhenCaseBody(body semantic.TypedStatement, resultType Type) (*Value, error) {
	// If this is an expression-position when, try to get the value directly
	if resultType != nil {
		if exprStmt, ok := body.(*semantic.TypedExprStmt); ok {
			return g.generateExpr(exprStmt.Expr)
		}
	}
	// For block bodies or statement-position when, generate normally
	if err := g.generateStatement(body); err != nil {
		return nil, err
	}
	if resultType != nil {
		return g.getLastValue(), nil
	}
	return nil, nil
}

// generateExpr generates IR for an expression.
func (g *Generator) generateExpr(expr semantic.TypedExpression) (*Value, error) {
	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		return g.generateLiteral(e)

	case *semantic.TypedInterpolatedStringExpr:
		return g.generateInterpolatedString(e)

	case *semantic.TypedIdentifierExpr:
		return g.generateIdentifier(e)

	case *semantic.TypedBinaryExpr:
		return g.generateBinary(e)

	case *semantic.TypedUnaryExpr:
		return g.generateUnary(e)

	case *semantic.TypedCallExpr:
		return g.generateCall(e)

	case *semantic.TypedFieldAccessExpr:
		return g.generateFieldAccess(e)

	case *semantic.TypedIndexExpr:
		return g.generateIndex(e)

	case *semantic.TypedLenExpr:
		return g.generateLen(e)

	case *semantic.TypedArrayLiteralExpr:
		return g.generateArrayLiteral(e)

	case *semantic.TypedStructLiteralExpr:
		return g.generateStructLiteral(e)

	case *semantic.TypedClassLiteralExpr:
		return g.generateClassLiteral(e)

	case *semantic.TypedNewExpr:
		return g.generateNewExpr(e)

	case *semantic.TypedMethodCallExpr:
		return g.generateMethodCall(e)

	case *semantic.TypedSafeCallExpr:
		return g.generateSafeCall(e)

	case *semantic.TypedSelfExpr:
		return g.generateSelf(e)

	case *semantic.TypedIfStmt:
		return g.generateIfExpr(e)

	case *semantic.TypedWhenExpr:
		return g.generateWhen(e)

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// generateLiteral generates IR for a literal expression.
func (g *Generator) generateLiteral(le *semantic.TypedLiteralExpr) (*Value, error) {
	irType := g.convertType(le.Type)

	switch le.LitType {
	case ast.LiteralTypeInteger:
		var intVal int64

		// Check if the type is unsigned to use the correct parsing
		if intType, ok := irType.(*IntType); ok && !intType.Signed {
			// Parse as unsigned, then store bits as int64
			uval, err := strconv.ParseUint(le.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid unsigned integer literal: %s", le.Value)
			}
			intVal = int64(uval) // Store bits, interpretation depends on type
		} else {
			// Try parsing as signed first
			val, err := strconv.ParseInt(le.Value, 10, 64)
			if err != nil {
				// If signed parsing fails, try unsigned (for u64 max value etc.)
				uval, uerr := strconv.ParseUint(le.Value, 10, 64)
				if uerr != nil {
					return nil, fmt.Errorf("invalid integer literal: %s", le.Value)
				}
				intVal = int64(uval)
			} else {
				intVal = val
			}
		}

		v := g.block.NewValue(OpConst, irType)
		v.AuxInt = intVal
		return v, nil

	case ast.LiteralTypeFloat:
		var floatVal float64
		fmt.Sscanf(le.Value, "%f", &floatVal)

		v := g.block.NewValue(OpConst, irType)
		v.AuxFloat = floatVal
		return v, nil

	case ast.LiteralTypeString:
		// Add string to constant pool
		idx := g.prog.AddString(le.Value)

		v := g.block.NewValue(OpConst, irType)
		v.AuxInt = int64(idx)
		v.AuxString = le.Value
		return v, nil

	case ast.LiteralTypeBoolean:
		v := g.block.NewValue(OpConst, irType)
		if le.Value == "true" {
			v.AuxInt = 1
		} else {
			v.AuxInt = 0
		}
		return v, nil

	case ast.LiteralTypeNull:
		v := g.block.NewValue(OpWrapNull, irType)
		return v, nil

	default:
		return nil, fmt.Errorf("unknown literal type: %d", le.LitType)
	}
}

// generateInterpolatedString generates IR for an interpolated string. Each part
// is converted to a string value and the parts are concatenated left-to-right
// with OpStrConcat. Intermediate results (per-part conversions and intermediate
// concatenations) that this code owns are freed as soon as their bytes have
// been copied, so the only surviving allocation is the final result. The result
// is always a fresh heap string the caller owns (and must free).
func (g *Generator) generateInterpolatedString(e *semantic.TypedInterpolatedStringExpr) (*Value, error) {
	var result *Value
	resultOwned := false // whether `result` is a heap temp this code owns

	for _, part := range e.Parts {
		// Skip empty literal chunks — they contribute nothing.
		if lit, ok := part.(*semantic.TypedLiteralExpr); ok && lit.LitType == ast.LiteralTypeString && lit.Value == "" {
			continue
		}
		partVal, partOwned, err := g.generateStringPart(part)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result, resultOwned = partVal, partOwned
			continue
		}
		concat := g.builder().StrConcat(result, partVal)
		// Both inputs have been copied into `concat`; free the ones we own.
		if resultOwned {
			g.builder().StrFree(result)
		}
		if partOwned {
			g.builder().StrFree(partVal)
		}
		result, resultOwned = concat, true
	}

	if result == nil {
		// Pathological all-empty case — return a fresh empty string.
		return g.builder().StrCopy(g.builder().ConstString(g.prog, "")), nil
	}
	// Guarantee the caller receives an owned heap string: a lone constant chunk
	// or borrowed string must be copied so the caller can free it safely.
	if !resultOwned {
		result = g.builder().StrCopy(result)
	}
	return result, nil
}

// generateStringPart converts a single interpolation part to a string value,
// reporting whether the returned value is a heap temporary owned by the caller
// (true) or a borrowed/constant/static pointer that must not be freed (false).
func (g *Generator) generateStringPart(part semantic.TypedExpression) (*Value, bool, error) {
	switch part.GetType().(type) {
	case semantic.StringType:
		v, err := g.generateExpr(part)
		if err != nil {
			return nil, false, err
		}
		// Fresh strings (interpolation, call results) are owned; identifiers,
		// fields, and constants are borrowed/static.
		return v, isOwnedStringTemp(part), nil
	case semantic.S64Type:
		v, err := g.generateExpr(part)
		if err != nil {
			return nil, false, err
		}
		return g.builder().IntToStr(v), true, nil
	case semantic.BooleanType:
		v, err := g.generateExpr(part)
		if err != nil {
			return nil, false, err
		}
		// Returns a pointer to a static "true"/"false" constant — not owned.
		return g.builder().BoolToStr(v), false, nil
	case semantic.NullableType:
		v, err := g.generateNullableToStr(part)
		return v, true, err
	default:
		return nil, false, fmt.Errorf("cannot interpolate value of type %s", part.GetType().String())
	}
}

// convertScalarToStr converts an already-unwrapped (non-nullable) value of the
// given element type to a freshly allocated, caller-owned heap string. Borrowed
// strings and static bool strings are copied so the result is uniformly owned.
func (g *Generator) convertScalarToStr(val *Value, elem semantic.Type) (*Value, error) {
	switch elem.(type) {
	case semantic.StringType:
		return g.builder().StrCopy(val), nil
	case semantic.S64Type:
		return g.builder().IntToStr(val), nil
	case semantic.BooleanType:
		return g.builder().StrCopy(g.builder().BoolToStr(val)), nil
	default:
		return nil, fmt.Errorf("cannot interpolate value of type %s", elem.String())
	}
}

// generateNullableToStr renders a nullable interpolation part: it emits a branch
// that yields the literal "null" when the value is null, otherwise unwraps and
// converts the inner value. Modeled on generateElvis.
func (g *Generator) generateNullableToStr(part semantic.TypedExpression) (*Value, error) {
	nt := part.GetType().(semantic.NullableType)

	val, err := g.generateExpr(part)
	if err != nil {
		return nil, err
	}

	nullBlock := g.fn.NewBlock(BlockPlain)
	valBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(val)
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(nullBlock)
	g.block.AddSucc(valBlock)
	g.sealBlock(nullBlock)
	g.sealBlock(valBlock)

	// null edge -> owned heap copy of "null" (so both phi inputs are owned heap)
	g.block = nullBlock
	nullStr := g.builder().StrCopy(g.builder().ConstString(g.prog, "null"))
	nullEnd := g.block
	nullEnd.AddSucc(mergeBlock)

	// not-null edge -> unwrap and convert
	g.block = valBlock
	innerIR := g.convertType(nt.InnerType)
	unwrapped := g.block.NewValue(OpUnwrap, innerIR)
	unwrapped.AddArg(val)
	// Free a temporary heap slot if the nullable came from an owning temporary
	// (e.g. a function returning T?), mirroring generateElvis.
	if isOwningTemp(part) {
		freeOwningTemp(g, val, part.GetType())
	}
	converted, err := g.convertScalarToStr(unwrapped, nt.InnerType)
	if err != nil {
		return nil, err
	}
	valEnd := g.block
	valEnd.AddSucc(mergeBlock)

	g.sealBlock(mergeBlock)
	g.block = mergeBlock
	phi := g.block.NewPhiValue(TypeString)
	phi.PhiArgs = []*PhiArg{
		{From: nullEnd, Value: nullStr},
		{From: valEnd, Value: converted},
	}
	return phi, nil
}

// generateIdentifier generates IR for an identifier expression.
func (g *Generator) generateIdentifier(ie *semantic.TypedIdentifierExpr) (*Value, error) {
	// readVariable handles prefixing/mangling automatically
	return g.readVariable(ie.Name, g.block), nil
}

// generateBinary generates IR for a binary expression.
// Dispatches to specialized handlers for special operators.
func (g *Generator) generateBinary(be *semantic.TypedBinaryExpr) (*Value, error) {
	switch be.Op {
	case "&&", "||":
		return g.generateShortCircuit(be)
	case "?:":
		return g.generateElvis(be)
	case "==", "!=":
		if result, handled, err := g.tryGenerateNullComparison(be); handled {
			return result, err
		}
		// String comparison uses OpStrEq instead of numeric OpEq
		if _, isStr := be.Left.GetType().(semantic.StringType); isStr {
			return g.generateStringComparison(be)
		}
	}

	return g.generateBinaryOp(be)
}

// generateStringComparison generates IR for string == and != comparisons.
func (g *Generator) generateStringComparison(be *semantic.TypedBinaryExpr) (*Value, error) {
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}
	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}

	v := g.block.NewValue(OpStrEq, TypeBool)
	v.AddArg(left)
	v.AddArg(right)

	if be.Op == "!=" {
		// Negate the result
		notV := g.block.NewValue(OpNot, TypeBool)
		notV.AddArg(v)
		return notV, nil
	}
	return v, nil
}

// tryGenerateNullComparison handles `x == null` and `x != null` comparisons.
// Returns (result, true, nil) if one operand is a null literal.
// Returns (nil, false, nil) if neither operand is null (caller should use regular comparison).
func (g *Generator) tryGenerateNullComparison(be *semantic.TypedBinaryExpr) (*Value, bool, error) {
	leftIsNull := isNullLiteral(be.Left)
	rightIsNull := isNullLiteral(be.Right)

	if !leftIsNull && !rightIsNull {
		return nil, false, nil
	}

	// Determine which operand to test for null
	testExpr := be.Left
	if leftIsNull {
		testExpr = be.Right
	}

	val, err := g.generateExpr(testExpr)
	if err != nil {
		return nil, true, err
	}

	b := g.builder()
	isNullVal := b.IsNull(val)

	if be.Op == "!=" {
		return b.Not(isNullVal), true, nil
	}
	return isNullVal, true, nil
}

// generateBinaryOp generates IR for standard binary operations (+, -, *, /, etc).
func (g *Generator) generateBinaryOp(be *semantic.TypedBinaryExpr) (*Value, error) {
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}

	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}

	op, ok := binaryOpMap[be.Op]
	if !ok {
		return nil, fmt.Errorf("unknown binary operator: %s", be.Op)
	}

	resultType := g.convertType(be.Type)
	v := g.block.NewValue(op, resultType)
	v.AddArg(left)
	v.AddArg(right)
	return v, nil
}

// generateShortCircuit generates IR for && and || with short-circuit evaluation.
func (g *Generator) generateShortCircuit(be *semantic.TypedBinaryExpr) (*Value, error) {
	// Generate left operand
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}

	// Create blocks for short-circuit
	rightBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Set up branch based on operator
	g.block.Kind = BlockIf
	g.block.Control = left
	leftBlock := g.block

	if be.Op == "&&" {
		// AND: if left is false, skip right (result is false)
		g.block.AddSucc(rightBlock) // true -> evaluate right
		g.block.AddSucc(mergeBlock) // false -> skip
	} else {
		// OR: if left is true, skip right (result is true)
		g.block.AddSucc(mergeBlock) // true -> skip
		g.block.AddSucc(rightBlock) // false -> evaluate right
	}

	// Generate right operand
	g.block = rightBlock
	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}
	rightBlock = g.block // May have changed
	rightBlock.AddSucc(mergeBlock)

	// Create phi in merge block
	g.block = mergeBlock

	phi := g.block.NewPhiValue(TypeBool)
	// The short-circuit constant flows in from leftBlock, so materialize it
	// there rather than in the merge block. Emitting it in the merge block
	// would leave a trailing OpConst as the block's last value, which
	// generateIfExpr's getLastValue() would mistake for the if-branch result.
	if be.Op == "&&" {
		// AND: false from left, right's value from right
		falseVal := leftBlock.NewValue(OpConst, TypeBool)
		falseVal.AuxInt = 0
		phi.PhiArgs = []*PhiArg{
			{From: leftBlock, Value: falseVal},
			{From: rightBlock, Value: right},
		}
	} else {
		// OR: true from left, right's value from right
		trueVal := leftBlock.NewValue(OpConst, TypeBool)
		trueVal.AuxInt = 1
		phi.PhiArgs = []*PhiArg{
			{From: leftBlock, Value: trueVal},
			{From: rightBlock, Value: right},
		}
	}

	return phi, nil
}

// generateElvis generates IR for the elvis operator (a ?: b).
// Returns a if a is not null, otherwise returns b.
func (g *Generator) generateElvis(be *semantic.TypedBinaryExpr) (*Value, error) {
	// Generate left operand
	left, err := g.generateExpr(be.Left)
	if err != nil {
		return nil, err
	}

	// Three blocks: rightBlock evaluates the default, notNullBlock unwraps
	// the left operand, mergeBlock joins them. Unwrap must happen on the
	// not-null edge — value-type nullables are heap pointers and unwrap
	// dereferences, so it is unsafe to emit unconditionally.
	rightBlock := g.fn.NewBlock(BlockPlain)
	notNullBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Check if left is null
	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(left)

	// Branch: null -> rightBlock, not-null -> notNullBlock
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(rightBlock)
	g.block.AddSucc(notNullBlock)

	g.sealBlock(rightBlock)
	g.sealBlock(notNullBlock)

	// Right operand on the null edge
	g.block = rightBlock
	right, err := g.generateExpr(be.Right)
	if err != nil {
		return nil, err
	}
	rightBlock = g.block // may have changed during generation
	rightBlock.AddSucc(mergeBlock)

	// Unwrap on the not-null edge
	resultType := g.convertType(be.Type)
	g.block = notNullBlock
	unwrapped := g.block.NewValue(OpUnwrap, resultType)
	unwrapped.AddArg(left)
	// If the LHS produced a temporary heap allocation that no variable owns
	// (e.g., a function call returning T?, a wrap from safe navigation),
	// the heap slot would otherwise leak. Free it after the unwrap, before
	// the value is returned.
	if isOwningTemp(be.Left) {
		freeOwningTemp(g, left, be.Left.GetType())
	}
	notNullBlockEnd := g.block
	notNullBlockEnd.AddSucc(mergeBlock)

	g.sealBlock(mergeBlock)
	g.block = mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: rightBlock, Value: right},
		{From: notNullBlockEnd, Value: unwrapped},
	}

	return phi, nil
}

// generateUnary generates IR for a unary expression.
func (g *Generator) generateUnary(ue *semantic.TypedUnaryExpr) (*Value, error) {
	operand, err := g.generateExpr(ue.Operand)
	if err != nil {
		return nil, err
	}

	b := g.builder()
	resultType := g.convertType(ue.Type)

	switch ue.Op {
	case "!":
		return b.Not(operand), nil
	case "-":
		return b.Neg(resultType, operand), nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", ue.Op)
	}
}

// generateCall generates IR for a function call.
func (g *Generator) generateCall(ce *semantic.TypedCallExpr) (*Value, error) {
	// Resolve the callee's semantic param types. Under the scope-frees-it model
	// parameters are values or borrows (never owned *T), so arguments are always
	// borrowed — the caller frees any heap temporary it passed. The IR collapses
	// pointer kinds into PtrType.
	calleeName := ce.Name
	if strings.Contains(calleeName, ".") {
		calleeName = strings.ReplaceAll(calleeName, "/", "__")
		calleeName = strings.ReplaceAll(calleeName, ".", "__")
	}

	// Built-in vec operations lower to dedicated IR ops. The vec argument is a
	// borrowed pointer the op reads/mutates in place — no copy.
	switch ce.Name {
	case "vec":
		return g.builder().VecNew(), nil
	case "push":
		vec, err := g.generateExpr(ce.Arguments[0])
		if err != nil {
			return nil, err
		}
		val, err := g.generateExpr(ce.Arguments[1])
		if err != nil {
			return nil, err
		}
		return g.builder().VecPush(vec, val), nil
	case "get":
		vec, err := g.generateExpr(ce.Arguments[0])
		if err != nil {
			return nil, err
		}
		idx, err := g.generateExpr(ce.Arguments[1])
		if err != nil {
			return nil, err
		}
		return g.builder().VecGet(vec, idx), nil
	case "set":
		vec, err := g.generateExpr(ce.Arguments[0])
		if err != nil {
			return nil, err
		}
		idx, err := g.generateExpr(ce.Arguments[1])
		if err != nil {
			return nil, err
		}
		val, err := g.generateExpr(ce.Arguments[2])
		if err != nil {
			return nil, err
		}
		return g.builder().VecSet(vec, idx, val), nil
	}

	calleeParams := g.funcSemanticParams[calleeName]

	// Generate arguments
	var args []*Value
	var strTempFrees []*Value // fresh string temps to free after the call
	// Aggregate (struct/class/array) temporaries the callee only borrows;
	// the caller must free them after the call or they leak.
	type aggTemp struct {
		val *Value
		sem semantic.Type
	}
	var aggTempFrees []aggTemp
	for i, arg := range ce.Arguments {
		v, err := g.generateExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)

		// Strings are borrowed by callees, so a fresh string temporary passed
		// as an argument (interpolation result, string-returning call) is no
		// longer referenced after the call and must be freed.
		if isOwnedStringTemp(arg) {
			strTempFrees = append(strTempFrees, v)
		} else if isOwningTemp(arg) && isHeapValueType(arg.GetType()) {
			// Struct/class/array values are borrowed by callees (parameters
			// are never owned *T). A temp produced by a literal or call result
			// has no owner to free it at scope exit — free it here.
			transfers := i < len(calleeParams) && argTransfersOwnership(arg.GetType(), calleeParams[i])
			if !transfers {
				aggTempFrees = append(aggTempFrees, aggTemp{val: v, sem: arg.GetType()})
			}
		}
	}

	resultType := g.convertSSAType(ce.Type)

	callName := calleeName

	// exit() terminates the program before reaching scope boundaries, so
	// emit cleanup for every owned variable in scope before the call. This
	// keeps the language guarantee — owned heap is released when its owner
	// goes out of scope — true even for early termination paths.
	if callName == "exit" {
		g.emitAllScopesCleanup("")
	}

	// Wrap args to match nullable parameter types. Track which args were
	// freshly wrapped — those are caller-owned heap allocations passed by
	// borrow to the callee, so the caller frees them after the call.
	var wrappedTemps []*Value
	if targetFn := g.prog.FunctionByName(callName); targetFn != nil {
		for i, arg := range args {
			if i < len(targetFn.Params) {
				paramType := targetFn.Params[i].Type
				if _, isNullable := paramType.(*NullableType); isNullable {
					if arg.Op == OpWrapNull {
						// null literal — update its type to match the param
						arg.Type = paramType
					} else if _, argIsNullable := arg.Type.(*NullableType); !argIsNullable {
						// Non-nullable value — wrap it. The wrap allocates a
						// heap slot only for legacy boxed value-type nullables
						// (string?/array?); reference and flat nullables are not
						// heap-backed, so only boxed wraps need freeing.
						wrap := g.block.NewValue(OpWrap, paramType)
						wrap.AddArg(arg)
						args[i] = wrap
						if nt, ok := paramType.(*NullableType); ok && !nt.IsReferenceNullable() && !nt.IsFlat() {
							wrappedTemps = append(wrappedTemps, wrap)
						}
					}
				}
			}
		}
	}

	callVal := g.builder().Call(callName, resultType, args...)

	// Free the wrap-temps after the call. Their heap slots are no longer
	// referenced — the callee borrowed them and won't free.
	for _, wrap := range wrappedTemps {
		nt := wrap.Type.(*NullableType)
		size := 8
		if elemSize := nt.Elem.Size(); elemSize > 8 {
			size = elemSize
		}
		g.emitNullCheckedFree(wrap, size)
	}

	// Free fresh string-temp arguments now that the callee has used them.
	for _, sv := range strTempFrees {
		g.builder().StrFree(sv)
	}

	// Free borrowed aggregate temporaries now that the callee has used them.
	for _, at := range aggTempFrees {
		freeOwningTemp(g, at.val, at.sem)
	}

	return callVal, nil
}

// generateFieldAccess generates IR for field access.
func (g *Generator) generateFieldAccess(fa *semantic.TypedFieldAccessExpr) (*Value, error) {
	// Generate object
	obj, err := g.generateExpr(fa.Object)
	if err != nil {
		return nil, err
	}

	// Get field offset
	offset := g.getFieldOffset(fa.Object.GetType(), fa.Field)

	resultType := g.convertType(fa.Type)

	// Create field pointer
	fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: resultType})
	fieldPtr.AddArg(obj)
	fieldPtr.AuxInt = int64(offset)

	// For struct/class types embedded by value, return the pointer directly
	// (the caller will use it for further field access or as needed)
	// Only load primitive values (int, bool, string) and pointers
	switch resultType.(type) {
	case *StructType:
		// Return pointer to embedded struct, don't load
		return fieldPtr, nil
	default:
		// Load primitive value
		load := g.block.NewValue(OpLoad, resultType)
		load.AddArg(fieldPtr)
		return load, nil
	}
}

// generateIndex generates IR for array or string index access.
func (g *Generator) generateIndex(ie *semantic.TypedIndexExpr) (*Value, error) {
	// Generate array and index
	arr, err := g.generateExpr(ie.Array)
	if err != nil {
		return nil, err
	}

	idx, err := g.generateExpr(ie.Index)
	if err != nil {
		return nil, err
	}

	// String byte index: emit OpStringIndex (handles bounds check + byte load in backend)
	if _, isString := ie.Array.GetType().(semantic.StringType); isString {
		v := g.block.NewValue(OpStringIndex, TypeU8)
		v.AddArg(arr)
		v.AddArg(idx)
		return v, nil
	}

	// Array slots hold struct/class elements as pointers, so elements read
	// at SSA (pointer) representation.
	resultType := g.convertSSAType(ie.Type)

	// Create index pointer
	elemPtr := g.block.NewValue(OpIndexPtr, &PtrType{Elem: resultType})
	elemPtr.AddArg(arr)
	elemPtr.AddArg(idx)

	// Load value
	load := g.block.NewValue(OpLoad, resultType)
	load.AddArg(elemPtr)

	return load, nil
}

// generateLen generates IR for len() builtin.
func (g *Generator) generateLen(le *semantic.TypedLenExpr) (*Value, error) {
	// String: runtime length load from the header
	if le.Array != nil {
		if _, isString := le.Array.GetType().(semantic.StringType); isString {
			strVal, err := g.generateExpr(le.Array)
			if err != nil {
				return nil, err
			}
			v := g.block.NewValue(OpStringLen, TypeS64)
			v.AddArg(strVal)
			return v, nil
		}
		// vec: runtime length from the header
		if isVecType(le.Array.GetType()) {
			vecVal, err := g.generateExpr(le.Array)
			if err != nil {
				return nil, err
			}
			return g.builder().VecLen(vecVal), nil
		}
	}
	// Array: size known at compile time
	v := g.block.NewValue(OpConst, TypeS64)
	v.AuxInt = int64(le.ArraySize)
	return v, nil
}

// generateArrayLiteral generates IR for an array literal.
func (g *Generator) generateArrayLiteral(al *semantic.TypedArrayLiteralExpr) (*Value, error) {
	elemType := g.convertSSAType(al.Type.ElementType)
	arrayType := &ArrayType{Elem: elemType, Len: al.Type.Size}
	b := g.builder()

	// Allocate space
	alloc := b.Alloc(arrayType, int64(arrayType.Size()))

	// Store each element
	for i, elemExpr := range al.Elements {
		// An identifier element that owns a heap slot (value-type nullable, or a
		// struct/array value) aliases that slot into the array. These are
		// copyable types, so semantic allows the alias; mark the source aliased so
		// it isn't double-freed at scope exit. Strings use copy semantics
		// (handled below) and container reads are copied, not aliased.
		if ident, ok := elemExpr.(*semantic.TypedIdentifierExpr); ok {
			if varOwnsHeap(ident.Type) && !isStringType(ident.Type) {
				g.markHeapAliased(ident.Name)
			}
		}

		elem, err := g.generateExpr(elemExpr)
		if err != nil {
			return nil, err
		}

		// Container-read elements alias the source — copy so the array owns
		// its own heap slot. copyNullableValue introduces new blocks, so
		// refresh the builder before emitting the element store.
		if shouldCopyOnReturn(elemExpr) {
			elem = g.copyNullableValue(elem, elemExpr.GetType())
			b = g.builder()
		}

		// Borrowed string elements must be copied so the array owns its own
		// buffer (value semantics).
		elem = g.maybeCopyString(elem, elemExpr)

		idx := b.ConstInt(TypeS64, int64(i))
		elemPtr := b.IndexPtr(alloc, idx, elemType)
		b.Store(elemPtr, elem)
	}

	return alloc, nil
}

// generateStructLiteral generates IR for a struct literal.
func (g *Generator) generateStructLiteral(sl *semantic.TypedStructLiteralExpr) (*Value, error) {
	structType := g.convertStructType(&sl.Type)
	b := g.builder()

	// Allocate space
	alloc := b.Alloc(structType, int64(structType.Size()))

	// Store each field
	for i, argExpr := range sl.Args {
		fieldType := structType.Fields[i].Type
		fieldOffset := int64(structType.Fields[i].Offset)

		// Generate field value with null/nullable handling
		arg, err := g.generateTypedValue(argExpr, fieldType)
		if err != nil {
			return nil, err
		}

		fieldPtr := b.FieldPtr(alloc, fieldType, fieldOffset)

		// For embedded struct fields, copy the data instead of storing a
		// pointer. The source is then a temporary heap allocation that
		// nothing references, so free it.
		if _, isStruct := fieldType.(*StructType); isStruct {
			b.MemCopy(fieldPtr, arg, int64(fieldType.Size()))
			if isFreshAlloc(argExpr) {
				freeVal := g.block.NewValue(OpFree, nil, arg)
				freeVal.AuxInt = int64(fieldType.Size())
			}
		} else {
			// Borrowed string fields must be copied so the struct owns its own
			// buffer (value semantics); fresh strings transfer ownership.
			arg = g.maybeCopyString(arg, argExpr)
			b.Store(fieldPtr, arg)
		}
	}

	return alloc, nil
}

// argTransfersOwnership reports whether passing an argument transfers ownership
// of its heap allocation to the callee. Under the scope-frees-it model a
// parameter can never be an owned pointer (the semantic analyzer rejects *T
// params), so for valid programs this is always false: aggregate and string
// temporaries are borrowed and freed by the caller. It is kept as a defensive
// guard documenting that contract.
func argTransfersOwnership(argType, paramType semantic.Type) bool {
	if !varOwnsHeap(argType) {
		return false
	}
	// A parameter is never an owned pointer under this model; these branches are
	// unreachable for valid programs and remain only as a defensive guard.
	switch paramType.(type) {
	case *semantic.OwnedPointerType, semantic.OwnedPointerType:
		return true
	}
	if isNullableOwnedType(paramType) {
		return true
	}
	return false
}

// isFreshAlloc reports whether evaluating expr produces a brand-new heap
// allocation owned by no one — typically an inline literal expression.
// Identifiers and other expressions that may reference an existing binding
// are conservatively treated as not-fresh.
func isFreshAlloc(expr semantic.TypedExpression) bool {
	switch expr.(type) {
	case *semantic.TypedStructLiteralExpr,
		*semantic.TypedClassLiteralExpr,
		*semantic.TypedArrayLiteralExpr:
		return true
	}
	return false
}

// generateClassLiteral generates IR for a class literal.
func (g *Generator) generateClassLiteral(cl *semantic.TypedClassLiteralExpr) (*Value, error) {
	classType := g.convertClassType(&cl.Type)
	b := g.builder()

	// Allocate space
	alloc := b.Alloc(classType, int64(classType.Size()))

	// Store each field
	for i, argExpr := range cl.Args {
		fieldType := classType.Fields[i].Type
		fieldOffset := int64(classType.Fields[i].Offset)

		// Generate field value with null/nullable handling
		arg, err := g.generateTypedValue(argExpr, fieldType)
		if err != nil {
			return nil, err
		}

		fieldPtr := b.FieldPtr(alloc, fieldType, fieldOffset)

		// For embedded aggregate fields (struct/class, both IR *StructType),
		// copy the data inline instead of storing a pointer. The source is then
		// a temporary heap allocation that nothing references, so free it.
		if _, isStruct := fieldType.(*StructType); isStruct {
			b.MemCopy(fieldPtr, arg, int64(fieldType.Size()))
			if isFreshAlloc(argExpr) {
				freeVal := g.block.NewValue(OpFree, nil, arg)
				freeVal.AuxInt = int64(fieldType.Size())
			}
		} else {
			// Borrowed string fields must be copied so the class owns its own
			// buffer (value semantics); fresh strings transfer ownership.
			arg = g.maybeCopyString(arg, argExpr)
			b.Store(fieldPtr, arg)
		}
	}

	return alloc, nil
}

// generateMethodCall generates IR for a method call.
func (g *Generator) generateMethodCall(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	// Special handling for .copy()
	if mc.Method == "copy" && len(mc.Arguments) == 0 {
		return g.generateCopy(mc)
	}

	// Handle safe navigation (?.method())
	if mc.SafeNavigation {
		return g.generateSafeMethodCall(mc)
	}

	// Generate object (becomes first argument for instance methods)
	obj, err := g.generateExpr(mc.Object)
	if err != nil {
		return nil, err
	}

	// Generate other arguments
	var args []*Value
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		args = append(args, obj)
	}
	for _, arg := range mc.Arguments {
		v, err := g.generateExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	// Determine mangled function name (include param count for overloading)
	mangledName := g.mangleMethodName(mc.Object.GetType(), mc)

	resultType := g.convertSSAType(mc.Type)

	call := g.block.NewValue(OpCall, resultType)
	call.AuxString = mangledName
	for _, arg := range args {
		call.AddArg(arg)
	}

	// Receivers that own heap but aren't bound to a name are temporaries:
	// literal receivers (Point{3,4}.foo()) and chained call results
	// (b.foo().bar()). After the outer call finishes using the receiver as
	// a borrow, free the temp — scope cleanup never sees it.
	if isOwningTemp(mc.Object) {
		freeOwningTemp(g, obj, mc.Object.GetType())
	}

	return call, nil
}

// isOwningTemp reports whether evaluating expr produces a heap allocation
// that is not bound to any variable: literal expressions, and method/call
// expressions whose return type owns heap.
func isOwningTemp(expr semantic.TypedExpression) bool {
	if isFreshAlloc(expr) {
		return true
	}
	switch expr.(type) {
	case *semantic.TypedMethodCallExpr, *semantic.TypedCallExpr:
		return varOwnsHeap(expr.GetType())
	}
	return false
}

// freeOwningTemp emits the free for a temporary heap allocation of the given
// semantic type, dispatching to recursive-free for owned pointers and a
// nullable-aware free for nullable values.
func freeOwningTemp(g *Generator, val *Value, semType semantic.Type) {
	if elemType, size := g.getOwnedPointerInfo(semType); elemType != nil {
		if isNullableOwnedType(semType) {
			freeBlock := g.fn.NewBlock(BlockPlain)
			continueBlock := g.fn.NewBlock(BlockPlain)
			isNull := g.block.NewValue(OpIsNull, TypeBool, val)
			g.block.Kind = BlockIf
			g.block.Control = isNull
			g.block.AddSucc(continueBlock)
			g.block.AddSucc(freeBlock)
			g.block = freeBlock
			unwrapped := g.block.NewValue(OpUnwrap, &PtrType{Elem: g.convertType(elemType)}, val)
			g.emitRecursiveFree(unwrapped, elemType, size)
			g.block.AddSucc(continueBlock)
			g.sealBlock(freeBlock)
			g.block = continueBlock
			g.sealBlock(continueBlock)
			return
		}
		g.emitRecursiveFree(val, elemType, size)
		return
	}
	if inner := nullableValueInner(semType); inner != nil {
		g.emitNullCheckedFree(val, g.nullableValueAllocSize(inner))
		return
	}
	if isHeapValueType(semType) {
		size := g.getElementTypeSize(semType)
		g.emitRecursiveFree(val, semType, size)
	}
}

// generateSafeMethodCall generates IR for safe method call (?.).
// If the object is null, returns null; otherwise calls the method.
func (g *Generator) generateSafeMethodCall(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	obj, err := g.generateExpr(mc.Object)
	if err != nil {
		return nil, err
	}

	resultType := g.convertSSAType(mc.Type)
	blocks := g.createNullCheckBlocks()

	// Branch on null check
	g.emitNullCheck(obj, blocks)

	// Null path: return wrapped null
	g.block = blocks.nullBlock
	nullResult := g.block.NewValue(OpWrapNull, resultType)
	blocks.nullBlock.AddSucc(blocks.mergeBlock)

	// Not-null path: unwrap, call method, wrap result
	g.block = blocks.notNullBlock
	wrapped, err := g.generateUnwrappedMethodCall(obj, mc, resultType)
	if err != nil {
		return nil, err
	}
	blocks.notNullBlock = g.block // block may have changed during arg generation
	blocks.notNullBlock.AddSucc(blocks.mergeBlock)

	// Merge results with phi node
	return g.mergeNullCheckResults(blocks, resultType, nullResult, wrapped)
}

// nullCheckBlocks holds the basic blocks for a null-check pattern.
type nullCheckBlocks struct {
	nullBlock    *Block
	notNullBlock *Block
	mergeBlock   *Block
}

// createNullCheckBlocks creates the three blocks needed for null-check control flow.
func (g *Generator) createNullCheckBlocks() *nullCheckBlocks {
	return &nullCheckBlocks{
		nullBlock:    g.fn.NewBlock(BlockPlain),
		notNullBlock: g.fn.NewBlock(BlockPlain),
		mergeBlock:   g.fn.NewBlock(BlockPlain),
	}
}

// emitNullCheck generates IR to branch based on whether a value is null.
func (g *Generator) emitNullCheck(val *Value, blocks *nullCheckBlocks) {
	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(val)

	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(blocks.nullBlock)
	g.block.AddSucc(blocks.notNullBlock)

	g.sealBlock(blocks.nullBlock)
	g.sealBlock(blocks.notNullBlock)
}

// generateUnwrappedMethodCall generates a method call on an unwrapped nullable value.
func (g *Generator) generateUnwrappedMethodCall(
	obj *Value,
	mc *semantic.TypedMethodCallExpr,
	resultType Type,
) (*Value, error) {
	// Unwrap the nullable object
	innerType := UnwrapNullableType(mc.Object.GetType())
	unwrappedIRType := g.convertType(innerType)
	unwrapped := g.block.NewValue(OpUnwrap, unwrappedIRType)
	unwrapped.AddArg(obj)

	// Build argument list
	args, err := g.buildMethodArgs(unwrapped, mc)
	if err != nil {
		return nil, err
	}

	// Generate the call
	mangledName := g.mangleMethodName(innerType, mc)
	callResultType := UnwrapIRNullableType(resultType)

	call := g.block.NewValue(OpCall, callResultType)
	call.AuxString = mangledName
	for _, arg := range args {
		call.AddArg(arg)
	}

	// Wrap result as nullable
	wrapped := g.block.NewValue(OpWrap, resultType)
	wrapped.AddArg(call)
	return wrapped, nil
}

// buildMethodArgs builds the argument list for a method call.
func (g *Generator) buildMethodArgs(receiver *Value, mc *semantic.TypedMethodCallExpr) ([]*Value, error) {
	var args []*Value

	// Add receiver for instance methods
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		args = append(args, receiver)
	}

	// Add explicit arguments
	for _, arg := range mc.Arguments {
		v, err := g.generateExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	return args, nil
}

// mangleMethodName generates the mangled name for a method call.
func (g *Generator) mangleMethodName(receiverType semantic.Type, mc *semantic.TypedMethodCallExpr) string {
	className := g.getTypeName(receiverType)
	paramCount := len(mc.Arguments)
	if mc.ResolvedMethod != nil && !mc.ResolvedMethod.IsStatic {
		paramCount++ // Count self parameter
	}

	// Type suffix must match generateMethod's, derived from the resolved
	// method's declared parameter types (which include self for instance
	// methods, matching the count above).
	suffix := ""
	if mc.ResolvedMethod != nil {
		suffix = mangleParamSuffix(mc.ResolvedMethod.ParamTypes)
	}

	// For cross-package types, prefix with the package path
	pkgPath := g.getTypePackagePath(receiverType)
	if pkgPath != "" && pkgPath != "main" {
		prefix := strings.ReplaceAll(pkgPath, "/", "__") + "__"
		return fmt.Sprintf("%s%s_%s_%d%s", prefix, className, mc.Method, paramCount, suffix)
	}

	return fmt.Sprintf("%s_%s_%d%s", className, mc.Method, paramCount, suffix)
}

// getTypePackagePath returns the PackagePath for a nominal type, unwrapping pointers/nullables.
func (g *Generator) getTypePackagePath(t semantic.Type) string {
	// Unwrap pointer types
	if ownedPtr, ok := t.(semantic.OwnedPointerType); ok {
		return g.getTypePackagePath(ownedPtr.ElementType)
	}
	if refPtr, ok := t.(semantic.RefPointerType); ok {
		return g.getTypePackagePath(refPtr.ElementType)
	}
	if mutRefPtr, ok := t.(semantic.MutRefPointerType); ok {
		return g.getTypePackagePath(mutRefPtr.ElementType)
	}
	// Unwrap nullable
	if nt, ok := t.(semantic.NullableType); ok {
		return g.getTypePackagePath(nt.InnerType)
	}

	switch ty := t.(type) {
	case semantic.StructType:
		return ty.PackagePath
	case *semantic.StructType:
		return ty.PackagePath
	case semantic.ClassType:
		return ty.PackagePath
	case *semantic.ClassType:
		return ty.PackagePath
	case semantic.ObjectType:
		return ty.PackagePath
	case *semantic.ObjectType:
		return ty.PackagePath
	}
	return ""
}

// mergeNullCheckResults creates a phi node to merge null and non-null paths.
func (g *Generator) mergeNullCheckResults(
	blocks *nullCheckBlocks,
	resultType Type,
	nullResult, notNullResult *Value,
) (*Value, error) {
	g.sealBlock(blocks.mergeBlock)
	g.block = blocks.mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: blocks.nullBlock, Value: nullResult},
		{From: blocks.notNullBlock, Value: notNullResult},
	}

	return phi, nil
}

// generateNewExpr generates IR for a 'new' expression (e.g., new Point{ 10, 20 }).
func (g *Generator) generateNewExpr(expr *semantic.TypedNewExpr) (*Value, error) {
	// If the operand is a struct or class literal, it already allocates on heap.
	// Just generate it directly and return the pointer.
	switch expr.Operand.(type) {
	case *semantic.TypedStructLiteralExpr, *semantic.TypedClassLiteralExpr:
		return g.generateExpr(expr.Operand)
	}

	// For other expressions, we need to allocate and copy.
	val, err := g.generateExpr(expr.Operand)
	if err != nil {
		return nil, err
	}

	// Get size of the value
	valType := val.Type
	size := valType.Size()

	// Allocate on heap
	alloc := g.block.NewValue(OpAlloc, &PtrType{Elem: valType})
	alloc.AuxInt = int64(size)

	// Store initial value
	store := g.block.NewValue(OpStore, nil)
	store.AddArg(alloc)
	store.AddArg(val)

	return alloc, nil
}

// aggregateIsCopyable reports whether an aggregate (struct/class/array) type
// has copy semantics. Normalizes pointer/value forms before consulting
// semantic.IsCopyable (which only matches value forms). Classes are non-copyable.
func (g *Generator) aggregateIsCopyable(t semantic.Type) bool {
	if st := g.getSemanticStructType(t); st != nil {
		return semantic.IsCopyable(*st)
	}
	if at, ok := asSemanticArrayType(t); ok {
		return semantic.IsCopyable(*at)
	}
	return false
}

// bindAggregateValue implements value semantics when binding an aggregate
// (struct/class/array) read to a variable. Fresh temporaries (literals, call
// results) already belong to the new binding. Aliasing reads of a copyable
// aggregate are deep-copied so each binding owns independent storage.
// A non-copyable aggregate passes its source allocation through unchanged.
func (g *Generator) bindAggregateValue(val *Value, expr semantic.TypedExpression) *Value {
	t := expr.GetType()
	if !isHeapValueType(t) {
		return val
	}
	if isOwningTemp(expr) {
		return val
	}
	if g.aggregateIsCopyable(t) {
		return g.emitDeepCopyAggregate(val, t)
	}
	return val
}

// emitDeepCopyAggregate returns a pointer to a fresh deep copy of the
// copyable aggregate (struct or array) at src. Non-copyable aggregates never
// reach here, so owned-pointer fields cannot appear at any depth and the
// recursion terminates.
func (g *Generator) emitDeepCopyAggregate(src *Value, semType semantic.Type) *Value {
	b := g.builder()
	size := g.getElementTypeSize(semType)
	alloc := b.Alloc(g.convertType(semType), int64(size))
	b.MemCopy(alloc, src, int64(size))
	g.emitDeepCopyFixups(alloc, semType)
	return alloc
}

// emitDeepCopyFixups walks a freshly shallow-copied aggregate at dst and
// replaces fields/elements that reference heap storage owned by the source
// with independent copies.
func (g *Generator) emitDeepCopyFixups(dst *Value, semType semantic.Type) {
	b := g.builder()

	if at, ok := asSemanticArrayType(semType); ok {
		elemSem := at.ElementType
		if !isStringType(elemSem) && !isHeapValueType(elemSem) && nullableValueInner(elemSem) == nil {
			return
		}
		elemIRType := g.convertSSAType(elemSem)
		for i := 0; i < at.Size; i++ {
			idx := g.block.NewValue(OpConst, TypeS64)
			idx.AuxInt = int64(i)
			elemPtr := g.block.NewValue(OpIndexPtr, &PtrType{Elem: elemIRType}, dst, idx)
			elemVal := g.block.NewValue(OpLoad, elemIRType, elemPtr)
			b.Store(elemPtr, g.deepCopiedValue(elemVal, elemSem))
		}
		return
	}

	if st := g.getSemanticStructType(semType); st != nil {
		for _, field := range st.Fields {
			offset := int64(st.FieldOffset(field.Name))
			fieldIRType := g.convertType(field.Type)

			// Embedded struct field: its bytes were already copied in place;
			// fix up its heap-referencing fields through the embedded region.
			if g.getSemanticStructType(field.Type) != nil {
				fieldPtr := b.FieldPtr(dst, fieldIRType, offset)
				g.emitDeepCopyFixups(fieldPtr, field.Type)
				continue
			}

			if isStringType(field.Type) || isVecType(field.Type) ||
				nullableValueInner(field.Type) != nil || isHeapValueType(field.Type) {
				fieldPtr := b.FieldPtr(dst, fieldIRType, offset)
				fieldVal := g.block.NewValue(OpLoad, fieldIRType, fieldPtr)
				b.Store(fieldPtr, g.deepCopiedValue(fieldVal, field.Type))
			}
		}
	}
}

// deepCopiedValue returns an independently owned copy of val, dispatching on
// the semantic type: strings copy their buffer, value-type nullables copy
// their heap slot (preserving null), and nested aggregates copy recursively.
// Plain values are returned unchanged.
func (g *Generator) deepCopiedValue(val *Value, semType semantic.Type) *Value {
	if isStringType(semType) {
		return g.builder().StrCopy(val)
	}
	if isVecType(semType) {
		return g.builder().VecCopy(val)
	}
	if nullableValueInner(semType) != nil {
		return g.copyNullableValue(val, semType)
	}
	if isHeapValueType(semType) {
		return g.emitDeepCopyAggregate(val, semType)
	}
	return val
}

// generateCopy generates IR for .copy() method.
func (g *Generator) generateCopy(mc *semantic.TypedMethodCallExpr) (*Value, error) {
	// Generate object to copy
	obj, err := g.generateExpr(mc.Object)
	if err != nil {
		return nil, err
	}

	// Create deep copy
	copy := g.block.NewValue(OpCopy, obj.Type)
	copy.AddArg(obj)

	return copy, nil
}

// generateSafeCall generates IR for safe navigation (?.).
func (g *Generator) generateSafeCall(sc *semantic.TypedSafeCallExpr) (*Value, error) {
	// Generate object
	obj, err := g.generateExpr(sc.Object)
	if err != nil {
		return nil, err
	}

	resultType := g.convertSSAType(sc.Type)

	// Check if null
	isNull := g.block.NewValue(OpIsNull, TypeBool)
	isNull.AddArg(obj)

	// Create blocks
	notNullBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Create null result in current block (before branching)
	// This value will be used by the phi when we take the null path
	nullResult := g.block.NewValue(OpWrapNull, resultType)

	// Branch
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(mergeBlock) // is null -> skip
	g.block.AddSucc(notNullBlock)
	nullBlock := g.block

	// Not null path: access field
	g.block = notNullBlock

	// Unwrap nullable
	// sc.InnerType is the struct type when ThroughPointer is true (pointer was auto-dereferenced)
	// We need the actual unwrapped type which includes the pointer wrapper
	innerType := g.convertType(sc.InnerType)
	unwrapType := innerType
	if sc.ThroughPointer {
		// When accessing through pointer (e.g., *TreeNode?), unwrap gives *TreeNode
		unwrapType = &PtrType{Elem: innerType}
	}
	unwrapped := g.block.NewValue(OpUnwrap, unwrapType)
	unwrapped.AddArg(obj)

	// Get field - FieldPtr operates on the pointer
	fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: resultType})
	fieldPtr.AddArg(unwrapped)
	fieldPtr.AuxInt = int64(sc.FieldOffset)

	fieldVal := g.block.NewValue(OpLoad, resultType)
	fieldVal.AddArg(fieldPtr)

	// Wrap result in nullable
	wrapped := g.block.NewValue(OpWrap, resultType)
	wrapped.AddArg(fieldVal)

	notNullBlock.AddSucc(mergeBlock)

	// Merge block
	g.block = mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: nullBlock, Value: nullResult},
		{From: notNullBlock, Value: wrapped},
	}

	return phi, nil
}

// generateSelf generates IR for self expression.
func (g *Generator) generateSelf(_ *semantic.TypedSelfExpr) (*Value, error) {
	// self is the first parameter
	return g.readVariable("self", g.block), nil
}

// generateIfExpr generates IR for an if expression (with result).
func (g *Generator) generateIfExpr(is *semantic.TypedIfStmt) (*Value, error) {
	if is.ResultType == nil {
		// Not an expression, generate as statement
		return nil, g.generateIf(is)
	}

	// Generate condition
	cond, err := g.generateExpr(is.Condition)
	if err != nil {
		return nil, err
	}

	resultType := g.convertType(is.ResultType)

	// Create blocks
	thenBlock := g.fn.NewBlock(BlockPlain)
	elseBlock := g.fn.NewBlock(BlockPlain)
	mergeBlock := g.fn.NewBlock(BlockPlain)

	// Branch
	g.block.Kind = BlockIf
	g.block.Control = cond
	g.block.AddSucc(thenBlock)
	g.block.AddSucc(elseBlock)

	// Generate then
	g.block = thenBlock
	if err := g.generateBlock(is.ThenBranch); err != nil {
		return nil, err
	}
	thenVal := g.getLastValue()
	thenEndBlock := g.block
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(mergeBlock)
	}

	// Generate else
	g.block = elseBlock
	if err := g.generateStatement(is.ElseBranch); err != nil {
		return nil, err
	}
	elseVal := g.getLastValue()
	elseEndBlock := g.block
	if g.block != nil && g.block.Kind == BlockPlain {
		g.block.AddSucc(mergeBlock)
	}

	// Merge
	g.block = mergeBlock

	phi := g.block.NewPhiValue(resultType)
	phi.PhiArgs = []*PhiArg{
		{From: thenEndBlock, Value: thenVal},
		{From: elseEndBlock, Value: elseVal},
	}

	return phi, nil
}

// ============================================================================
// SSA Construction
// ============================================================================

// writeVariable records a definition of a variable in a block.
// Delegates to SSABuilder.
func (g *Generator) writeVariable(name string, block *Block, val *Value) {
	pname := g.prefixedName(name)
	if _, isGlobal := g.globalVars[pname]; isGlobal {
		// Write to global variable via OpStoreGlobal
		store := block.NewValue(OpStoreGlobal, TypeVoid)
		store.AuxString = pname
		store.AddArg(val)
		// Also update SSA for reads within the same function
		g.ssa.WriteVariable(pname, block, val)
		return
	}
	g.ssa.WriteVariable(pname, block, val)
}

// readVariable returns the current definition of a variable.
// For global variables, emits OpLoadGlobal.
func (g *Generator) readVariable(name string, block *Block) *Value {
	pname := g.prefixedName(name)
	if globalType, isGlobal := g.globalVars[pname]; isGlobal {
		// Read from global variable via OpLoadGlobal
		load := block.NewValue(OpLoadGlobal, globalType)
		load.AuxString = pname
		return load
	}
	return g.ssa.ReadVariable(pname, block)
}

// prefixedName applies the package prefix to a variable name.
// Cross-package references (containing ".") are mangled separately, not prefixed.
// Function parameters (within the current function) are prefixed.
func (g *Generator) prefixedName(name string) string {
	// Cross-package references have dots (e.g., "math.add") and may have
	// slashes for nested packages (e.g., "graphics/color.red") — mangle both
	if strings.Contains(name, ".") {
		mangled := strings.ReplaceAll(name, "/", "__")
		mangled = strings.ReplaceAll(mangled, ".", "__")
		return mangled
	}
	return g.packagePrefix + name
}

// sealBlock marks a block as sealed (no more predecessors will be added).
// Delegates to SSABuilder.
func (g *Generator) sealBlock(block *Block) {
	g.ssa.SealBlock(block)
}

// ============================================================================
// Memory Management
// ============================================================================

// emitFreeIfOwned emits an OpFree if the variable owns heap storage:
// owned pointers (*T, T? where T is *X) free their pointee recursively;
// value-type nullables (s64?, bool?, ...) free the wrap heap slot;
// struct/class/array bindings free the literal's heap allocation.
//
// Variables whose heap slot was aliased into another owner (markHeapAliased) are
// skipped — freeing them would double-free the new owner.
func (g *Generator) emitFreeIfOwned(name string, semType semantic.Type) {
	if g.aliasedHeapVars[name] {
		return
	}

	// Strings: free the heap buffer (no-op for constant pointers).
	if isStringType(semType) {
		if !g.ssa.IsVariableDefinedOnAllPaths(name, g.block) {
			return
		}
		if oldVal := g.readVariable(name, g.block); oldVal != nil {
			g.builder().StrFree(oldVal)
		}
		return
	}

	// vec: free its header + data (no-op for non-heap pointers).
	if isVecType(semType) {
		if !g.ssa.IsVariableDefinedOnAllPaths(name, g.block) {
			return
		}
		if oldVal := g.readVariable(name, g.block); oldVal != nil {
			g.builder().VecFree(oldVal)
		}
		return
	}

	if inner := nullableValueInner(semType); inner != nil {
		g.emitFreeNullableValue(name, inner)
		return
	}

	// Plain struct/class/array variables: the binding owns the heap region
	// allocated by the literal expression. Recursive-free walks any owned
	// pointer fields nested inside.
	if isHeapValueType(semType) {
		if !g.ssa.IsVariableDefinedOnAllPaths(name, g.block) {
			return
		}
		oldVal := g.readVariable(name, g.block)
		if oldVal == nil {
			return
		}
		size := g.getElementTypeSize(semType)
		g.emitRecursiveFree(oldVal, semType, size)
		return
	}

	// Get the owned pointer element type and size
	elemType, size := g.getOwnedPointerInfo(semType)
	if elemType == nil {
		return // Not an owned pointer
	}

	// Check if variable is defined on all paths to current block
	// Variables defined only in loops/conditionals may not be available here
	if !g.ssa.IsVariableDefinedOnAllPaths(name, g.block) {
		return // Variable not defined on all paths, skip cleanup
	}

	// Read the current value
	oldVal := g.readVariable(name, g.block)
	if oldVal == nil {
		return
	}

	// Check if this is a nullable type
	if isNullableOwnedType(semType) {
		// For nullable, check if non-null before freeing
		freeBlock := g.fn.NewBlock(BlockPlain)
		continueBlock := g.fn.NewBlock(BlockPlain)

		// Check if null
		isNull := g.block.NewValue(OpIsNull, TypeBool, oldVal)

		// Branch: if null -> continue, if not null -> free
		g.block.Kind = BlockIf
		g.block.Control = isNull
		g.block.AddSucc(continueBlock) // null -> skip free
		g.block.AddSucc(freeBlock)     // not null -> free

		// Free block - unwrap and recursively free
		g.block = freeBlock
		unwrapped := g.block.NewValue(OpUnwrap, &PtrType{Elem: g.convertType(elemType)}, oldVal)
		g.emitRecursiveFree(unwrapped, elemType, size)
		g.block.AddSucc(continueBlock)
		g.sealBlock(freeBlock)

		// Continue block
		g.block = continueBlock
		g.sealBlock(continueBlock)
	} else {
		// Non-nullable owned pointer - recursively free it
		g.emitRecursiveFree(oldVal, elemType, size)
	}
}

// emitFreeNullableValue frees the heap slot owned by a value-type nullable
// variable (s64?, bool?, ...). The slot was allocated by genWrap; null values
// are skipped.
func (g *Generator) emitFreeNullableValue(name string, inner semantic.Type) {
	if !g.ssa.IsVariableDefinedOnAllPaths(name, g.block) {
		return
	}
	oldVal := g.readVariable(name, g.block)
	if oldVal == nil {
		return
	}
	g.emitNullCheckedFree(oldVal, g.nullableValueAllocSize(inner))
}

// emitNullCheckedFree emits a null-check followed by an OpFree of the given
// pointer with the given size. Safe to call with any pointer-shaped value;
// emits no-op control flow if the pointer is null.
func (g *Generator) emitNullCheckedFree(ptr *Value, size int) {
	freeBlock := g.fn.NewBlock(BlockPlain)
	continueBlock := g.fn.NewBlock(BlockPlain)

	isNull := g.block.NewValue(OpIsNull, TypeBool, ptr)
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(continueBlock) // null -> skip
	g.block.AddSucc(freeBlock)     // not null -> free

	g.block = freeBlock
	freeVal := g.block.NewValue(OpFree, nil, ptr)
	freeVal.AuxInt = int64(size)
	g.block.AddSucc(continueBlock)
	g.sealBlock(freeBlock)

	g.block = continueBlock
	g.sealBlock(continueBlock)
}

// emitFreeOwnedValue frees a freshly loaded value if its semantic type owns
// heap storage. Used at field/index assignment to release the old contents
// before overwriting. Mirrors the dispatch in emitFreeIfOwned but operates
// on a value rather than a tracked variable.
func (g *Generator) emitFreeOwnedValue(val *Value, semType semantic.Type) {
	if isStringType(semType) {
		g.builder().StrFree(val)
		return
	}

	if inner := nullableValueInner(semType); inner != nil {
		g.emitNullCheckedFree(val, g.nullableValueAllocSize(inner))
		return
	}

	elemType, size := g.getOwnedPointerInfo(semType)
	if elemType == nil {
		return
	}

	if isNullableOwnedType(semType) {
		freeBlock := g.fn.NewBlock(BlockPlain)
		continueBlock := g.fn.NewBlock(BlockPlain)

		isNull := g.block.NewValue(OpIsNull, TypeBool, val)
		g.block.Kind = BlockIf
		g.block.Control = isNull
		g.block.AddSucc(continueBlock)
		g.block.AddSucc(freeBlock)

		g.block = freeBlock
		unwrapped := g.block.NewValue(OpUnwrap, &PtrType{Elem: g.convertType(elemType)}, val)
		g.emitRecursiveFree(unwrapped, elemType, size)
		g.block.AddSucc(continueBlock)
		g.sealBlock(freeBlock)

		g.block = continueBlock
		g.sealBlock(continueBlock)
		return
	}

	g.emitRecursiveFree(val, elemType, size)
}

// emitRecursiveFree frees a struct and all its owned pointer fields recursively.
// ptr is the pointer to the struct, elemType is the semantic type of the element,
// and size is the byte size of the struct.
// The visiting map tracks types currently being processed to detect self-referential types.
func (g *Generator) emitRecursiveFree(ptr *Value, elemType semantic.Type, size int) {
	g.emitRecursiveFreeWithVisited(ptr, elemType, size, make(map[string]bool))
}

func (g *Generator) emitRecursiveFreeWithVisited(ptr *Value, elemType semantic.Type, size int, visiting map[string]bool) {
	// String: ptr is the string buffer itself. Free it directly (no-op for
	// constant pointers). Strings have no nested owned storage.
	if isStringType(elemType) {
		g.builder().StrFree(ptr)
		return
	}

	// Array: walk elements, recursively free those whose type owns heap, then
	// free the array's own allocation. Length is known at compile time, so
	// the walk is a compile-time unroll. Nullable elements are null-checked.
	if at, ok := asSemanticArrayType(elemType); ok {
		if varOwnsHeap(at.ElementType) {
			elemSize := g.getElementTypeSize(at.ElementType)
			elemIRType := g.convertSSAType(at.ElementType)
			needsNullCheck := nullableValueInner(at.ElementType) != nil ||
				isNullableOwnedType(at.ElementType)
			for i := 0; i < at.Size; i++ {
				idxVal := g.block.NewValue(OpConst, TypeS64)
				idxVal.AuxInt = int64(i)
				elemPtr := g.block.NewValue(OpIndexPtr, &PtrType{Elem: elemIRType}, ptr, idxVal)
				elemVal := g.block.NewValue(OpLoad, elemIRType, elemPtr)

				if needsNullCheck {
					if inner := nullableValueInner(at.ElementType); inner != nil {
						g.emitNullCheckedFree(elemVal, g.nullableValueAllocSize(inner))
					} else {
						// Nullable owned pointer: null-check then recursive free.
						freeBlock := g.fn.NewBlock(BlockPlain)
						continueBlock := g.fn.NewBlock(BlockPlain)
						isNull := g.block.NewValue(OpIsNull, TypeBool, elemVal)
						g.block.Kind = BlockIf
						g.block.Control = isNull
						g.block.AddSucc(continueBlock)
						g.block.AddSucc(freeBlock)
						g.block = freeBlock
						unwrapped := g.block.NewValue(OpUnwrap, elemIRType, elemVal)
						g.emitRecursiveFreeWithVisited(unwrapped, at.ElementType, elemSize, visiting)
						g.block.AddSucc(continueBlock)
						g.sealBlock(freeBlock)
						g.block = continueBlock
						g.sealBlock(continueBlock)
					}
				} else {
					g.emitRecursiveFreeWithVisited(elemVal, at.ElementType, elemSize, visiting)
				}
			}
		}
		freeVal := g.block.NewValue(OpFree, nil, ptr)
		freeVal.AuxInt = int64(size)
		return
	}

	// First, recursively free any owned pointer fields in this struct
	if st := g.getSemanticStructType(elemType); st != nil {
		// Check if this is a self-referential type (already being processed)
		typeName := st.Name
		if visiting[typeName] {
			// Self-referential type - generate a runtime loop instead of compile-time recursion
			g.emitRuntimeFreeLoop(ptr, st, size)
			return
		}

		// Mark as visiting
		visiting[typeName] = true
		defer func() { delete(visiting, typeName) }()

		for _, field := range st.Fields {
			// String field: load the buffer pointer and free it (no-op for
			// constant pointers).
			if isStringType(field.Type) {
				offset := st.FieldOffset(field.Name)
				fieldIRType := g.convertType(field.Type)
				fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: fieldIRType}, ptr)
				fieldPtr.AuxInt = int64(offset)
				fieldVal := g.block.NewValue(OpLoad, fieldIRType, fieldPtr)
				g.builder().StrFree(fieldVal)
				continue
			}

			// Vec field: load the header pointer and free it (no-op for non-heap).
			if isVecType(field.Type) {
				offset := st.FieldOffset(field.Name)
				fieldIRType := g.convertType(field.Type)
				fieldPtr := g.block.NewValue(OpFieldPtr, &PtrType{Elem: fieldIRType}, ptr)
				fieldPtr.AuxInt = int64(offset)
				fieldVal := g.block.NewValue(OpLoad, fieldIRType, fieldPtr)
				g.builder().VecFree(fieldVal)
				continue
			}

			// Value-type nullable field: load the pointer, null-check, free.
			if inner := nullableValueInner(field.Type); inner != nil {
				offset := st.FieldOffset(field.Name)
				fieldIRType := g.convertType(field.Type)
				fieldPtrType := &PtrType{Elem: fieldIRType}
				fieldPtr := g.block.NewValue(OpFieldPtr, fieldPtrType, ptr)
				fieldPtr.AuxInt = int64(offset)
				fieldVal := g.block.NewValue(OpLoad, fieldIRType, fieldPtr)
				g.emitNullCheckedFree(fieldVal, g.nullableValueAllocSize(inner))
				continue
			}

			if ownedElemType, fieldSize := g.getOwnedPointerInfo(field.Type); ownedElemType != nil {
				// This field is an owned pointer - need to free it
				offset := st.FieldOffset(field.Name)
				irElemType := g.convertType(ownedElemType)

				// Get pointer to the field
				fieldPtrType := &PtrType{Elem: g.convertType(field.Type)}
				fieldPtr := g.block.NewValue(OpFieldPtr, fieldPtrType, ptr)
				fieldPtr.AuxInt = int64(offset)

				// Load the field value
				fieldVal := g.block.NewValue(OpLoad, g.convertType(field.Type), fieldPtr)

				// Check if this is a nullable owned pointer
				if isNullableOwnedType(field.Type) {
					// Need to check for null before freeing
					freeFieldBlock := g.fn.NewBlock(BlockPlain)
					continueFieldBlock := g.fn.NewBlock(BlockPlain)

					isNull := g.block.NewValue(OpIsNull, TypeBool, fieldVal)
					g.block.Kind = BlockIf
					g.block.Control = isNull
					g.block.AddSucc(continueFieldBlock) // null -> skip
					g.block.AddSucc(freeFieldBlock)     // not null -> free

					// Free field block
					g.block = freeFieldBlock
					unwrapped := g.block.NewValue(OpUnwrap, &PtrType{Elem: irElemType}, fieldVal)
					g.emitRecursiveFreeWithVisited(unwrapped, ownedElemType, fieldSize, visiting)
					g.block.AddSucc(continueFieldBlock)
					g.sealBlock(freeFieldBlock)

					// Continue block
					g.block = continueFieldBlock
					g.sealBlock(continueFieldBlock)
				} else {
					// Non-nullable owned pointer field - just free it
					g.emitRecursiveFreeWithVisited(fieldVal, ownedElemType, fieldSize, visiting)
				}
			}
		}
	}

	// Now free the struct itself
	freeVal := g.block.NewValue(OpFree, nil, ptr)
	freeVal.AuxInt = int64(size)
}

// emitRuntimeFreeLoop generates a runtime loop to free a self-referential linked structure.
// This handles types like linked lists where the struct contains a pointer to the same type.
func (g *Generator) emitRuntimeFreeLoop(ptr *Value, st *semantic.StructType, size int) {
	// Look up the IR struct by name to get accurate field info
	// (the semantic st may have stale/empty fields due to two-pass registration)
	irStruct := g.prog.StructByName(st.Name)
	if irStruct == nil || len(irStruct.Fields) == 0 {
		// Can't find struct info, just free without recursion
		freeVal := g.block.NewValue(OpFree, nil, ptr)
		freeVal.AuxInt = int64(size)
		return
	}

	// Find the self-referential owned pointer field
	var selfRefField *StructField
	var selfRefOffset int
	for i := range irStruct.Fields {
		field := &irStruct.Fields[i]
		// Check if this field is a nullable pointer to the same struct type
		if nt, ok := field.Type.(*NullableType); ok {
			if pt, ok := nt.Elem.(*PtrType); ok {
				if innerSt, ok := pt.Elem.(*StructType); ok && innerSt.Name == st.Name {
					selfRefField = field
					selfRefOffset = field.Offset
					break
				}
			}
		}
		// Also check non-nullable pointer
		if pt, ok := field.Type.(*PtrType); ok {
			if innerSt, ok := pt.Elem.(*StructType); ok && innerSt.Name == st.Name {
				selfRefField = field
				selfRefOffset = field.Offset
				break
			}
		}
	}

	if selfRefField == nil {
		// No self-referential field found, just free normally
		freeVal := g.block.NewValue(OpFree, nil, ptr)
		freeVal.AuxInt = int64(size)
		return
	}

	// Generate a runtime loop (following the while loop pattern):
	// current = ptr (wrapped as nullable)
	// while (current != null) {
	//     unwrapped = unwrap(current)
	//     next = unwrapped.selfRefField
	//     free(unwrapped)
	//     current = next
	// }

	// Use a unique variable name for this loop's current pointer
	loopVarName := fmt.Sprintf("__free_loop_%d", g.fn.NumBlocks())

	loopHeader := g.fn.NewBlock(BlockPlain)
	loopBody := g.fn.NewBlock(BlockPlain)
	loopExit := g.fn.NewBlock(BlockPlain)

	// Use the field type directly (it's already nullable)
	loopVarType := selfRefField.Type

	// Wrap the initial pointer as nullable to match the field type
	irPtrType := &PtrType{Elem: irStruct}
	nullablePtrType := &NullableType{Elem: irPtrType}
	wrappedPtr := g.block.NewValue(OpWrap, nullablePtrType, ptr)

	// Write initial value to loop variable
	g.writeVariable(loopVarName, g.block, wrappedPtr)

	// Jump to header
	g.block.AddSucc(loopHeader)
	g.block = loopHeader

	// Read current pointer (SSA builder will create phi if needed)
	currentPtr := g.readVariable(loopVarName, g.block)

	// Check if null
	isNull := g.block.NewValue(OpIsNull, TypeBool, currentPtr)
	g.block.Kind = BlockIf
	g.block.Control = isNull
	g.block.AddSucc(loopExit)  // null -> exit
	g.block.AddSucc(loopBody)  // not null -> body

	// Seal body block - only predecessor is header
	g.sealBlock(loopBody)

	// Loop body
	g.block = loopBody

	// Unwrap to get actual pointer
	unwrapped := g.block.NewValue(OpUnwrap, irPtrType, currentPtr)

	// Load the next pointer (self-referential field)
	nextFieldPtrType := &PtrType{Elem: loopVarType}
	nextFieldPtr := g.block.NewValue(OpFieldPtr, nextFieldPtrType, unwrapped)
	nextFieldPtr.AuxInt = int64(selfRefOffset)
	nextVal := g.block.NewValue(OpLoad, loopVarType, nextFieldPtr)

	// Free current node
	freeVal := g.block.NewValue(OpFree, nil, unwrapped)
	freeVal.AuxInt = int64(size)

	// Update loop variable for next iteration
	g.writeVariable(loopVarName, g.block, nextVal)

	// Jump back to header
	g.block.AddSucc(loopHeader)

	// Seal header - now all predecessors are known (entry + body)
	g.sealBlock(loopHeader)

	// Seal exit block - only predecessor is header
	g.sealBlock(loopExit)

	// Continue after loop
	g.block = loopExit
}

// getSemanticStructType extracts StructType from a semantic type.
func (g *Generator) getSemanticStructType(t semantic.Type) *semantic.StructType {
	switch ty := t.(type) {
	case *semantic.StructType:
		return ty
	case semantic.StructType:
		return &ty
	default:
		return nil
	}
}

// getOwnedPointerInfo returns the element type and size if t is an owned pointer.
// Returns (nil, 0) if not an owned pointer.
func (g *Generator) getOwnedPointerInfo(t semantic.Type) (semantic.Type, int) {
	switch ty := t.(type) {
	case *semantic.OwnedPointerType:
		return ty.ElementType, g.getElementTypeSize(ty.ElementType)
	case semantic.OwnedPointerType:
		return ty.ElementType, g.getElementTypeSize(ty.ElementType)
	case *semantic.NullableType:
		// Check if inner type is owned pointer
		return g.getOwnedPointerInfo(ty.InnerType)
	case semantic.NullableType:
		return g.getOwnedPointerInfo(ty.InnerType)
	default:
		return nil, 0
	}
}

// getElementTypeSize computes the size of a type, looking up structs by name
// to avoid stale type copies from the two-pass registration in semantic analysis.
func (g *Generator) getElementTypeSize(t semantic.Type) int {
	// For struct types, look up by name to get the version with fields
	if st := g.getSemanticStructType(t); st != nil {
		if irStruct := g.prog.StructByName(st.Name); irStruct != nil {
			return irStruct.Size()
		}
	}
	// Fallback to direct conversion (for non-struct types)
	return g.convertType(t).Size()
}

// isNullableOwnedType checks if a type is a nullable owned pointer.
func isNullableOwnedType(t semantic.Type) bool {
	switch ty := t.(type) {
	case *semantic.NullableType:
		_, isOwned := ty.InnerType.(*semantic.OwnedPointerType)
		if !isOwned {
			_, isOwned = ty.InnerType.(semantic.OwnedPointerType)
		}
		return isOwned
	case semantic.NullableType:
		_, isOwned := ty.InnerType.(*semantic.OwnedPointerType)
		if !isOwned {
			_, isOwned = ty.InnerType.(semantic.OwnedPointerType)
		}
		return isOwned
	default:
		return false
	}
}

// isHeapValueType reports whether a value of this semantic type is heap-
// allocated by its literal expression — struct, class, and array literals
// all emit OpAlloc.
func isHeapValueType(t semantic.Type) bool {
	switch t.(type) {
	case *semantic.StructType, semantic.StructType:
		return true
	case *semantic.ClassType, semantic.ClassType:
		return true
	case *semantic.ArrayType, semantic.ArrayType:
		return true
	}
	return false
}

// nullableValueInner returns the inner semantic type if t is a nullable whose
// element is a plain value type (not an owned pointer or struct reference).
// These nullables allocate a heap slot at wrap time and must be freed at scope
// exit. Returns nil if t is not such a type.
func nullableValueInner(t semantic.Type) semantic.Type {
	var inner semantic.Type
	switch ty := t.(type) {
	case *semantic.NullableType:
		inner = ty.InnerType
	case semantic.NullableType:
		inner = ty.InnerType
	default:
		return nil
	}
	// Owned pointer nullables are handled by isNullableOwnedType.
	if _, ok := inner.(*semantic.OwnedPointerType); ok {
		return nil
	}
	if _, ok := inner.(semantic.OwnedPointerType); ok {
		return nil
	}
	// Struct-reference nullables share a pointer with the struct, no separate
	// heap slot is owned by the variable.
	if _, ok := inner.(*semantic.StructType); ok {
		return nil
	}
	if _, ok := inner.(semantic.StructType); ok {
		return nil
	}
	// Flat value-nullables (integer/bool inner) use the inline tag+payload
	// representation — they own no heap and need no copy/free. This mirrors
	// ir.NullableType.IsFlat; the two must agree. string?/array? remain boxed
	// and fall through to return their inner.
	if isFlatNullableInner(inner) {
		return nil
	}
	return inner
}

// isFlatNullableInner reports whether a nullable with this inner type uses the
// flat tag+payload representation (no heap). Must match ir.NullableType.IsFlat:
// integer and bool inners are flat.
func isFlatNullableInner(inner semantic.Type) bool {
	if semantic.IsIntegerType(inner) {
		return true
	}
	switch inner.(type) {
	case semantic.BooleanType, *semantic.BooleanType:
		return true
	}
	return false
}

// nullableValueAllocSize returns the heap slot size used to wrap a value-type
// nullable, matching the allocation in the ARM64 backend's genWrap.
func (g *Generator) nullableValueAllocSize(inner semantic.Type) int {
	size := g.convertType(inner).Size()
	if size < 8 {
		size = 8
	}
	return size
}

// ============================================================================
// Type Conversion
// ============================================================================

// convertType converts a semantic type to an IR type.
// This is a convenience wrapper around TypeConverter.Convert.
func (g *Generator) convertType(t semantic.Type) Type {
	return g.types().Convert(t)
}

// convertSSAType converts a semantic type to the IR type carried by an SSA
// value of that type: struct/class values are represented as pointers to
// their allocation. Use this for parameter, return, call-result, and
// variable types; use convertType for layout positions (fields, elements).
func (g *Generator) convertSSAType(t semantic.Type) Type {
	return g.types().ConvertSSA(t)
}

// convertStructType converts a semantic struct type to IR.
// This is a convenience wrapper around TypeConverter.convertStruct.
func (g *Generator) convertStructType(st *semantic.StructType) *StructType {
	return g.types().convertStruct(st)
}

// convertClassType converts a semantic class type to IR struct.
// This is a convenience wrapper around TypeConverter.convertClass.
func (g *Generator) convertClassType(ct *semantic.ClassType) *StructType {
	return g.types().convertClass(ct)
}

// getFieldOffset returns the byte offset of a field in a type.
// This is a convenience wrapper around TypeConverter.FieldOffset.
func (g *Generator) getFieldOffset(t semantic.Type, fieldName string) int {
	return g.types().FieldOffset(t, fieldName)
}

// getTypeName returns the name of a type for method mangling.
// This is a convenience wrapper around TypeConverter.TypeName.
func (g *Generator) getTypeName(t semantic.Type) string {
	return g.types().TypeName(t)
}


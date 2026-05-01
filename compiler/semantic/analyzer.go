package semantic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/errors"
)

// maxFunctionParameters is the maximum number of parameters a function can have.
// This limit exists because the ARM64 calling convention only supports passing
// the first 8 arguments in registers (x0-x7).
const maxFunctionParameters = 8

// toErrorPos converts an ast.Position to an errors.Position
func toErrorPos(p ast.Position) errors.Position {
	return errors.Position{Line: p.Line, Column: p.Column, Offset: p.Offset}
}

// VariableInfo holds information about a declared variable
type VariableInfo struct {
	Type    Type
	Mutable bool
}

// Scope represents a lexical scope for variable lookup
type Scope struct {
	parent    *Scope
	variables map[string]VariableInfo
}

// newScope creates a new scope with an optional parent
func newScope(parent *Scope) *Scope {
	return &Scope{
		parent:    parent,
		variables: make(map[string]VariableInfo),
	}
}

// declare adds a variable to the current scope
// Returns false if the variable is already declared in this scope
func (s *Scope) declare(name string, typ Type, mutable bool) bool {
	if _, exists := s.variables[name]; exists {
		return false
	}
	s.variables[name] = VariableInfo{Type: typ, Mutable: mutable}
	return true
}

// lookup finds a variable in this scope or any parent scope
// Returns the variable info and true if found, or empty info and false if not found
func (s *Scope) lookup(name string) (VariableInfo, bool) {
	if info, exists := s.variables[name]; exists {
		return info, true
	}
	if s.parent != nil {
		return s.parent.lookup(name)
	}
	return VariableInfo{}, false
}

// FunctionInfo holds information about a declared function
type FunctionInfo struct {
	ParamTypes []Type
	ReturnType Type
}

// Analyzer performs semantic analysis on the AST
type Analyzer struct {
	filename          string
	errors            []*errors.CompilerError
	currentScope      *Scope
	ownershipScope    *OwnershipScope         // tracks ownership state for move semantics
	functions         map[string]FunctionInfo // function registry
	TypeRegistry      *TypeRegistry           // centralized struct/class/object registry
	currentReturnType  Type                    // return type of current function being analyzed
	currentFunctionName string                 // name of current function being analyzed
	loopDepth         int                     // tracks nested loop depth for break/continue validation
	currentClass      *ClassType              // class being analyzed (for 'self' validation)
	requireMain       bool                    // whether to require a 'main' function (true for root package)
	packagePath       string                  // package path for type registration ("main" for root)
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(filename string) *Analyzer {
	return &Analyzer{
		filename:          filename,
		errors:            make([]*errors.CompilerError, 0),
		currentScope:      newScope(nil),           // global scope
		ownershipScope:    newOwnershipScope(nil),  // global ownership scope
		functions:         make(map[string]FunctionInfo),
		TypeRegistry:      NewTypeRegistry(),
		currentReturnType: nil,
		currentClass:      nil,
		requireMain:       true,                    // default: require main (root package)
		packagePath:       "main",                  // default: root package
	}
}

// enterScope creates a new nested scope
func (a *Analyzer) enterScope() {
	a.currentScope = newScope(a.currentScope)
	a.ownershipScope = newOwnershipScope(a.ownershipScope)
}

// exitScope returns to the parent scope
func (a *Analyzer) exitScope() {
	if a.currentScope.parent != nil {
		a.currentScope = a.currentScope.parent
	}
	if a.ownershipScope.parent != nil {
		a.ownershipScope = a.ownershipScope.parent
	}
}

// moveRecord tracks a variable that was moved in a branch
type moveRecord struct {
	name     string
	moveInfo MoveInfo
}

// snapshotOwnershipState captures the current ownership state of all tracked variables
func (a *Analyzer) snapshotOwnershipState() map[string]OwnershipInfo {
	snapshot := make(map[string]OwnershipInfo)
	for scope := a.ownershipScope; scope != nil; scope = scope.parent {
		for name, info := range scope.ownership {
			// Don't overwrite - inner scope takes precedence
			if _, exists := snapshot[name]; !exists {
				snapshot[name] = info
			}
		}
	}
	return snapshot
}

// collectBranchMoves returns variables that were moved since the snapshot was taken
func (a *Analyzer) collectBranchMoves(beforeSnapshot map[string]OwnershipInfo) []moveRecord {
	var moves []moveRecord

	// Check all scopes for moved variables
	for scope := a.ownershipScope; scope != nil; scope = scope.parent {
		for name, info := range scope.ownership {
			if info.State == StateMoved {
				// Was it owned before?
				if before, existed := beforeSnapshot[name]; existed && before.State == StateOwned {
					moves = append(moves, moveRecord{
						name:     name,
						moveInfo: info.MoveInfo,
					})
				}
			}
		}
	}

	return moves
}

// mergeConditionalMoves marks variables as moved if they were moved in any branch
func (a *Analyzer) mergeConditionalMoves(thenMoves, elseMoves []moveRecord) {
	// Combine all moves from both branches
	allMoves := make(map[string]MoveInfo)
	for _, m := range thenMoves {
		allMoves[m.name] = m.moveInfo
	}
	for _, m := range elseMoves {
		// If already moved in then branch, keep that info
		if _, exists := allMoves[m.name]; !exists {
			allMoves[m.name] = m.moveInfo
		}
	}

	// Mark each variable as moved in the current ownership scope
	for name, moveInfo := range allMoves {
		a.ownershipScope.markMoved(name, moveInfo.MovedTo, moveInfo.Location)
	}
}

// AnalyzePackage performs semantic analysis on a package consisting of one or more files.
// This is the primary entry point for the module system.
// isRoot indicates whether this is the root package (which must contain a main function).
// deps is a map of import path -> PackageNamespace for already-analyzed dependencies (nil for no deps).
func (a *Analyzer) AnalyzePackage(files []*ast.FileAST, pkgPath string, isRoot bool, deps map[string]*PackageNamespace) ([]*errors.CompilerError, *TypedProgram) {
	a.requireMain = isRoot
	a.packagePath = pkgPath
	// Merge all files' declarations into a single program
	merged := &ast.Program{
		Declarations: []ast.Declaration{},
		Statements:   []ast.Statement{},
	}

	for _, f := range files {
		merged.Declarations = append(merged.Declarations, f.AST.Declarations...)
		merged.Statements = append(merged.Statements, f.AST.Statements...)
		merged.Imports = append(merged.Imports, f.AST.Imports...)
	}

	if len(files) > 0 {
		merged.StartPos = files[0].AST.StartPos
		merged.EndPos = files[len(files)-1].AST.EndPos
	}

	// Bind import namespaces in scope, checking for duplicate import names
	seenImportNames := make(map[string]string) // name -> import path
	if deps != nil {
		for _, imp := range merged.Imports {
			if prevPath, exists := seenImportNames[imp.Name]; exists {
				a.addError(
					fmt.Sprintf("import name '%s' is already used by import \"%s\"", imp.Name, prevPath),
					imp.ImportPos, imp.ImportPos,
				).WithHint("use explicit import form to alias one of them")
			}
			seenImportNames[imp.Name] = imp.Path
			if ns, ok := deps[imp.Path]; ok {
				a.currentScope.variables[imp.Name] = VariableInfo{
					Type:    PackageNamespaceType{Namespace: ns},
					Mutable: false,
				}
			}
		}
	}

	// Check for conflicts between import names and declaration names
	importNames := make(map[string]ast.Position)
	for _, imp := range merged.Imports {
		importNames[imp.Name] = imp.ImportPos
	}
	for _, decl := range merged.Declarations {
		var declName string
		var declPos ast.Position
		switch d := decl.(type) {
		case *ast.FunctionDecl:
			declName = d.Name
			declPos = d.NamePos
		case *ast.StructDecl:
			declName = d.Name
			declPos = d.NamePos
		case *ast.ClassDecl:
			declName = d.Name
			declPos = d.NamePos
		case *ast.ObjectDecl:
			declName = d.Name
			declPos = d.NamePos
		}
		if declName != "" {
			if impPos, conflict := importNames[declName]; conflict {
				_ = impPos // import position available if needed for better error
				a.addError(
					fmt.Sprintf("declaration '%s' conflicts with import of the same name", declName),
					declPos, declPos,
				)
			}
		}
	}

	return a.Analyze(merged)
}

// Analyze performs semantic analysis on a program
func (a *Analyzer) Analyze(program *ast.Program) ([]*errors.CompilerError, *TypedProgram) {
	typedProgram := &TypedProgram{
		Declarations: make([]TypedDeclaration, 0),
		Statements:   make([]TypedStatement, 0),
		StartPos:     program.StartPos,
		EndPos:       program.EndPos,
	}

	// Pass 1: register all type names (so forward references work)
	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.StructDecl:
			a.registerStructName(d)
		case *ast.ClassDecl:
			a.registerClassName(d)
		case *ast.ObjectDecl:
			a.registerObjectName(d)
		}
	}

	// Pass 2: resolve field/method types (now all type names are known)
	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.StructDecl:
			a.resolveStructFields(d)
		case *ast.ClassDecl:
			a.resolveClassFieldsAndMethods(d)
		case *ast.ObjectDecl:
			a.resolveObjectMethods(d)
		}
	}

	// Pass 3: collect all function signatures
	hasMain := false
	for _, decl := range program.Declarations {
		if fnDecl, ok := decl.(*ast.FunctionDecl); ok {
			a.registerFunction(fnDecl)
			if fnDecl.Name == "main" {
				hasMain = true
			}
		}
	}

	if !hasMain && a.requireMain && len(program.Declarations) > 0 {
		a.addError("program must have a 'main' function", program.EndPos, program.EndPos)
	}

	// Check for circular initialization dependencies among top-level variables
	a.checkCircularInit(program.Statements)

	// Pass 4: analyze top-level variable declarations (val/var)
	for _, stmt := range program.Statements {
		typedStmt := a.analyzeStatement(stmt)
		typedProgram.Statements = append(typedProgram.Statements, typedStmt)
	}

	// Pass 5: analyze all declarations (function bodies, class methods, etc.)
	for _, decl := range program.Declarations {
		typedDecl := a.analyzeDeclaration(decl)
		typedProgram.Declarations = append(typedProgram.Declarations, typedDecl)
	}

	return a.errors, typedProgram
}

// hasTopLevelVarDecls checks if any statements are variable declarations.
// checkCircularInit detects circular dependencies among top-level variable initializers.
// e.g., val x = y + 1 and val y = x + 1 is a cycle.
func (a *Analyzer) checkCircularInit(stmts []ast.Statement) {
	// Build dependency graph: variable name -> set of variable names referenced in initializer
	varNames := make(map[string]bool)
	deps := make(map[string][]string)

	for _, stmt := range stmts {
		varDecl, ok := stmt.(*ast.VarDeclStmt)
		if !ok {
			continue
		}
		varNames[varDecl.Name] = true
		refs := collectIdentifierRefs(varDecl.Initializer)
		for _, ref := range refs {
			deps[varDecl.Name] = append(deps[varDecl.Name], ref)
		}
	}

	// DFS cycle detection (only consider references to other top-level variables)
	const (
		white = 0
		gray  = 1
		black = 2
	)
	colors := make(map[string]int)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		colors[node] = gray
		for _, dep := range deps[node] {
			if !varNames[dep] {
				continue // not a top-level variable, skip
			}
			if colors[dep] == gray {
				a.addError(
					fmt.Sprintf("circular initialization dependency: '%s' and '%s' depend on each other", node, dep),
					ast.Position{}, ast.Position{},
				)
				return true
			}
			if colors[dep] == white {
				if dfs(dep) {
					return true
				}
			}
		}
		colors[node] = black
		return false
	}

	for name := range varNames {
		if colors[name] == white {
			dfs(name)
		}
	}
}

// collectIdentifierRefs extracts all identifier names referenced in an expression tree.
func collectIdentifierRefs(expr ast.Expression) []string {
	if expr == nil {
		return nil
	}
	var refs []string
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		refs = append(refs, e.Name)
	case *ast.BinaryExpr:
		refs = append(refs, collectIdentifierRefs(e.Left)...)
		refs = append(refs, collectIdentifierRefs(e.Right)...)
	case *ast.UnaryExpr:
		refs = append(refs, collectIdentifierRefs(e.Operand)...)
	case *ast.CallExpr:
		for _, arg := range e.Arguments {
			refs = append(refs, collectIdentifierRefs(arg)...)
		}
	case *ast.GroupingExpr:
		refs = append(refs, collectIdentifierRefs(e.Expr)...)
	case *ast.FieldAccessExpr:
		refs = append(refs, collectIdentifierRefs(e.Object)...)
	case *ast.MethodCallExpr:
		refs = append(refs, collectIdentifierRefs(e.Object)...)
		for _, arg := range e.Arguments {
			refs = append(refs, collectIdentifierRefs(arg)...)
		}
	case *ast.IndexExpr:
		refs = append(refs, collectIdentifierRefs(e.Array)...)
		refs = append(refs, collectIdentifierRefs(e.Index)...)
	case *ast.NewExpr:
		refs = append(refs, collectIdentifierRefs(e.Operand)...)
	}
	return refs
}

// registerFunction registers a function's signature in the function registry
func (a *Analyzer) registerFunction(fn *ast.FunctionDecl) {
	// Check for duplicate function
	if _, exists := a.functions[fn.Name]; exists {
		a.addError(fmt.Sprintf("function '%s' is already declared", fn.Name), fn.NamePos, fn.NamePos)
		return
	}

	// Check for too many parameters
	if len(fn.Parameters) > maxFunctionParameters {
		a.addError(
			fmt.Sprintf("function '%s' has %d parameters, maximum allowed is %d",
				fn.Name, len(fn.Parameters), maxFunctionParameters),
			fn.NamePos, fn.NamePos,
		).WithHint("consider passing a struct or reducing the number of parameters")
	}

	// Convert parameter types (supports both primitive and struct types)
	paramTypes := make([]Type, len(fn.Parameters))
	for i, param := range fn.Parameters {
		paramType := a.resolveTypeName(param.TypeName, param.TypePos)

		// var modifier on parameters is no longer supported
		// Use MutRef<T> instead of var &T
		if param.Mutable {
			if IsRefPointer(paramType) {
				a.addError("use &&T instead of 'var &T'", param.VarPos, param.TypePos)
			} else {
				a.addError("'var' modifier is not supported on parameters; use &&T for mutable references", param.VarPos, param.VarPos)
			}
		}

		paramTypes[i] = paramType
	}

	// Convert return type (supports both primitive and struct types)
	returnType := a.resolveTypeName(fn.ReturnType, fn.ReturnPos)

	// References cannot be used as return type
	if IsAnyRefPointer(returnType) {
		a.addError("references cannot be used as return types; use *T instead", fn.ReturnPos, fn.ReturnPos)
	}

	a.functions[fn.Name] = FunctionInfo{
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}
}

// registerStructName registers only the struct name (first pass for forward references)
func (a *Analyzer) registerStructName(s *ast.StructDecl) {
	// Check for duplicate struct
	if kind, exists := a.TypeRegistry.NameExists(s.Name); exists {
		if kind == TypeKindStruct {
			a.addError(fmt.Sprintf("struct '%s' is already declared", s.Name), s.NamePos, s.NamePos)
		} else {
			a.addError(fmt.Sprintf("type '%s' is already declared as %s", s.Name, kind), s.NamePos, s.NamePos)
		}
		return
	}

	// Register the struct with empty fields (will be resolved in second pass)
	a.TypeRegistry.RegisterStruct(s.Name, StructType{
		Name:        s.Name,
		PackagePath: a.packagePath,
		Fields:      nil, // Placeholder until resolveStructFields is called
	})
}

// resolveStructFields resolves field types for a registered struct (second pass)
func (a *Analyzer) resolveStructFields(s *ast.StructDecl) {
	// Convert field types (now all struct names are known, so forward references work)
	fields := make([]StructFieldInfo, len(s.Fields))
	for i, field := range s.Fields {
		fieldType := a.resolveTypeName(field.TypeName, field.TypePos)

		// Ref<T> cannot be used as struct field type
		if IsRefPointer(fieldType) {
			a.addError("&T cannot be used as a struct field type; use *T instead", field.TypePos, field.TypePos)
		}

		fields[i] = StructFieldInfo{
			Name:    field.Name,
			Type:    fieldType,
			Mutable: field.Mutable,
			Index:   i,
		}
	}

	// Update the struct with resolved fields (preserve PackagePath from registration)
	a.TypeRegistry.UpdateStruct(s.Name, StructType{
		Name:        s.Name,
		PackagePath: a.packagePath,
		Fields:      fields,
	})
}

// registerClassName registers only the class name (first pass for forward references)
func (a *Analyzer) registerClassName(c *ast.ClassDecl) {
	// Check for duplicate type name
	if kind, exists := a.TypeRegistry.NameExists(c.Name); exists {
		if kind == TypeKindClass {
			a.addError(fmt.Sprintf("class '%s' is already declared", c.Name), c.NamePos, c.NamePos)
		} else {
			a.addError(fmt.Sprintf("type '%s' is already declared as %s", c.Name, kind), c.NamePos, c.NamePos)
		}
		return
	}

	// Register the class with empty fields/methods (will be resolved in second pass)
	a.TypeRegistry.RegisterClass(c.Name, ClassType{
		Name:        c.Name,
		PackagePath: a.packagePath,
		Fields:      nil,                             // Placeholder until resolveClassFields is called
		Methods:     make(map[string][]*MethodInfo),  // Placeholder until resolveClassMethods is called
	})
}

// registerObjectName registers only the object name (first pass for forward references)
func (a *Analyzer) registerObjectName(o *ast.ObjectDecl) {
	// Check for duplicate type name
	if kind, exists := a.TypeRegistry.NameExists(o.Name); exists {
		if kind == TypeKindObject {
			a.addError(fmt.Sprintf("object '%s' is already declared", o.Name), o.NamePos, o.NamePos)
		} else {
			a.addError(fmt.Sprintf("type '%s' is already declared as %s", o.Name, kind), o.NamePos, o.NamePos)
		}
		return
	}

	// Register the object with empty methods (will be resolved in second pass)
	a.TypeRegistry.RegisterObject(o.Name, ObjectType{
		Name:        o.Name,
		PackagePath: a.packagePath,
		Methods:     make(map[string][]*MethodInfo),  // Placeholder until resolveObjectMethods is called
	})
}

// resolveClassFieldsAndMethods resolves field and method types for a registered class (second pass)
func (a *Analyzer) resolveClassFieldsAndMethods(c *ast.ClassDecl) {
	classType, exists := a.TypeRegistry.LookupClass(c.Name)
	if !exists {
		return // class was not registered (error already reported)
	}

	// Resolve fields (same as struct fields)
	fields := make([]StructFieldInfo, len(c.Fields))
	for i, field := range c.Fields {
		fieldType := a.resolveTypeName(field.TypeName, field.TypePos)

		// Ref<T> cannot be used as class field type
		if IsRefPointer(fieldType) {
			a.addError("&T cannot be used as a class field type; use *T instead", field.TypePos, field.TypePos)
		}

		fields[i] = StructFieldInfo{
			Name:    field.Name,
			Type:    fieldType,
			Mutable: field.Mutable,
			Index:   i,
		}
	}
	classType.Fields = fields

	// Update the class in the registry BEFORE resolving methods
	// This ensures that when method parameters reference the class (e.g., &Counter),
	// the lookup will find the ClassType with fields already populated
	a.TypeRegistry.UpdateClass(c.Name, classType)

	// Resolve methods
	for _, method := range c.Methods {
		methodInfo := a.resolveMethodDecl(c.Name, &classType, &method)
		if methodInfo != nil {
			// Check for duplicate method signature
			if existingMethods, ok := classType.Methods[method.Name]; ok {
				if duplicate := findDuplicateSignature(existingMethods, methodInfo); duplicate != nil {
					a.addError(
						fmt.Sprintf("duplicate method signature: '%s' already has an overload with parameters (%s)",
							method.Name, formatParamTypes(methodInfo.ParamTypes)),
						method.NamePos, method.NamePos)
					continue
				}
			}
			classType.Methods[method.Name] = append(classType.Methods[method.Name], methodInfo)
		}
	}

	// Update the class again to include resolved methods
	a.TypeRegistry.UpdateClass(c.Name, classType)
}

// resolveObjectMethods resolves method types for a registered object (second pass)
func (a *Analyzer) resolveObjectMethods(o *ast.ObjectDecl) {
	objectType, exists := a.TypeRegistry.LookupObject(o.Name)
	if !exists {
		return // object was not registered (error already reported)
	}

	// Resolve methods (all must be static)
	for _, method := range o.Methods {
		methodInfo := a.resolveObjectMethodDecl(o.Name, &method)
		if methodInfo != nil {
			// Check for duplicate method signature
			if existingMethods, ok := objectType.Methods[method.Name]; ok {
				if duplicate := findDuplicateSignature(existingMethods, methodInfo); duplicate != nil {
					a.addError(
						fmt.Sprintf("duplicate method signature: '%s' already has an overload with parameters (%s)",
							method.Name, formatParamTypes(methodInfo.ParamTypes)),
						method.NamePos, method.NamePos)
					continue
				}
			}
			objectType.Methods[method.Name] = append(objectType.Methods[method.Name], methodInfo)
		}
	}

	// Update the object in the registry
	a.TypeRegistry.UpdateObject(o.Name, objectType)
}

// methodOwnerKind distinguishes between class and object method contexts
type methodOwnerKind int

const (
	methodOwnerClass  methodOwnerKind = iota // class method (can have 'self')
	methodOwnerObject                        // object method (always static)
)

// resolveMethodDecl resolves a class method declaration and returns MethodInfo
func (a *Analyzer) resolveMethodDecl(className string, classType *ClassType, method *ast.MethodDecl) *MethodInfo {
	return a.resolveMethodDeclCore(className, method, methodOwnerClass)
}

// resolveObjectMethodDecl resolves an object method declaration and returns MethodInfo
func (a *Analyzer) resolveObjectMethodDecl(objectName string, method *ast.MethodDecl) *MethodInfo {
	return a.resolveMethodDeclCore(objectName, method, methodOwnerObject)
}

// resolveMethodDeclCore is the shared implementation for resolving method declarations.
// ownerKind determines whether 'self' parameters are allowed (class) or forbidden (object).
func (a *Analyzer) resolveMethodDeclCore(ownerName string, method *ast.MethodDecl, ownerKind methodOwnerKind) *MethodInfo {
	paramTypes := make([]Type, len(method.Parameters))
	paramNames := make([]string, len(method.Parameters))

	// Check if this is an instance method (first param is 'self')
	// Only meaningful for class methods; objects don't support instance methods
	isInstance := ownerKind == methodOwnerClass &&
		len(method.Parameters) > 0 &&
		method.Parameters[0].Name == "self"

	for i, param := range method.Parameters {
		paramNames[i] = param.Name
		paramType := a.resolveTypeName(param.TypeName, param.TypePos)

		// Handle 'self' parameter based on owner kind
		if param.Name == "self" {
			if ownerKind == methodOwnerObject {
				// Object methods cannot have 'self'
				a.addError("object methods cannot have 'self' parameter; objects only support static methods", param.NamePos, param.NamePos)
				paramTypes[i] = TypeError
				continue
			} else if i == 0 {
				// Class method: validate self type
				if !a.validateSelfType(ownerName, paramType, param.TypePos) {
					paramType = TypeError
				}
			} else {
				// 'self' must be the first parameter
				a.addError("'self' must be the first parameter", param.NamePos, param.NamePos)
			}
		}

		paramTypes[i] = paramType
	}

	// Resolve return type
	var returnType Type = TypeVoid
	if method.ReturnType != "" {
		returnType = a.resolveTypeName(method.ReturnType, method.ReturnPos)
	}

	// References cannot be used as return type
	if IsAnyRefPointer(returnType) {
		a.addError("references cannot be used as return types; use *T instead", method.ReturnPos, method.ReturnPos)
	}

	return &MethodInfo{
		Name:       method.Name,
		ParamTypes: paramTypes,
		ParamNames: paramNames,
		ReturnType: returnType,
		IsStatic:   !isInstance,
	}
}

// validateSelfType validates that the self parameter type is correct for the class
func (a *Analyzer) validateSelfType(className string, selfType Type, pos ast.Position) bool {
	// Self type must be a pointer type: &ClassName, &&ClassName, or *ClassName
	var elementType Type

	switch t := selfType.(type) {
	case RefPointerType:
		elementType = t.ElementType
	case MutRefPointerType:
		elementType = t.ElementType
	case OwnedPointerType:
		elementType = t.ElementType
	default:
		a.addError(fmt.Sprintf("'self' must have a pointer type (&%s, &&%s, or *%s)", className, className, className), pos, pos)
		return false
	}

	// Check that the element type is the enclosing class
	if ct, ok := elementType.(ClassType); ok {
		if ct.Name != className {
			a.addError(fmt.Sprintf("'self' type must reference the enclosing class '%s', not '%s'", className, ct.Name), pos, pos)
			return false
		}
		return true
	}

	// At this point, element type should be looked up by name since ClassType might not be registered yet
	// In the second pass, we might have the class registered but not fully resolved
	// For now, check if it's the class name string
	if elementType == nil {
		a.addError(fmt.Sprintf("'self' type must reference the enclosing class '%s'", className), pos, pos)
		return false
	}

	// The element type string should match the class name
	if elementType.String() != className {
		a.addError(fmt.Sprintf("'self' type must reference the enclosing class '%s', not '%s'", className, elementType.String()), pos, pos)
		return false
	}

	return true
}

// resolveTypeName converts a type name string to a Type, checking both primitive types and structs.
// Reports errors for unknown types or invalid type constructs.
func (a *Analyzer) resolveTypeName(name string, pos ast.Position) Type {
	return a.resolveTypeNameCore(name, pos, true)
}

// resolveTypeNameNoError converts a type name string to a Type without adding errors
// (used when caller wants to handle errors itself)
func (a *Analyzer) resolveTypeNameNoError(name string) Type {
	return a.resolveTypeNameCore(name, ast.Position{}, false)
}

// resolveTypeNameCore is the shared implementation for type name resolution.
// If reportErrors is true, errors are reported at the given position.
func (a *Analyzer) resolveTypeNameCore(name string, pos ast.Position, reportErrors bool) Type {
	// Helper to optionally report an error
	maybeError := func(msg string) {
		if reportErrors {
			a.addError(msg, pos, pos)
		}
	}

	// Check for nullable type T?
	if strings.HasSuffix(name, "?") {
		innerName := name[:len(name)-1]
		innerType := a.resolveTypeNameCore(innerName, pos, reportErrors)
		if _, isErr := innerType.(ErrorType); isErr {
			return TypeError
		}
		// Check for nested nullables (T??) - defense-in-depth
		// The parser already catches this at parse time, but this check remains
		// for programmatically constructed ASTs (e.g., in tests)
		if IsNullable(innerType) {
			maybeError("nested nullable types are not allowed")
			return TypeError
		}
		return NullableType{InnerType: innerType}
	}

	// Check for T[] array syntax, but NOT when there's a borrow prefix.
	// &s64[] should parse as &(s64[]) = RefPointer{ArrayType}, not Array{RefPointer{s64}}.
	// *Point[] stays as ArrayType{OwnedPointerType{Point}} (array of owned pointers).
	if strings.HasSuffix(name, "[]") && !strings.HasPrefix(name, "&") {
		elementTypeName := name[:len(name)-2] // extract T from T[]
		elementType := a.resolveTypeNameCore(elementTypeName, pos, reportErrors)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		return ArrayType{ElementType: elementType, Size: ArraySizeUnknown}
	}

	// Check for symbol-based pointer syntax: *T, &&T, &T
	// IMPORTANT: Check "&&" before "&" to avoid wrong prefix match

	// Check for &&T syntax (mutable borrow)
	if strings.HasPrefix(name, "&&") {
		elementTypeName := strings.TrimPrefix(name, "&&")
		elementType := a.resolveTypeNameCore(elementTypeName, pos, reportErrors)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// &&T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			maybeError("&&T cannot contain nullable type; use &&T? for nullable mutable references")
			return TypeError
		}
		return MutRefPointerType{ElementType: elementType}
	}

	// Check for &T syntax (immutable borrow)
	if strings.HasPrefix(name, "&") {
		elementTypeName := strings.TrimPrefix(name, "&")
		elementType := a.resolveTypeNameCore(elementTypeName, pos, reportErrors)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// &T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			maybeError("&T cannot contain nullable type; use &T? for nullable references")
			return TypeError
		}
		return RefPointerType{ElementType: elementType}
	}

	// Check for *T syntax (owned pointer)
	if strings.HasPrefix(name, "*") {
		elementTypeName := strings.TrimPrefix(name, "*")
		elementType := a.resolveTypeNameCore(elementTypeName, pos, reportErrors)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// *T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			maybeError("*T cannot contain nullable type; use *T? for nullable owned pointers")
			return TypeError
		}
		return OwnedPointerType{ElementType: elementType}
	}

	// Try primitive types first
	t := TypeFromName(name)
	if _, isErr := t.(ErrorType); !isErr {
		return t
	}

	// Check for qualified type name: pkg.Type (e.g., geometry.Point)
	if dotIdx := strings.IndexByte(name, '.'); dotIdx != -1 {
		pkgAlias := name[:dotIdx]
		typeName := name[dotIdx+1:]

		varInfo, found := a.currentScope.lookup(pkgAlias)
		if !found {
			maybeError(fmt.Sprintf("undefined package '%s'", pkgAlias))
			return TypeError
		}
		nsType, isNs := varInfo.Type.(PackageNamespaceType)
		if !isNs {
			maybeError(fmt.Sprintf("'%s' is not a package", pkgAlias))
			return TypeError
		}
		export, exportFound := nsType.Namespace.Exports[typeName]
		if !exportFound {
			maybeError(fmt.Sprintf("package '%s' has no type '%s'", nsType.Namespace.Path, typeName))
			return TypeError
		}
		return export.Type
	}

	// Try user-defined types (struct, class, object)
	if userType, ok := a.TypeRegistry.Lookup(name); ok {
		return userType
	}

	// Unknown type
	maybeError(fmt.Sprintf("unknown type '%s'", name))
	return TypeError
}

// analyzeDeclaration performs semantic analysis on a declaration
func (a *Analyzer) analyzeDeclaration(decl ast.Declaration) TypedDeclaration {
	switch d := decl.(type) {
	case *ast.FunctionDecl:
		return a.analyzeFunctionDecl(d)
	case *ast.StructDecl:
		return a.analyzeStructDecl(d)
	case *ast.ClassDecl:
		return a.analyzeClassDecl(d)
	case *ast.ObjectDecl:
		return a.analyzeObjectDecl(d)
	default:
		a.addError("unknown declaration type", decl.Pos(), decl.End())
		return &TypedFunctionDecl{
			Name:       "error",
			NamePos:    decl.Pos(),
			EqualsPos:  decl.Pos(),
			LeftParen:  decl.Pos(),
			RightParen: decl.Pos(),
			Body: &TypedBlockStmt{
				LeftBrace:  decl.Pos(),
				Statements: []TypedStatement{},
				RightBrace: decl.End(),
			},
		}
	}
}

// analyzeStructDecl analyzes a struct declaration
func (a *Analyzer) analyzeStructDecl(s *ast.StructDecl) TypedDeclaration {
	// The struct type was already registered in the first pass
	structType, _ := a.TypeRegistry.LookupStruct(s.Name)

	return &TypedStructDecl{
		Name:          s.Name,
		NamePos:       s.NamePos,
		EqualsPos:     s.EqualsPos,
		StructKeyword: s.StructKeyword,
		LeftBrace:     s.LeftBrace,
		StructType:    structType,
		RightBrace:    s.RightBrace,
	}
}

// analyzeClassDecl analyzes a class declaration
func (a *Analyzer) analyzeClassDecl(c *ast.ClassDecl) TypedDeclaration {
	// The class type was already registered in the first pass
	classType, _ := a.TypeRegistry.LookupClass(c.Name)

	// Analyze method bodies
	typedMethods := make([]*TypedMethodDecl, 0, len(c.Methods))
	for _, method := range c.Methods {
		typedMethod := a.analyzeMethodDecl(&classType, &method)
		typedMethods = append(typedMethods, typedMethod)
	}

	return &TypedClassDecl{
		Name:         c.Name,
		NamePos:      c.NamePos,
		EqualsPos:    c.EqualsPos,
		ClassKeyword: c.ClassKeyword,
		LeftBrace:    c.LeftBrace,
		ClassType:    classType,
		Methods:      typedMethods,
		RightBrace:   c.RightBrace,
	}
}

// analyzeObjectDecl analyzes a singleton object declaration
func (a *Analyzer) analyzeObjectDecl(o *ast.ObjectDecl) TypedDeclaration {
	// The object type was already registered in the first pass
	objectType, _ := a.TypeRegistry.LookupObject(o.Name)

	// Analyze method bodies (all methods are static)
	typedMethods := make([]*TypedMethodDecl, 0, len(o.Methods))
	for _, method := range o.Methods {
		typedMethod := a.analyzeObjectMethodDecl(&objectType, &method)
		typedMethods = append(typedMethods, typedMethod)
	}

	return &TypedObjectDecl{
		Name:          o.Name,
		NamePos:       o.NamePos,
		EqualsPos:     o.EqualsPos,
		ObjectKeyword: o.ObjectKeyword,
		LeftBrace:     o.LeftBrace,
		ObjectType:    objectType,
		Methods:       typedMethods,
		RightBrace:    o.RightBrace,
	}
}

// findMatchingMethodInfo finds the MethodInfo that matches the given method declaration.
// This is used for overloaded methods where multiple MethodInfos exist with the same name.
// We match by parameter count and type name strings for precise matching.
func findMatchingMethodInfo(methodInfos []*MethodInfo, method *ast.MethodDecl) *MethodInfo {
	for _, mi := range methodInfos {
		// Check if parameter count matches
		if len(mi.ParamTypes) != len(method.Parameters) {
			continue
		}
		// Check if parameter type names match
		match := true
		for i, param := range method.Parameters {
			// Compare the resolved type's string representation with the AST type name
			// This handles cases like "i64" vs "i64?"
			typeStr := mi.ParamTypes[i].String()
			if typeStr != param.TypeName {
				match = false
				break
			}
		}
		if match {
			return mi
		}
	}
	// Fallback: return first if no exact match (shouldn't happen normally)
	if len(methodInfos) > 0 {
		return methodInfos[0]
	}
	return nil
}

// findDuplicateSignature checks if newMethod has the same signature as any existing method.
// Returns the duplicate if found, nil otherwise.
func findDuplicateSignature(existingMethods []*MethodInfo, newMethod *MethodInfo) *MethodInfo {
	for _, existing := range existingMethods {
		if methodSignaturesEqual(existing, newMethod) {
			return existing
		}
	}
	return nil
}

// methodSignaturesEqual checks if two methods have the same signature.
// Signatures are equal if they have the same number of parameters with identical types.
func methodSignaturesEqual(a, b *MethodInfo) bool {
	if len(a.ParamTypes) != len(b.ParamTypes) {
		return false
	}
	for i := range a.ParamTypes {
		if !a.ParamTypes[i].Equals(b.ParamTypes[i]) {
			return false
		}
	}
	return true
}

// formatParamTypes formats a list of parameter types for error messages.
func formatParamTypes(types []Type) string {
	if len(types) == 0 {
		return ""
	}
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = t.String()
	}
	return strings.Join(strs, ", ")
}

// analyzeMethodDecl analyzes a class method body and returns a typed method declaration
func (a *Analyzer) analyzeMethodDecl(classType *ClassType, method *ast.MethodDecl) *TypedMethodDecl {
	return a.analyzeMethodDeclCore(classType, nil, method)
}

// analyzeObjectMethodDecl analyzes an object method body and returns a typed method declaration
func (a *Analyzer) analyzeObjectMethodDecl(objectType *ObjectType, method *ast.MethodDecl) *TypedMethodDecl {
	return a.analyzeMethodDeclCore(nil, objectType, method)
}

// analyzeMethodDeclCore is the shared implementation for analyzing class and object method bodies.
// Pass classType for class methods, or objectType for object methods (exactly one must be non-nil).
func (a *Analyzer) analyzeMethodDeclCore(classType *ClassType, objectType *ObjectType, method *ast.MethodDecl) *TypedMethodDecl {
	// Determine method owner kind and get method map
	var methodMap map[string][]*MethodInfo
	isClassMethod := classType != nil

	if isClassMethod {
		// Set current class for 'self' validation
		a.currentClass = classType
		methodMap = classType.Methods
	} else {
		methodMap = objectType.Methods
	}

	// Save state for restoration
	prevClass := a.currentClass
	if !isClassMethod {
		a.currentClass = nil
	}

	// Check if this is an instance method (only applies to class methods)
	isInstance := isClassMethod &&
		len(method.Parameters) > 0 &&
		method.Parameters[0].Name == "self"

	// Get method info - find the matching overload
	methodInfos, ok := methodMap[method.Name]
	var methodInfo *MethodInfo
	if ok && len(methodInfos) > 0 {
		if isClassMethod {
			// Class methods may be overloaded - find matching one
			methodInfo = findMatchingMethodInfo(methodInfos, method)
		} else {
			// Object methods - just use first (overloading still supported)
			methodInfo = findMatchingMethodInfo(methodInfos, method)
		}
	}

	// Enter a new scope for the method body
	a.enterScope()

	// Add parameters to scope
	typedParams := make([]TypedParameter, len(method.Parameters))
	for i, param := range method.Parameters {
		var paramType Type = TypeError
		if methodInfo != nil && i < len(methodInfo.ParamTypes) {
			paramType = methodInfo.ParamTypes[i]
		}

		typedParams[i] = TypedParameter{
			Name:    param.Name,
			NamePos: param.NamePos,
			Colon:   param.Colon,
			Type:    paramType,
			TypePos: param.TypePos,
		}

		// Add parameter to scope
		// For class methods, 'self' with &&T allows mutation
		isMutable := param.Mutable
		if isClassMethod && param.Name == "self" && IsMutRefPointer(paramType) {
			isMutable = true
		}
		a.currentScope.declare(param.Name, paramType, isMutable)
	}

	// Set return type for return statement checking
	prevReturnType := a.currentReturnType
	if methodInfo != nil {
		a.currentReturnType = methodInfo.ReturnType
	} else {
		a.currentReturnType = TypeVoid
	}

	// Analyze method body
	typedBody := a.analyzeBlockStmt(method.Body)

	// Restore previous state
	a.currentReturnType = prevReturnType
	a.exitScope()
	a.currentClass = prevClass

	var returnType Type = TypeVoid
	if methodInfo != nil {
		returnType = methodInfo.ReturnType
	}

	return &TypedMethodDecl{
		Name:       method.Name,
		NamePos:    method.NamePos,
		EqualsPos:  method.EqualsPos,
		LeftParen:  method.LeftParen,
		Parameters: typedParams,
		RightParen: method.RightParen,
		ArrowPos:   method.ArrowPos,
		ReturnType: returnType,
		ReturnPos:  method.ReturnPos,
		Body:       typedBody,
		IsStatic:   !isInstance,
	}
}

// analyzeFunctionDecl analyzes a function declaration
func (a *Analyzer) analyzeFunctionDecl(fn *ast.FunctionDecl) TypedDeclaration {
	// Get function info
	fnInfo := a.functions[fn.Name]

	// Set current return type and function name for return statement checking
	prevReturnType := a.currentReturnType
	prevFunctionName := a.currentFunctionName
	a.currentReturnType = fnInfo.ReturnType
	a.currentFunctionName = fn.Name

	// Enter a new scope for the function body
	a.enterScope()

	// Add parameters to scope
	typedParams := make([]TypedParameter, len(fn.Parameters))
	for i, param := range fn.Parameters {
		paramType := fnInfo.ParamTypes[i]
		typedParams[i] = TypedParameter{
			Name:    param.Name,
			NamePos: param.NamePos,
			Colon:   param.Colon,
			Type:    paramType,
			TypePos: param.TypePos,
		}
		// Declare parameter in scope (immutable)
		if !a.currentScope.declare(param.Name, paramType, false) {
			a.addError(fmt.Sprintf("parameter '%s' is already declared", param.Name), param.NamePos, param.NamePos)
		}
		// Track ownership for Own<T> parameters
		if IsMoveOnly(paramType) {
			a.ownershipScope.declare(param.Name, paramType)
		}
	}

	// Analyze the function body
	typedBody := a.analyzeBlockStmt(fn.Body)

	// Check that non-void functions return on all paths
	if _, isVoid := fnInfo.ReturnType.(VoidType); !isVoid {
		if !allPathsReturn(typedBody.Statements) {
			a.addError(
				fmt.Sprintf("function '%s' does not return a value on all code paths", fn.Name),
				fn.NamePos, fn.NamePos,
			).WithHint("ensure all branches end with a return statement")
		}
	}

	// Exit the function scope
	a.exitScope()

	// Restore previous return type and function name
	a.currentReturnType = prevReturnType
	a.currentFunctionName = prevFunctionName

	return &TypedFunctionDecl{
		Name:       fn.Name,
		NamePos:    fn.NamePos,
		EqualsPos:  fn.EqualsPos,
		LeftParen:  fn.LeftParen,
		Parameters: typedParams,
		RightParen: fn.RightParen,
		ArrowPos:   fn.ArrowPos,
		ReturnType: fnInfo.ReturnType,
		ReturnPos:  fn.ReturnPos,
		Body:       typedBody,
	}
}

// analyzeBlockStmt analyzes a block statement
func (a *Analyzer) analyzeBlockStmt(block *ast.BlockStmt) *TypedBlockStmt {
	typedStmts := make([]TypedStatement, 0, len(block.Statements))

	for _, stmt := range block.Statements {
		typedStmt := a.analyzeStatement(stmt)
		typedStmts = append(typedStmts, typedStmt)
	}

	return &TypedBlockStmt{
		LeftBrace:  block.LeftBrace,
		Statements: typedStmts,
		RightBrace: block.RightBrace,
	}
}

// analyzeBlockStmtForExpression analyzes a block in expression context.
// The last statement, if it's an IfStmt (directly or wrapped in ExprStmt), is analyzed as an expression.
func (a *Analyzer) analyzeBlockStmtForExpression(block *ast.BlockStmt) *TypedBlockStmt {
	typedStmts := make([]TypedStatement, 0, len(block.Statements))

	for i, stmt := range block.Statements {
		var typedStmt TypedStatement
		// For the last statement, check if it's an IfStmt that should be an expression
		if i == len(block.Statements)-1 {
			if ifStmt, ok := stmt.(*ast.IfStmt); ok {
				// Direct IfStmt - analyze as expression to get proper type
				typedStmt = a.analyzeIfExpression(ifStmt).(*TypedIfStmt)
			} else if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
				// Check if it's an ExprStmt containing an IfStmt
				if ifStmt, ok := exprStmt.Expr.(*ast.IfStmt); ok {
					// Analyze the IfStmt as expression and wrap in ExprStmt
					typedIf := a.analyzeIfExpression(ifStmt).(*TypedIfStmt)
					typedStmt = &TypedExprStmt{Expr: typedIf}
				} else {
					typedStmt = a.analyzeStatement(stmt)
				}
			} else {
				typedStmt = a.analyzeStatement(stmt)
			}
		} else {
			typedStmt = a.analyzeStatement(stmt)
		}
		typedStmts = append(typedStmts, typedStmt)
	}

	return &TypedBlockStmt{
		LeftBrace:  block.LeftBrace,
		Statements: typedStmts,
		RightBrace: block.RightBrace,
	}
}

// analyzeStatement performs semantic analysis on a statement
func (a *Analyzer) analyzeStatement(stmt ast.Statement) TypedStatement {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return a.analyzeExprStatement(s)
	case *ast.BlockStmt:
		return a.analyzeBlockStmt(s)
	case *ast.VarDeclStmt:
		return a.analyzeVarDeclStatement(s)
	case *ast.AssignStmt:
		return a.analyzeAssignStatement(s)
	case *ast.FieldAssignStmt:
		return a.analyzeFieldAssignStatement(s)
	case *ast.IndexAssignStmt:
		return a.analyzeIndexAssignStatement(s)
	case *ast.ReturnStmt:
		return a.analyzeReturnStatement(s)
	case *ast.IfStmt:
		return a.analyzeIfStatement(s)
	case *ast.ForStmt:
		return a.analyzeForStatement(s)
	case *ast.WhileStmt:
		return a.analyzeWhileStatement(s)
	case *ast.BreakStmt:
		return a.analyzeBreakStatement(s)
	case *ast.ContinueStmt:
		return a.analyzeContinueStatement(s)
	case *ast.WhenExpr:
		return a.analyzeWhenStatement(s)
	default:
		a.addError("unknown statement type", stmt.Pos(), stmt.End())
		return &TypedExprStmt{
			Expr: &TypedLiteralExpr{
				Type:     TypeError,
				LitType:  ast.LiteralTypeInteger,
				Value:    "0",
				StartPos: stmt.Pos(),
				EndPos:   stmt.End(),
			},
		}
	}
}

// analyzeVarDeclStatement analyzes a variable declaration statement
func (a *Analyzer) analyzeVarDeclStatement(stmt *ast.VarDeclStmt) TypedStatement {
	var typedInit TypedExpression
	var initType Type
	var declaredType Type

	// Check if this is an anonymous struct literal with a type annotation
	if anonLit, ok := stmt.Initializer.(*ast.AnonStructLiteral); ok {
		if stmt.TypeName == "" {
			a.addError("anonymous struct literal requires type annotation (e.g., val p: Point = { ... })",
				anonLit.LeftBrace, anonLit.RightBrace)
			typedInit = &TypedLiteralExpr{Type: ErrorType{}}
			initType = ErrorType{}
			declaredType = ErrorType{}
		} else {
			// Resolve the declared type
			declaredType = a.resolveTypeNameNoError(stmt.TypeName)
			if _, isErr := declaredType.(ErrorType); isErr {
				a.addError(
					fmt.Sprintf("unknown type '%s'", stmt.TypeName),
					stmt.TypePos, stmt.TypePos,
				)
				declaredType = TypeError
				typedInit = &TypedLiteralExpr{Type: ErrorType{}}
				initType = ErrorType{}
			} else if structType, ok := declaredType.(StructType); ok {
				// Analyze the anonymous struct literal with the expected type
				typedInit = a.analyzeAnonStructLiteralWithType(anonLit, structType)
				initType = typedInit.GetType()
			} else {
				a.addError(
					fmt.Sprintf("anonymous struct literal cannot be used with non-struct type '%s'", stmt.TypeName),
					anonLit.LeftBrace, anonLit.RightBrace,
				)
				typedInit = &TypedLiteralExpr{Type: ErrorType{}}
				initType = ErrorType{}
			}
		}
	} else {
		// Regular expression - analyze normally
		typedInit = a.analyzeExpression(stmt.Initializer)
		initType = typedInit.GetType()

		// Determine the declared type
		if stmt.TypeName != "" {
			// Explicit type annotation - use resolveTypeName to support struct types
			declaredType = a.resolveTypeNameNoError(stmt.TypeName)
			if _, isErr := declaredType.(ErrorType); isErr {
				a.addError(
					fmt.Sprintf("unknown type '%s'", stmt.TypeName),
					stmt.TypePos, stmt.TypePos,
				)
				declaredType = TypeError
			} else {
				// Type compatibility check
				if _, isErr := initType.(ErrorType); !isErr {
					// Check if initializer type is compatible with declared type
					if !a.checkTypeCompatibility(declaredType, initType, typedInit, stmt.Initializer.Pos()) {
						// Error already reported by checkTypeCompatibility
					}
				}
				// Refine array type: if annotation has unknown size but
				// initializer has concrete size, use the concrete size
				if declArr, ok := declaredType.(ArrayType); ok && declArr.Size == ArraySizeUnknown {
					if initArr, ok := initType.(ArrayType); ok && initArr.Size != ArraySizeUnknown {
						declaredType = ArrayType{
							ElementType: declArr.ElementType,
							Size:        initArr.Size,
						}
					}
				}
			}
		} else {
			// Infer type from initializer
			declaredType = initType

			// Check for bare null without type annotation
			if _, isNothing := initType.(NothingType); isNothing {
				a.addError("cannot infer type from null, add type annotation", stmt.Initializer.Pos(), stmt.Initializer.End())
				declaredType = TypeError
			}

			// For integer literals without type annotation, check bounds against i64 (the default type)
			if litExpr, ok := typedInit.(*TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeInteger {
				if !a.checkIntegerBounds(litExpr.Value, TypeInteger, litExpr.StartPos) {
					declaredType = TypeError
				}
			}
		}
	}

	// Ref<T> cannot be used as local variable type
	if IsRefPointer(declaredType) {
		a.addError("&T cannot be stored in local variables; references can only be function parameters",
			stmt.TypePos, stmt.TypePos)
	}

	// Check for duplicate declaration in the current scope
	if !a.currentScope.declare(stmt.Name, declaredType, stmt.Mutable) {
		a.addError(
			fmt.Sprintf("variable '%s' is already declared in this scope", stmt.Name),
			stmt.NamePos, stmt.NamePos,
		)
	}

	// Track ownership for move-only types and handle moves from initializer
	if IsMoveOnly(declaredType) {
		a.ownershipScope.declare(stmt.Name, declaredType)
		// Check if initializer is a variable being moved
		a.checkAndRecordMove(stmt.Initializer, stmt.Name, stmt.NamePos)
	}

	return &TypedVarDeclStmt{
		Keyword:      stmt.Keyword,
		Mutable:      stmt.Mutable,
		Name:         stmt.Name,
		NamePos:      stmt.NamePos,
		Colon:        stmt.Colon,
		TypeName:     stmt.TypeName,
		TypePos:      stmt.TypePos,
		DeclaredType: declaredType,
		Equals:       stmt.Equals,
		Initializer:  typedInit,
	}
}

// typeCheckContext indicates the context for type compatibility checking
type typeCheckContext int

const (
	contextAssignment typeCheckContext = iota // variable/field/element assignment
	contextReturn                             // return statement
)

// checkTypeCompatibility checks if an initializer is compatible with the declared type
func (a *Analyzer) checkTypeCompatibility(declaredType, initType Type, typedInit TypedExpression, pos ast.Position) bool {
	return a.checkTypeCompatibilityCore(declaredType, initType, typedInit, pos, contextAssignment)
}

// checkReturnTypeCompatibility checks if a return value is compatible with the expected return type.
func (a *Analyzer) checkReturnTypeCompatibility(expectedType, actualType Type, typedValue TypedExpression, pos ast.Position) bool {
	return a.checkTypeCompatibilityCore(expectedType, actualType, typedValue, pos, contextReturn)
}

// checkTypeCompatibilityCore is the shared implementation for type compatibility checking.
// It handles nullable types, literal bounds checking, and generates context-appropriate error messages.
func (a *Analyzer) checkTypeCompatibilityCore(targetType, sourceType Type, typedSource TypedExpression, pos ast.Position, ctx typeCheckContext) bool {
	// Helper to generate context-appropriate error messages
	errMsg := func(msg string) string {
		if ctx == contextReturn {
			return "return type mismatch: " + msg
		}
		return msg
	}

	// If either type is ErrorType, skip compatibility check to avoid cascading errors
	if _, isErr := targetType.(ErrorType); isErr {
		return true
	}
	if _, isErr := sourceType.(ErrorType); isErr {
		return true
	}

	// If types are exactly equal, always ok
	if targetType.Equals(sourceType) {
		return true
	}

	// Nullable type rules:
	// 1. Nothing (null) can be assigned to any T?
	if _, isNothing := sourceType.(NothingType); isNothing {
		if IsNullable(targetType) {
			return true
		}
		a.addError(
			errMsg(fmt.Sprintf("cannot assign null to non-nullable type '%s'", targetType.String())),
			pos, pos,
		)
		return false
	}

	// 2. T can be assigned to T? (implicit upcast)
	if nullableType, isNullable := targetType.(NullableType); isNullable {
		if nullableType.InnerType.Equals(sourceType) {
			return true
		}
		// Also allow integer literal bounds check for nullable integer types
		if litExpr, ok := typedSource.(*TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeInteger {
			if IsIntegerType(nullableType.InnerType) {
				return a.checkIntegerBounds(litExpr.Value, nullableType.InnerType, pos)
			}
		}
		// Also allow float literal bounds check for nullable float types
		if litExpr, ok := typedSource.(*TypedLiteralExpr); ok && litExpr.LitType == ast.LiteralTypeFloat {
			if IsFloatType(nullableType.InnerType) {
				return a.checkFloatBounds(litExpr.Value, nullableType.InnerType, pos)
			}
		}
	}

	// 3. T? cannot be assigned to T
	if IsNullable(sourceType) && !IsNullable(targetType) {
		innerType, _ := UnwrapNullable(sourceType)
		if innerType.Equals(targetType) {
			hint := "use a null check or provide a default value"
			if ctx == contextAssignment {
				hint = "use a null check like 'if x != null { ... }' or provide a default value"
			}
			a.addError(
				errMsg(fmt.Sprintf("cannot assign '%s' to '%s', handle null first", sourceType.String(), targetType.String())),
				pos, pos,
			).WithHint(hint)
			return false
		}
	}

	// Check for literal bounds when assigning to a specific type
	if litExpr, ok := typedSource.(*TypedLiteralExpr); ok {
		// Integer literal -> any integer type (with bounds check)
		if litExpr.LitType == ast.LiteralTypeInteger && IsIntegerType(targetType) {
			return a.checkIntegerBounds(litExpr.Value, targetType, pos)
		}

		// Float literal -> any float type (with bounds check)
		if litExpr.LitType == ast.LiteralTypeFloat && IsFloatType(targetType) {
			return a.checkFloatBounds(litExpr.Value, targetType, pos)
		}

		// Integer literal cannot be assigned to float type
		if litExpr.LitType == ast.LiteralTypeInteger && IsFloatType(targetType) {
			a.addError(
				errMsg(fmt.Sprintf("cannot assign integer literal to %s", targetType.String())),
				pos, pos,
			).WithHint("use a float literal like 42.0 instead")
			return false
		}

		// Float literal cannot be assigned to integer type
		if litExpr.LitType == ast.LiteralTypeFloat && IsIntegerType(targetType) {
			a.addError(
				errMsg(fmt.Sprintf("cannot assign float literal to %s", targetType.String())),
				pos, pos,
			)
			return false
		}
	}

	// Types don't match and no special conversion allowed
	if ctx == contextReturn {
		a.addError(
			fmt.Sprintf("return type mismatch: expected %s, got %s", targetType.String(), sourceType.String()),
			pos, pos,
		)
	} else {
		a.addError(
			fmt.Sprintf("cannot assign %s to variable of type %s", sourceType.String(), targetType.String()),
			pos, pos,
		)
	}
	return false
}

// checkLiteralIndexBounds checks if a literal index is within array bounds at compile time.
// If the index is not a literal, no check is performed (runtime check will handle it).
func (a *Analyzer) checkLiteralIndexBounds(index TypedExpression, arraySize int, startPos, endPos ast.Position) {
	// Skip bounds check when array size is unknown (from type annotation)
	if arraySize == ArraySizeUnknown {
		return
	}

	// Only check literal integer indices
	lit, ok := index.(*TypedLiteralExpr)
	if !ok || lit.LitType != ast.LiteralTypeInteger {
		return
	}

	// Parse the index value
	indexVal, err := strconv.ParseInt(lit.Value, 10, 64)
	if err != nil {
		return // Invalid literal, other errors will catch this
	}

	// Check bounds
	if indexVal < 0 {
		a.addError(
			fmt.Sprintf("array index %d is negative", indexVal),
			startPos, endPos,
		).WithHint("array indices must be non-negative")
	} else if indexVal >= int64(arraySize) {
		a.addError(
			fmt.Sprintf("array index %d is out of bounds for array of size %d", indexVal, arraySize),
			startPos, endPos,
		).WithHint(fmt.Sprintf("valid indices are 0 to %d", arraySize-1))
	}
}

// isExpressionMutable checks if an expression refers to a mutable location.
// Used to determine if a value can be borrowed as a mutable reference.
func (a *Analyzer) isExpressionMutable(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		// Check if the variable was declared with 'var'
		info, found := a.currentScope.lookup(e.Name)
		if found {
			return info.Mutable
		}
		return false
	case *ast.FieldAccessExpr:
		// For field access through pointer, check if the root variable is mutable
		return a.isExpressionMutable(e.Object)
	case *ast.IndexExpr:
		// For array index, check if the array is mutable
		return a.isExpressionMutable(e.Array)
	default:
		// Other expressions (like literals, function calls) are not mutable
		return false
	}
}

// analyzeAssignStatement analyzes a variable assignment statement
func (a *Analyzer) analyzeAssignStatement(stmt *ast.AssignStmt) TypedStatement {
	// Look up the variable
	info, found := a.currentScope.lookup(stmt.Name)
	if !found {
		a.addError(
			fmt.Sprintf("undefined variable '%s'", stmt.Name),
			stmt.NamePos, stmt.NamePos,
		).WithHint("did you forget to declare it with 'val' or 'var'?")
		// Return error node
		typedValue := a.analyzeExpression(stmt.Value)
		return &TypedAssignStmt{
			Name:    stmt.Name,
			NamePos: stmt.NamePos,
			Equals:  stmt.Equals,
			Value:   typedValue,
			VarType: TypeError,
		}
	}

	// Check mutability
	if !info.Mutable {
		a.addError(
			fmt.Sprintf("cannot assign to immutable variable '%s'", stmt.Name),
			stmt.NamePos, stmt.Equals,
		).WithHint("consider using 'var' instead of 'val' if you need to reassign")
	}

	// Check for direct self-assignment of move-only types: p = p
	// This would move p and then try to use it, which is invalid
	// Only check for direct identifier assignment, not field access like p = p.x
	if IsMoveOnly(info.Type) {
		if valueIdent, ok := stmt.Value.(*ast.IdentifierExpr); ok {
			if valueIdent.Name == stmt.Name {
				a.addError(
					fmt.Sprintf("cannot assign '%s' to itself", stmt.Name),
					stmt.NamePos, stmt.Value.End(),
				).WithHint("self-assignment of move-only types is not allowed")
			}
		}
	}

	// Analyze the value expression
	typedValue := a.analyzeExpression(stmt.Value)
	valueType := typedValue.GetType()

	// Type check using the same rules as variable declaration
	// This handles: null -> T?, T -> T?, exact match, etc.
	a.checkTypeCompatibility(info.Type, valueType, typedValue, stmt.Value.Pos())

	// Reassigning to a moved variable restores its ownership
	// Example: list = prepend(list, 1) - list is moved, then receives new value
	if IsMoveOnly(info.Type) {
		a.ownershipScope.restoreOwned(stmt.Name)
	}

	return &TypedAssignStmt{
		Name:    stmt.Name,
		NamePos: stmt.NamePos,
		Equals:  stmt.Equals,
		Value:   typedValue,
		VarType: info.Type,
	}
}

// analyzeFieldAssignStatement analyzes a field assignment statement (e.g., p.y = 25)
func (a *Analyzer) analyzeFieldAssignStatement(stmt *ast.FieldAssignStmt) TypedStatement {
	// Analyze the object expression (the struct being accessed)
	typedObject := a.analyzeExpression(stmt.Object)
	objectType := typedObject.GetType()

	// Auto-dereference owned pointers: Own<StructType> -> StructType
	// This allows p.x = 10 where p is Own<Point>
	if ownedType, isOwned := objectType.(OwnedPointerType); isOwned {
		objectType = ownedType.ElementType
	}

	// Auto-dereference reference pointers: Ref<T>/MutRef<T> -> T
	var accessingThroughImmutableRef bool
	var accessingThroughMutableRef bool
	if refType, isRef := objectType.(RefPointerType); isRef {
		objectType = refType.ElementType
		accessingThroughImmutableRef = true
	} else if mutRefType, isMutRef := objectType.(MutRefPointerType); isMutRef {
		objectType = mutRefType.ElementType
		accessingThroughMutableRef = true
	}
	_ = accessingThroughMutableRef // used for documentation, mutation is allowed

	// Check that the object is a struct or class type
	var fields []StructFieldInfo
	var typeName string

	if structType, isStruct := objectType.(StructType); isStruct {
		fields = structType.Fields
		typeName = structType.Name
	} else if classType, isClass := objectType.(ClassType); isClass {
		fields = classType.Fields
		typeName = classType.Name
	} else {
		if _, isErr := objectType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("cannot access field '%s' on non-struct type '%s'", stmt.Field, objectType.String()),
				stmt.Dot, stmt.FieldPos,
			)
		}
		// Still analyze the value to find any errors in it
		typedValue := a.analyzeExpression(stmt.Value)
		return &TypedFieldAssignStmt{
			Object:   typedObject,
			Dot:      stmt.Dot,
			Field:    stmt.Field,
			FieldPos: stmt.FieldPos,
			Equals:   stmt.Equals,
			Value:    typedValue,
		}
	}

	// Look up the field
	var fieldInfo StructFieldInfo
	var found bool
	for _, f := range fields {
		if f.Name == stmt.Field {
			fieldInfo = f
			found = true
			break
		}
	}
	if !found {
		a.addError(
			fmt.Sprintf("type '%s' has no field '%s'", typeName, stmt.Field),
			stmt.FieldPos, stmt.FieldPos,
		)
		typedValue := a.analyzeExpression(stmt.Value)
		return &TypedFieldAssignStmt{
			Object:   typedObject,
			Dot:      stmt.Dot,
			Field:    stmt.Field,
			FieldPos: stmt.FieldPos,
			Equals:   stmt.Equals,
			Value:    typedValue,
		}
	}

	// Check ref mutability (for assignment through Ref<T>)
	// Ref<T> is immutable - cannot assign through it
	// MutRef<T> is mutable - can assign through it
	if accessingThroughImmutableRef {
		a.addError(
			"cannot assign through immutable reference",
			stmt.Dot, stmt.FieldPos,
		).WithHint("use &&T for mutable references")
	}

	// Note: val/var on Own<T> bindings only controls reassignability, not mutation.
	// Mutation through Own<T> is allowed as long as the field is var.
	// (accessingThroughOwned allows mutation regardless of val/var binding)

	// Check field mutability
	if !fieldInfo.Mutable {
		a.addError(
			fmt.Sprintf("cannot assign to immutable field '%s'", stmt.Field),
			stmt.FieldPos, stmt.Equals,
		).WithHint("consider using 'var' instead of 'val' in the struct definition")
	}

	// Check for self-referential assignment of move-only types: n.next = n
	// This would create a cycle or cause ownership issues
	if IsMoveOnly(fieldInfo.Type) {
		targetRoot := GetRootVarName(stmt.Object)
		valueRoot := GetRootVarName(stmt.Value)
		if targetRoot != "" && targetRoot == valueRoot {
			a.addError(
				fmt.Sprintf("cannot assign '%s' to a field of itself", valueRoot),
				stmt.Object.Pos(), stmt.Value.End(),
			).WithHint("self-referential assignment of move-only types creates ownership cycles")
		}
	}

	// Analyze the value expression
	typedValue := a.analyzeExpression(stmt.Value)
	valueType := typedValue.GetType()

	// Type check using the same rules as variable declaration
	// This handles: null -> T?, T -> T?, exact match, etc.
	a.checkTypeCompatibility(fieldInfo.Type, valueType, typedValue, stmt.Value.Pos())

	return &TypedFieldAssignStmt{
		Object:   typedObject,
		Dot:      stmt.Dot,
		Field:    stmt.Field,
		FieldPos: stmt.FieldPos,
		Equals:   stmt.Equals,
		Value:    typedValue,
	}
}

// analyzeIndexAssignStatement analyzes an array index assignment (e.g., arr[0] = 5)
func (a *Analyzer) analyzeIndexAssignStatement(stmt *ast.IndexAssignStmt) TypedStatement {
	// Analyze the array expression
	typedArray := a.analyzeExpression(stmt.Array)
	arrayType := typedArray.GetType()

	// Analyze the index expression
	typedIndex := a.analyzeExpression(stmt.Index)
	indexType := typedIndex.GetType()

	// Analyze the value expression
	typedValue := a.analyzeExpression(stmt.Value)
	valueType := typedValue.GetType()

	// Unwrap reference types to support indexing through &&T[] parameters
	if ref, ok := arrayType.(RefPointerType); ok {
		arrayType = ref.ElementType
	} else if mutRef, ok := arrayType.(MutRefPointerType); ok {
		arrayType = mutRef.ElementType
	}

	// Check that the array is actually an array type
	arrType, isArray := arrayType.(ArrayType)
	if !isArray {
		if _, isErr := arrayType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("cannot index non-array type '%s'", arrayType.String()),
				stmt.LeftBracket, stmt.RightBracket,
			)
		}
		return &TypedIndexAssignStmt{
			Array:        typedArray,
			LeftBracket:  stmt.LeftBracket,
			Index:        typedIndex,
			RightBracket: stmt.RightBracket,
			Equals:       stmt.Equals,
			Value:        typedValue,
			ArraySize:    ArraySizeUnknown,
		}
	}

	// Check that the array variable is mutable
	if ident, ok := stmt.Array.(*ast.IdentifierExpr); ok {
		info, found := a.currentScope.lookup(ident.Name)
		if found && !info.Mutable {
			// Allow mutation through &&T[] mutable reference parameters
			isMutRef := false
			if _, ok := info.Type.(MutRefPointerType); ok {
				isMutRef = true
			}
			if !isMutRef {
				a.addError(
					fmt.Sprintf("cannot assign to element of immutable array '%s'", ident.Name),
					stmt.LeftBracket, stmt.Equals,
				).WithHint("consider using 'var' instead of 'val' if you need to modify elements")
			}
		}
	}

	// Check index type is integer
	if !IsIntegerType(indexType) {
		if _, isErr := indexType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("array index must be integer, got '%s'", indexType.String()),
				stmt.Index.Pos(), stmt.Index.End(),
			)
		}
	}

	// Compile-time bounds check for literal indices
	a.checkLiteralIndexBounds(typedIndex, arrType.Size, stmt.Index.Pos(), stmt.Index.End())

	// Type check using the same rules as variable declaration
	// This handles: null -> T?, T -> T?, exact match, etc.
	a.checkTypeCompatibility(arrType.ElementType, valueType, typedValue, stmt.Value.Pos())

	return &TypedIndexAssignStmt{
		Array:        typedArray,
		LeftBracket:  stmt.LeftBracket,
		Index:        typedIndex,
		RightBracket: stmt.RightBracket,
		Equals:       stmt.Equals,
		Value:        typedValue,
		ArraySize:    arrType.Size,
	}
}

// analyzeArrayLiteral analyzes an array literal expression (e.g., [1, 2, 3])
func (a *Analyzer) analyzeArrayLiteral(expr *ast.ArrayLiteralExpr) TypedExpression {
	// Empty arrays are not allowed (we need at least one element to infer the type)
	if len(expr.Elements) == 0 {
		a.addError(
			"empty array literals are not allowed",
			expr.LeftBracket, expr.RightBracket,
		).WithHint("array type cannot be inferred from an empty literal")
		return &TypedArrayLiteralExpr{
			Type:         ArrayType{ElementType: TypeError, Size: ArraySizeUnknown},
			LeftBracket:  expr.LeftBracket,
			Elements:     []TypedExpression{},
			RightBracket: expr.RightBracket,
		}
	}

	typedElements := make([]TypedExpression, len(expr.Elements))

	// Analyze first element to get type
	typedElements[0] = a.analyzeExpression(expr.Elements[0])
	elementType := typedElements[0].GetType()

	// Check for nested arrays (not supported)
	if _, isArray := elementType.(ArrayType); isArray {
		a.addError(
			"nested arrays are not supported",
			expr.LeftBracket, expr.RightBracket,
		).WithHint("arrays can only contain primitive types (i64, bool, string, etc.)")
		return &TypedArrayLiteralExpr{
			Type:         ArrayType{ElementType: TypeError, Size: ArraySizeUnknown},
			LeftBracket:  expr.LeftBracket,
			Elements:     typedElements,
			RightBracket: expr.RightBracket,
		}
	}

	// Analyze remaining elements and check type consistency
	for i := 1; i < len(expr.Elements); i++ {
		typedElements[i] = a.analyzeExpression(expr.Elements[i])
		elemType := typedElements[i].GetType()

		if _, isErr := elemType.(ErrorType); isErr {
			continue
		}

		if _, isErr := elementType.(ErrorType); isErr {
			elementType = elemType
			continue
		}

		if !elementType.Equals(elemType) {
			a.addError(
				fmt.Sprintf("array element type mismatch: expected %s, got %s",
					elementType.String(), elemType.String()),
				expr.Elements[i].Pos(), expr.Elements[i].End(),
			)
		}
	}

	arrayType := ArrayType{
		ElementType: elementType,
		Size:        len(expr.Elements),
	}

	return &TypedArrayLiteralExpr{
		Type:         arrayType,
		LeftBracket:  expr.LeftBracket,
		Elements:     typedElements,
		RightBracket: expr.RightBracket,
	}
}

// analyzeIndexExpr analyzes an array index expression (e.g., arr[0])
func (a *Analyzer) analyzeIndexExpr(expr *ast.IndexExpr) TypedExpression {
	// Analyze the array expression
	typedArray := a.analyzeExpression(expr.Array)
	arrayType := typedArray.GetType()

	// Analyze the index expression
	typedIndex := a.analyzeExpression(expr.Index)
	indexType := typedIndex.GetType()

	// Unwrap reference types to support indexing through &T[] and &&T[] parameters
	if ref, ok := arrayType.(RefPointerType); ok {
		arrayType = ref.ElementType
	} else if mutRef, ok := arrayType.(MutRefPointerType); ok {
		arrayType = mutRef.ElementType
	}

	// String indexing: s[i] returns a u8 byte, bounds checked at runtime.
	if _, isString := arrayType.(StringType); isString {
		if !IsIntegerType(indexType) {
			if _, isErr := indexType.(ErrorType); !isErr {
				a.addError(
					fmt.Sprintf("string index must be integer, got '%s'", indexType.String()),
					expr.Index.Pos(), expr.Index.End(),
				)
			}
		}
		return &TypedIndexExpr{
			Type:         TypeU8,
			Array:        typedArray,
			LeftBracket:  expr.LeftBracket,
			Index:        typedIndex,
			RightBracket: expr.RightBracket,
			ArraySize:    ArraySizeUnknown,
		}
	}

	// Check that the expression is an array type
	arrType, isArray := arrayType.(ArrayType)
	if !isArray {
		if _, isErr := arrayType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("cannot index non-array type '%s'", arrayType.String()),
				expr.LeftBracket, expr.RightBracket,
			)
		}
		return &TypedIndexExpr{
			Type:         TypeError,
			Array:        typedArray,
			LeftBracket:  expr.LeftBracket,
			Index:        typedIndex,
			RightBracket: expr.RightBracket,
			ArraySize:    ArraySizeUnknown,
		}
	}

	// Check index type is integer
	if !IsIntegerType(indexType) {
		if _, isErr := indexType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("array index must be integer, got '%s'", indexType.String()),
				expr.Index.Pos(), expr.Index.End(),
			)
		}
	}

	// Compile-time bounds check for literal indices
	a.checkLiteralIndexBounds(typedIndex, arrType.Size, expr.Index.Pos(), expr.Index.End())

	return &TypedIndexExpr{
		Type:         arrType.ElementType,
		Array:        typedArray,
		LeftBracket:  expr.LeftBracket,
		Index:        typedIndex,
		RightBracket: expr.RightBracket,
		ArraySize:    arrType.Size,
	}
}

// analyzeReturnStatement analyzes a return statement
func (a *Analyzer) analyzeReturnStatement(stmt *ast.ReturnStmt) TypedStatement {
	// Check if we're in a function
	if a.currentReturnType == nil {
		a.addError("return statement outside of function", stmt.Keyword, stmt.Keyword)
		return &TypedReturnStmt{
			Keyword:      stmt.Keyword,
			Value:        nil,
			ExpectedType: TypeVoid,
		}
	}

	// Analyze the return value if present
	var typedValue TypedExpression
	if stmt.Value != nil {
		typedValue = a.analyzeExpression(stmt.Value)
		valueType := typedValue.GetType()

		// Check type matches expected return type
		if _, isVoid := a.currentReturnType.(VoidType); isVoid {
			a.addError("void function should not return a value", stmt.Value.Pos(), stmt.Value.End())
		} else {
			a.checkReturnTypeCompatibility(a.currentReturnType, valueType, typedValue, stmt.Value.Pos())

			// Refine array return type: if declared type has unknown size but
			// the actual value has a known size, use the concrete size
			if declArr, ok := a.currentReturnType.(ArrayType); ok && declArr.Size == ArraySizeUnknown {
				if valArr, ok := valueType.(ArrayType); ok && valArr.Size != ArraySizeUnknown {
					refined := ArrayType{
						ElementType: declArr.ElementType,
						Size:        valArr.Size,
					}
					a.currentReturnType = refined
					// Update the function registry so callers get the concrete size
					if a.currentFunctionName != "" {
						if fnInfo, exists := a.functions[a.currentFunctionName]; exists {
							fnInfo.ReturnType = refined
							a.functions[a.currentFunctionName] = fnInfo
						}
					}
				}
			}
		}

		// Record move for return value if it's a move-only type
		// Returning a variable moves it out of the function
		if IsMoveOnly(valueType) {
			a.checkAndRecordMove(stmt.Value, "<return>", stmt.Value.Pos())
		}
	} else {
		// No return value
		if _, isVoid := a.currentReturnType.(VoidType); !isVoid {
			a.addError(
				fmt.Sprintf("function expects return value of type %s", a.currentReturnType.String()),
				stmt.Keyword, stmt.Keyword,
			)
		}
	}

	return &TypedReturnStmt{
		Keyword:      stmt.Keyword,
		Value:        typedValue,
		ExpectedType: a.currentReturnType,
	}
}

// analyzeIfCondition analyzes an if condition and validates it's a boolean.
// This is shared between analyzeIfStatement and analyzeIfExpression.
func (a *Analyzer) analyzeIfCondition(condition ast.Expression) TypedExpression {
	typedCond := a.analyzeExpression(condition)
	condType := typedCond.GetType()

	// Check that condition is boolean
	if _, isBool := condType.(BooleanType); !isBool {
		if _, isErr := condType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("if condition must be boolean, got '%s'", condType.String()),
				condition.Pos(), condition.End(),
			).WithHint("use a comparison like x > 0 or a boolean expression")
		}
	}

	return typedCond
}

// analyzeIfStatement analyzes an if statement
func (a *Analyzer) analyzeIfStatement(stmt *ast.IfStmt) TypedStatement {
	// Analyze and validate the condition
	typedCond := a.analyzeIfCondition(stmt.Condition)

	// Snapshot ownership state before branches
	// We need to track which variables are moved in any branch
	beforeSnapshot := a.snapshotOwnershipState()

	// Analyze the then branch (with its own scope)
	a.enterScope()
	typedThenBranch := a.analyzeBlockStmt(stmt.ThenBranch)
	thenMoves := a.collectBranchMoves(beforeSnapshot)
	a.exitScope()

	// Collect moves from else branch
	var elseMoves []moveRecord

	// Analyze the else branch if present
	var typedElseBranch TypedStatement
	if stmt.ElseBranch != nil {
		switch elseBranch := stmt.ElseBranch.(type) {
		case *ast.IfStmt:
			// else if: recursively analyze (no extra scope needed, the if will create its own)
			typedElseBranch = a.analyzeIfStatement(elseBranch)
			// The nested if statement handles its own move merging
			elseMoves = a.collectBranchMoves(beforeSnapshot)
		case *ast.BlockStmt:
			// else block: create scope
			a.enterScope()
			typedElseBranch = a.analyzeBlockStmt(elseBranch)
			elseMoves = a.collectBranchMoves(beforeSnapshot)
			a.exitScope()
		default:
			a.addError("unexpected else branch type", stmt.ElseBranch.Pos(), stmt.ElseBranch.End())
		}
	}

	// Merge moves from both branches: if a variable was moved in ANY branch,
	// it must be considered moved after the if statement
	a.mergeConditionalMoves(thenMoves, elseMoves)

	return &TypedIfStmt{
		IfKeyword:   stmt.IfKeyword,
		Condition:   typedCond,
		ThenBranch:  typedThenBranch,
		ElseKeyword: stmt.ElseKeyword,
		ElseBranch:  typedElseBranch,
	}
}

// analyzeForStatement analyzes a for-loop statement
func (a *Analyzer) analyzeForStatement(stmt *ast.ForStmt) TypedStatement {
	// Enter a new scope for the loop (loop variable should be scoped to the loop)
	a.enterScope()

	// Analyze initialization if present
	var typedInit TypedStatement
	if stmt.Init != nil {
		typedInit = a.analyzeStatement(stmt.Init)
	}

	// Analyze condition if present
	var typedCond TypedExpression
	if stmt.Condition != nil {
		typedCond = a.analyzeExpression(stmt.Condition)
		condType := typedCond.GetType()

		// Condition must be boolean
		if _, isBool := condType.(BooleanType); !isBool {
			if _, isErr := condType.(ErrorType); !isErr {
				a.addError(
					fmt.Sprintf("for-loop condition must be boolean, got '%s'", condType.String()),
					stmt.Condition.Pos(), stmt.Condition.End(),
				).WithHint("use a comparison like i < 10 or a boolean expression")
			}
		}
	}

	// Analyze update if present
	var typedUpdate TypedStatement
	if stmt.Update != nil {
		typedUpdate = a.analyzeStatement(stmt.Update)
	}

	// Enter loop context for break/continue validation and ownership tracking
	a.loopDepth++
	a.ownershipScope.inLoop = true

	// Analyze body
	typedBody := a.analyzeBlockStmt(stmt.Body)

	// Exit loop context
	a.loopDepth--

	// Exit loop scope
	a.exitScope()

	return &TypedForStmt{
		ForKeyword: stmt.ForKeyword,
		Init:       typedInit,
		Condition:  typedCond,
		Update:     typedUpdate,
		Body:       typedBody,
	}
}

// analyzeWhileStatement analyzes a while statement
func (a *Analyzer) analyzeWhileStatement(stmt *ast.WhileStmt) TypedStatement {
	// Analyze condition (required for while loops)
	typedCond := a.analyzeExpression(stmt.Condition)
	condType := typedCond.GetType()

	// Condition must be boolean
	if _, isBool := condType.(BooleanType); !isBool {
		if _, isErr := condType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("while-loop condition must be boolean, got '%s'", condType.String()),
				stmt.Condition.Pos(), stmt.Condition.End(),
			).WithHint("use a comparison like i < 10 or a boolean expression")
		}
	}

	// Enter loop context for break/continue validation and ownership tracking
	a.loopDepth++
	a.ownershipScope.inLoop = true

	// Analyze body
	typedBody := a.analyzeBlockStmt(stmt.Body)

	// Exit loop context
	a.loopDepth--

	return &TypedWhileStmt{
		WhileKeyword: stmt.WhileKeyword,
		Condition:    typedCond,
		Body:         typedBody,
	}
}

// analyzeBreakStatement analyzes a break statement
func (a *Analyzer) analyzeBreakStatement(stmt *ast.BreakStmt) TypedStatement {
	if a.loopDepth == 0 {
		a.addError("'break' statement not inside a loop", stmt.Keyword, stmt.Keyword).
			WithHint("break can only be used inside for or while loops")
	}
	return &TypedBreakStmt{Keyword: stmt.Keyword}
}

// analyzeContinueStatement analyzes a continue statement
func (a *Analyzer) analyzeContinueStatement(stmt *ast.ContinueStmt) TypedStatement {
	if a.loopDepth == 0 {
		a.addError("'continue' statement not inside a loop", stmt.Keyword, stmt.Keyword).
			WithHint("continue can only be used inside for or while loops")
	}
	return &TypedContinueStmt{Keyword: stmt.Keyword}
}

// analyzeIfExpression analyzes an if expression (if used in expression context)
func (a *Analyzer) analyzeIfExpression(stmt *ast.IfStmt) TypedExpression {
	// Analyze and validate the condition
	typedCond := a.analyzeIfCondition(stmt.Condition)

	// If expressions require an else branch
	if stmt.ElseBranch == nil {
		a.addError(
			"if expression must have an else branch",
			stmt.IfKeyword, stmt.ThenBranch.End(),
		).WithHint("add an else branch to provide a value for all cases")
		// Still analyze the then branch
		a.enterScope()
		typedThenBranch := a.analyzeBlockStmt(stmt.ThenBranch)
		a.exitScope()
		return &TypedIfStmt{
			IfKeyword:   stmt.IfKeyword,
			Condition:   typedCond,
			ThenBranch:  typedThenBranch,
			ElseKeyword: stmt.ElseKeyword,
			ElseBranch:  nil,
			ResultType:  TypeError,
		}
	}

	// Snapshot ownership state before branches for conditional move tracking
	beforeSnapshot := a.snapshotOwnershipState()

	// Analyze the then branch (with its own scope)
	a.enterScope()
	typedThenBranch := a.analyzeBlockStmtForExpression(stmt.ThenBranch)
	thenType := a.getBlockResultType(typedThenBranch)
	thenMoves := a.collectBranchMoves(beforeSnapshot)
	a.exitScope()

	// Collect moves from else branch
	var elseMoves []moveRecord

	// Analyze the else branch
	var typedElseBranch TypedStatement
	var elseType Type

	switch elseBranch := stmt.ElseBranch.(type) {
	case *ast.IfStmt:
		// else if: recursively analyze as expression
		typedElseExpr := a.analyzeIfExpression(elseBranch)
		typedElseBranch = typedElseExpr.(*TypedIfStmt)
		elseType = typedElseExpr.GetType()
		// The nested if expression handles its own move merging
		elseMoves = a.collectBranchMoves(beforeSnapshot)
	case *ast.BlockStmt:
		// else block: create scope
		a.enterScope()
		typedBlock := a.analyzeBlockStmtForExpression(elseBranch)
		typedElseBranch = typedBlock
		elseType = a.getBlockResultType(typedBlock)
		elseMoves = a.collectBranchMoves(beforeSnapshot)
		a.exitScope()
	default:
		a.addError("unexpected else branch type", stmt.ElseBranch.Pos(), stmt.ElseBranch.End())
		elseType = TypeError
	}

	// Merge moves from both branches: if a variable was moved in ANY branch,
	// it must be considered moved after the if expression
	a.mergeConditionalMoves(thenMoves, elseMoves)

	// Check that both branches have the same type
	var resultType Type = thenType
	if _, isErr := thenType.(ErrorType); !isErr {
		if _, isErr := elseType.(ErrorType); !isErr {
			if !thenType.Equals(elseType) {
				a.addError(
					fmt.Sprintf("if expression branches have different types: '%s' and '%s'",
						thenType.String(), elseType.String()),
					stmt.IfKeyword, stmt.End(),
				).WithHint("both branches must evaluate to the same type")
				resultType = TypeError
			}
		} else {
			resultType = TypeError
		}
	} else {
		resultType = TypeError
	}

	return &TypedIfStmt{
		IfKeyword:   stmt.IfKeyword,
		Condition:   typedCond,
		ThenBranch:  typedThenBranch,
		ElseKeyword: stmt.ElseKeyword,
		ElseBranch:  typedElseBranch,
		ResultType:  resultType,
	}
}

// getBlockResultType returns the type of a block's result (last expression)
func (a *Analyzer) getBlockResultType(block *TypedBlockStmt) Type {
	if len(block.Statements) == 0 {
		return TypeVoid
	}
	// The result is the last statement's type if it's an expression statement
	lastStmt := block.Statements[len(block.Statements)-1]
	if exprStmt, ok := lastStmt.(*TypedExprStmt); ok {
		return exprStmt.Expr.GetType()
	}
	// If last statement is an if-expression, get its type
	if ifStmt, ok := lastStmt.(*TypedIfStmt); ok {
		return ifStmt.GetType()
	}
	// If last statement is a when-expression, get its type
	if whenExpr, ok := lastStmt.(*TypedWhenExpr); ok {
		return whenExpr.GetType()
	}
	return TypeVoid
}

// analyzeExprStatement analyzes an expression statement
func (a *Analyzer) analyzeExprStatement(stmt *ast.ExprStmt) TypedStatement {
	typedExpr := a.analyzeExpression(stmt.Expr)
	return &TypedExprStmt{
		Expr: typedExpr,
	}
}

// analyzeExpression performs semantic analysis on an expression
func (a *Analyzer) analyzeExpression(expr ast.Expression) TypedExpression {
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return a.analyzeLiteral(e)
	case *ast.BinaryExpr:
		return a.analyzeBinaryExpression(e)
	case *ast.UnaryExpr:
		return a.analyzeUnaryExpression(e)
	case *ast.IdentifierExpr:
		return a.analyzeIdentifier(e)
	case *ast.CallExpr:
		return a.analyzeCallExpr(e)
	case *ast.StructLiteral:
		return a.analyzeStructLiteralExpr(e)
	case *ast.AnonStructLiteral:
		// Anonymous struct literals require type context - report error here
		// The proper way is to use analyzeAnonStructLiteralWithType from variable declaration
		a.addError("anonymous struct literal requires type annotation (e.g., val p: Point = { ... })", e.LeftBrace, e.RightBrace)
		return &TypedLiteralExpr{Type: ErrorType{}}
	case *ast.FieldAccessExpr:
		return a.analyzeFieldAccessExpr(e)
	case *ast.NewExpr:
		return a.analyzeNewExpr(e)
	case *ast.MethodCallExpr:
		return a.analyzeMethodCallExpr(e)
	case *ast.SafeCallExpr:
		return a.analyzeSafeCallExpr(e)
	case *ast.ArrayLiteralExpr:
		return a.analyzeArrayLiteral(e)
	case *ast.IndexExpr:
		return a.analyzeIndexExpr(e)
	case *ast.GroupingExpr:
		// Grouping is purely syntactic for precedence; just analyze the inner expression
		return a.analyzeExpression(e.Expr)
	case *ast.IfStmt:
		// If can be used as an expression
		return a.analyzeIfExpression(e)
	case *ast.WhenExpr:
		// When can be used as an expression
		return a.analyzeWhenExpression(e)
	case *ast.SelfExpr:
		return a.analyzeSelfExpr(e)
	default:
		a.addError("unknown expression type", expr.Pos(), expr.End())
		return &TypedLiteralExpr{
			Type:     TypeError,
			LitType:  ast.LiteralTypeInteger,
			Value:    "0",
			StartPos: expr.Pos(),
			EndPos:   expr.End(),
		}
	}
}

// analyzeCallExpr analyzes a function call expression
func (a *Analyzer) analyzeCallExpr(call *ast.CallExpr) TypedExpression {
	// Check for built-in functions first
	if builtin, ok := Builtins[call.Name]; ok {
		return a.analyzeBuiltinCall(call, builtin)
	}

	// Check if this is a struct construction
	if structType, ok := a.TypeRegistry.LookupStruct(call.Name); ok {
		return a.analyzeStructLiteral(call, structType)
	}

	// Look up the function
	fnInfo, exists := a.functions[call.Name]
	if !exists {
		a.addError(
			fmt.Sprintf("undefined function '%s'", call.Name),
			call.NamePos, call.NamePos,
		)
		// Return error typed call
		typedArgs := make([]TypedExpression, len(call.Arguments))
		for i, arg := range call.Arguments {
			typedArgs[i] = a.analyzeExpression(arg)
		}
		return &TypedCallExpr{
			Type:       TypeError,
			Name:       call.Name,
			NamePos:    call.NamePos,
			LeftParen:  call.LeftParen,
			Arguments:  typedArgs,
			RightParen: call.RightParen,
		}
	}

	// Check argument count
	if len(call.Arguments) != len(fnInfo.ParamTypes) {
		a.addError(
			fmt.Sprintf("function '%s' expects %d arguments, got %d",
				call.Name, len(fnInfo.ParamTypes), len(call.Arguments)),
			call.LeftParen, call.RightParen,
		)
	}

	// Check for too many arguments (redundant with parameter check, but catches mismatches)
	if len(call.Arguments) > maxFunctionParameters {
		a.addError(
			fmt.Sprintf("call to '%s' has %d arguments, maximum allowed is %d",
				call.Name, len(call.Arguments), maxFunctionParameters),
			call.LeftParen, call.RightParen,
		).WithHint("consider passing a struct or reducing the number of parameters")
	}

	// Analyze arguments and check types
	// Also track borrows for exclusivity checking
	typedArgs := make([]TypedExpression, len(call.Arguments))
	var borrows []BorrowInfo

	for i, arg := range call.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding parameter
		if i < len(fnInfo.ParamTypes) {
			argType := typedArgs[i].GetType()
			paramType := fnInfo.ParamTypes[i]

			// Skip if argument has error type
			if _, isErr := argType.(ErrorType); isErr {
				continue
			}

			// Check for implicit Own<T> to Ref<T> conversion (auto-borrow, immutable)
			if refType, ok := paramType.(RefPointerType); ok {
				if ownType, isOwn := argType.(OwnedPointerType); isOwn {
					// Auto-borrow: Own<T> -> Ref<T> if element types match
					if refType.ElementType.Equals(ownType.ElementType) {
						// Track this borrow for exclusivity checking (immutable borrow)
						borrows = append(borrows, BorrowInfo{
							VarName:  GetRootVarName(arg),
							Mutable:  false,
							Position: arg.Pos(),
							ArgIndex: i,
						})
						// Auto-borrow is valid
						continue
					}
				}
				// Auto-borrow: ArrayType -> Ref<ArrayType> (pass array by immutable reference)
				if _, isArr := argType.(ArrayType); isArr {
					if refType.ElementType.Equals(argType) {
						continue
					}
				}
			}

			// Check for implicit Own<T> to MutRef<T> conversion (auto-borrow, mutable)
			if mutRefType, ok := paramType.(MutRefPointerType); ok {
				if ownType, isOwn := argType.(OwnedPointerType); isOwn {
					// Auto-borrow: Own<T> -> MutRef<T> if element types match
					if mutRefType.ElementType.Equals(ownType.ElementType) {
						// Track this borrow for exclusivity checking (mutable borrow)
						borrows = append(borrows, BorrowInfo{
							VarName:  GetRootVarName(arg),
							Mutable:  true,
							Position: arg.Pos(),
							ArgIndex: i,
						})
						// Auto-borrow is valid
						continue
					}
				}
				// Auto-borrow: ArrayType -> MutRef<ArrayType> (pass array by mutable reference)
				if _, isArr := argType.(ArrayType); isArr {
					if mutRefType.ElementType.Equals(argType) {
						continue
					}
				}
			}

			if !IsAssignableTo(argType, paramType) {
				a.addError(
					fmt.Sprintf("argument %d: expected %s, got %s",
						i+1, paramType.String(), argType.String()),
					arg.Pos(), arg.End(),
				)
			} else {
				// If passing an owned pointer (to Own<T> or Own<T>? param), ownership moves
				if IsOwnedPointer(argType) || IsNullableOwnedPointer(argType) {
					a.checkAndRecordMove(arg, "<param>", arg.Pos())
				}
			}
		}
	}

	// Check for borrow exclusivity conflicts
	if hasConflict, msg, pos1, pos2 := CheckBorrowConflicts(borrows); hasConflict {
		a.addError(msg, pos1, pos2)
	}

	return &TypedCallExpr{
		Type:       fnInfo.ReturnType,
		Name:       call.Name,
		NamePos:    call.NamePos,
		LeftParen:  call.LeftParen,
		Arguments:  typedArgs,
		RightParen: call.RightParen,
	}
}

// analyzeBuiltinCall analyzes a call to a built-in function
func (a *Analyzer) analyzeBuiltinCall(call *ast.CallExpr, builtin BuiltinFunc) TypedExpression {
	// Special handling for len() - it returns a TypedLenExpr
	if builtin.IsArrayLen {
		return a.analyzeLenBuiltin(call)
	}

	// Check argument count
	if len(call.Arguments) != len(builtin.ParamTypes) {
		a.addError(
			fmt.Sprintf("built-in function '%s' expects %d argument(s), got %d",
				call.Name, len(builtin.ParamTypes), len(call.Arguments)),
			call.LeftParen, call.RightParen,
		)
	}

	// Analyze arguments and check types
	typedArgs := make([]TypedExpression, len(call.Arguments))
	for i, arg := range call.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding parameter
		if i < len(builtin.ParamTypes) {
			argType := typedArgs[i].GetType()

			// Skip error types
			if _, isErr := argType.(ErrorType); isErr {
				continue
			}

			// Check if this parameter has AcceptedTypes (accepts multiple types)
			if acceptedTypes, ok := builtin.AcceptedTypes[i]; ok {
				if !isAcceptedType(argType, acceptedTypes) {
					a.addError(
						fmt.Sprintf("argument %d: expected one of %s, got %s",
							i+1, formatAcceptedTypes(acceptedTypes), argType.String()),
						arg.Pos(), arg.End(),
					)
				}
			} else {
				// Normal type checking against ParamTypes
				paramType := builtin.ParamTypes[i]
				if !paramType.Equals(argType) && !isCompatibleIntegerType(paramType, argType) {
					a.addError(
						fmt.Sprintf("argument %d: expected %s, got %s",
							i+1, paramType.String(), argType.String()),
						arg.Pos(), arg.End(),
					)
				}
			}
		}
	}

	return &TypedCallExpr{
		Type:       builtin.ReturnType,
		Name:       call.Name,
		NamePos:    call.NamePos,
		LeftParen:  call.LeftParen,
		Arguments:  typedArgs,
		RightParen: call.RightParen,
	}
}

// analyzeLenBuiltin analyzes a len() call on an array or string
func (a *Analyzer) analyzeLenBuiltin(call *ast.CallExpr) TypedExpression {
	// Check argument count
	if len(call.Arguments) != 1 {
		a.addError(
			fmt.Sprintf("len() takes exactly 1 argument, got %d", len(call.Arguments)),
			call.LeftParen, call.RightParen,
		)
		// Return error typed expression
		return &TypedLenExpr{
			Type:       TypeS64,
			Array:      nil,
			ArraySize:  ArraySizeUnknown,
			NamePos:    call.NamePos,
			LeftParen:  call.LeftParen,
			RightParen: call.RightParen,
		}
	}

	// Analyze the argument
	typedArg := a.analyzeExpression(call.Arguments[0])
	argType := typedArg.GetType()

	// Accept array types (compile-time size) or string types (runtime length from header)
	if arrayType, isArray := argType.(ArrayType); isArray {
		return &TypedLenExpr{
			Type:       TypeS64,
			Array:      typedArg,
			ArraySize:  arrayType.Size,
			NamePos:    call.NamePos,
			LeftParen:  call.LeftParen,
			RightParen: call.RightParen,
		}
	}
	if _, isString := argType.(StringType); isString {
		return &TypedLenExpr{
			Type:       TypeS64,
			Array:      typedArg,
			ArraySize:  ArraySizeUnknown,
			NamePos:    call.NamePos,
			LeftParen:  call.LeftParen,
			RightParen: call.RightParen,
		}
	}

	if _, isErr := argType.(ErrorType); !isErr {
		a.addError(
			fmt.Sprintf("len() argument must be an array or string, got '%s'", argType.String()),
			call.Arguments[0].Pos(), call.Arguments[0].End(),
		)
	}
	return &TypedLenExpr{
		Type:       TypeS64,
		Array:      typedArg,
		ArraySize:  ArraySizeUnknown,
		NamePos:    call.NamePos,
		LeftParen:  call.LeftParen,
		RightParen: call.RightParen,
	}
}

// analyzeStructLiteral analyzes a struct construction expression (e.g., Point(10, 20) or Point(x: 10, y: 20))
func (a *Analyzer) analyzeStructLiteral(call *ast.CallExpr, structType StructType) TypedExpression {
	// Handle named arguments
	if call.HasNamedArguments() {
		return a.analyzeStructLiteralNamed(call, structType)
	}

	// Handle positional arguments
	// Check argument count matches field count
	if len(call.Arguments) != len(structType.Fields) {
		a.addError(
			fmt.Sprintf("struct '%s' has %d field(s), but %d argument(s) were provided",
				structType.Name, len(structType.Fields), len(call.Arguments)),
			call.LeftParen, call.RightParen,
		)
	}

	// Analyze arguments and check types
	typedArgs := make([]TypedExpression, len(call.Arguments))
	for i, arg := range call.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding field
		// Use checkTypeCompatibilityCore to allow nullable coercions (i64 -> i64?, null -> T?)
		if i < len(structType.Fields) {
			fieldType := structType.Fields[i].Type
			a.checkTypeCompatibilityCore(fieldType, typedArgs[i].GetType(), typedArgs[i], arg.Pos(), contextAssignment)
		}
	}

	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    call.NamePos,
		LeftBrace:  call.LeftParen,
		Args:       typedArgs,
		RightBrace: call.RightParen,
	}
}

// analyzeStructLiteralNamed analyzes a struct construction with named arguments (e.g., Point(x: 10, y: 20))
func (a *Analyzer) analyzeStructLiteralNamed(call *ast.CallExpr, structType StructType) TypedExpression {
	// Build a map of field name -> index for quick lookup
	fieldIndex := make(map[string]int)
	for i, field := range structType.Fields {
		fieldIndex[field.Name] = i
	}

	// Check argument count matches field count
	if len(call.NamedArguments) != len(structType.Fields) {
		a.addError(
			fmt.Sprintf("struct '%s' has %d field(s), but %d argument(s) were provided",
				structType.Name, len(structType.Fields), len(call.NamedArguments)),
			call.LeftParen, call.RightParen,
		)
	}

	// Track which fields have been provided (for duplicate detection)
	providedFields := make(map[string]ast.Position)

	// Create typed arguments array in field order
	typedArgs := make([]TypedExpression, len(structType.Fields))

	for _, namedArg := range call.NamedArguments {
		// Check if field exists
		idx, exists := fieldIndex[namedArg.Name]
		if !exists {
			a.addError(
				fmt.Sprintf("struct '%s' has no field '%s'", structType.Name, namedArg.Name),
				namedArg.NamePos, namedArg.NamePos,
			)
			continue
		}

		// Check for duplicate field
		if prevPos, duplicate := providedFields[namedArg.Name]; duplicate {
			a.addError(
				fmt.Sprintf("field '%s' specified multiple times", namedArg.Name),
				namedArg.NamePos, namedArg.NamePos,
			).WithHint(fmt.Sprintf("first specified at line %d", prevPos.Line))
			continue
		}
		providedFields[namedArg.Name] = namedArg.NamePos

		// Analyze the argument value
		typedArg := a.analyzeExpression(namedArg.Value)
		typedArgs[idx] = typedArg

		// Type check - use checkTypeCompatibilityCore to allow nullable coercions
		fieldType := structType.Fields[idx].Type
		a.checkTypeCompatibilityCore(fieldType, typedArg.GetType(), typedArg, namedArg.Value.Pos(), contextAssignment)
	}

	// Check for missing fields
	for _, field := range structType.Fields {
		if _, provided := providedFields[field.Name]; !provided {
			// Only report if we haven't already reported a count mismatch
			if len(call.NamedArguments) == len(structType.Fields) {
				a.addError(
					fmt.Sprintf("missing field '%s' in struct '%s'", field.Name, structType.Name),
					call.LeftParen, call.RightParen,
				)
			}
		}
	}

	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    call.NamePos,
		LeftBrace:  call.LeftParen,
		Args:       typedArgs,
		RightBrace: call.RightParen,
	}
}

// analyzeStructLiteralExpr analyzes a struct literal expression with braces (e.g., Point { 10, 20 } or Point { x: 10, y: 20 })
// Also handles class literals (e.g., Counter { 10 })
func (a *Analyzer) analyzeStructLiteralExpr(lit *ast.StructLiteral) TypedExpression {
	// Handle qualified struct literal (pkg.Type{ ... })
	if lit.PackageAlias != "" {
		return a.analyzeQualifiedStructLiteral(lit)
	}

	// Check if this is a known struct type
	if structType, ok := a.TypeRegistry.LookupStruct(lit.Name); ok {
		// Handle named arguments
		if lit.HasNamedArguments() {
			return a.analyzeStructLiteralExprNamed(lit, structType)
		}
		return a.analyzeStructLiteralPositional(lit, structType.Fields, structType.Name, func(args []TypedExpression) TypedExpression {
			return &TypedStructLiteralExpr{
				Type:       structType,
				TypePos:    lit.NamePos,
				LeftBrace:  lit.LeftBrace,
				Args:       args,
				RightBrace: lit.RightBrace,
			}
		})
	}

	// Check if this is a known class type
	if classType, ok := a.TypeRegistry.LookupClass(lit.Name); ok {
		// Handle named arguments
		if lit.HasNamedArguments() {
			return a.analyzeClassLiteralExprNamed(lit, classType)
		}
		return a.analyzeStructLiteralPositional(lit, classType.Fields, classType.Name, func(args []TypedExpression) TypedExpression {
			return &TypedClassLiteralExpr{
				Type:       classType,
				TypePos:    lit.NamePos,
				LeftBrace:  lit.LeftBrace,
				Args:       args,
				RightBrace: lit.RightBrace,
			}
		})
	}

	// Check if this is an object type (objects cannot be instantiated)
	if _, ok := a.TypeRegistry.LookupObject(lit.Name); ok {
		a.addError(
			fmt.Sprintf("cannot instantiate object '%s' (objects are singletons)", lit.Name),
			lit.NamePos, lit.RightBrace,
		)
		return &TypedLiteralExpr{Type: ErrorType{}}
	}

	// Unknown type
	a.addError(
		fmt.Sprintf("undefined type '%s'", lit.Name),
		lit.NamePos, lit.NamePos,
	)
	return &TypedLiteralExpr{Type: ErrorType{}}
}

// analyzeNamespaceFieldAccess handles accessing a variable or constant from a package namespace.
func (a *Analyzer) analyzeNamespaceFieldAccess(ns *PackageNamespace, expr *ast.FieldAccessExpr) TypedExpression {
	export, found := ns.Exports[expr.Field]
	if !found {
		a.addError(
			fmt.Sprintf("package '%s' has no declaration '%s'", ns.Path, expr.Field),
			expr.FieldPos, expr.FieldPos,
		)
		return &TypedFieldAccessExpr{
			Type:     TypeError,
			Object:   &TypedIdentifierExpr{Type: TypeError, Name: ns.Path},
			Dot:      expr.Dot,
			Field:    expr.Field,
			FieldPos: expr.FieldPos,
		}
	}

	// Return a typed identifier that references the cross-package variable
	// The name is qualified: "config.db_port" for later IR mangling
	return &TypedIdentifierExpr{
		Type:     export.Type,
		Name:     ns.Path + "." + expr.Field,
		StartPos: expr.Object.Pos(),
		EndPos:   expr.FieldPos,
	}
}

// analyzeQualifiedStructLiteral handles pkg.Type{ ... } struct construction.
func (a *Analyzer) analyzeQualifiedStructLiteral(lit *ast.StructLiteral) TypedExpression {
	// Look up the package namespace
	varInfo, found := a.currentScope.lookup(lit.PackageAlias)
	if !found {
		a.addError(
			fmt.Sprintf("undefined package '%s'", lit.PackageAlias),
			lit.PackageAliasPos, lit.PackageAliasPos,
		)
		return &TypedLiteralExpr{Type: ErrorType{}}
	}

	nsType, isNs := varInfo.Type.(PackageNamespaceType)
	if !isNs {
		a.addError(
			fmt.Sprintf("'%s' is not a package", lit.PackageAlias),
			lit.PackageAliasPos, lit.PackageAliasPos,
		)
		return &TypedLiteralExpr{Type: ErrorType{}}
	}

	ns := nsType.Namespace

	// Look up the type in the package's exports
	export, exportFound := ns.Exports[lit.Name]
	if !exportFound {
		a.addError(
			fmt.Sprintf("package '%s' has no type '%s'", ns.Path, lit.Name),
			lit.NamePos, lit.NamePos,
		)
		return &TypedLiteralExpr{Type: ErrorType{}}
	}

	// Must be a struct or class type
	switch t := export.Type.(type) {
	case StructType:
		if lit.HasNamedArguments() {
			return a.analyzeStructLiteralExprNamed(lit, t)
		}
		return a.analyzeStructLiteralPositional(lit, t.Fields, t.Name, func(args []TypedExpression) TypedExpression {
			return &TypedStructLiteralExpr{
				Type:       t,
				TypePos:    lit.NamePos,
				LeftBrace:  lit.LeftBrace,
				Args:       args,
				RightBrace: lit.RightBrace,
			}
		})
	case ClassType:
		if lit.HasNamedArguments() {
			return a.analyzeClassLiteralExprNamed(lit, t)
		}
		return a.analyzeStructLiteralPositional(lit, t.Fields, t.Name, func(args []TypedExpression) TypedExpression {
			return &TypedClassLiteralExpr{
				Type:       t,
				TypePos:    lit.NamePos,
				LeftBrace:  lit.LeftBrace,
				Args:       args,
				RightBrace: lit.RightBrace,
			}
		})
	default:
		a.addError(
			fmt.Sprintf("'%s.%s' is not a struct or class type", ns.Path, lit.Name),
			lit.NamePos, lit.NamePos,
		)
		return &TypedLiteralExpr{Type: ErrorType{}}
	}
}

// analyzeStructLiteralPositional analyzes a struct/class literal with positional arguments
func (a *Analyzer) analyzeStructLiteralPositional(lit *ast.StructLiteral, fields []StructFieldInfo, typeName string, makeResult func([]TypedExpression) TypedExpression) TypedExpression {
	// Check argument count matches field count
	if len(lit.Arguments) != len(fields) {
		a.addError(
			fmt.Sprintf("type '%s' has %d field(s), but %d argument(s) were provided",
				typeName, len(fields), len(lit.Arguments)),
			lit.LeftBrace, lit.RightBrace,
		)
	}

	// Analyze arguments and check types
	typedArgs := make([]TypedExpression, len(lit.Arguments))
	for i, arg := range lit.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding field
		// Use checkTypeCompatibilityCore to allow nullable coercions (i64 -> i64?, null -> T?)
		if i < len(fields) {
			fieldType := fields[i].Type
			a.checkTypeCompatibilityCore(fieldType, typedArgs[i].GetType(), typedArgs[i], arg.Pos(), contextAssignment)
		}
	}

	return makeResult(typedArgs)
}

// analyzeClassLiteralExprNamed analyzes a class literal with named arguments (e.g., Counter { count: 10 })
func (a *Analyzer) analyzeClassLiteralExprNamed(lit *ast.StructLiteral, classType ClassType) TypedExpression {
	result := a.analyzeNamedLiteral(classType, classType.Name, lit.NamedArguments, lit.LeftBrace, lit.RightBrace)
	return &TypedClassLiteralExpr{
		Type:       classType,
		TypePos:    lit.NamePos,
		LeftBrace:  lit.LeftBrace,
		Args:       result.args,
		RightBrace: lit.RightBrace,
	}
}

// analyzeStructLiteralExprNamed analyzes a struct literal with named arguments (e.g., Point { x: 10, y: 20 })
func (a *Analyzer) analyzeStructLiteralExprNamed(lit *ast.StructLiteral, structType StructType) TypedExpression {
	result := a.analyzeNamedLiteral(structType, structType.Name, lit.NamedArguments, lit.LeftBrace, lit.RightBrace)
	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    lit.NamePos,
		LeftBrace:  lit.LeftBrace,
		Args:       result.args,
		RightBrace: lit.RightBrace,
	}
}

// analyzeAnonStructLiteralWithType analyzes an anonymous struct literal with a known type
// (e.g., val p: Point = { x: 0, y: 0 })
func (a *Analyzer) analyzeAnonStructLiteralWithType(lit *ast.AnonStructLiteral, structType StructType) TypedExpression {
	// Handle named arguments
	if lit.HasNamedArguments() {
		return a.analyzeAnonStructLiteralNamed(lit, structType)
	}

	// Handle positional arguments
	// Check argument count matches field count
	if len(lit.Arguments) != len(structType.Fields) {
		a.addError(
			fmt.Sprintf("struct '%s' has %d field(s), but %d argument(s) were provided",
				structType.Name, len(structType.Fields), len(lit.Arguments)),
			lit.LeftBrace, lit.RightBrace,
		)
	}

	// Analyze arguments and check types
	typedArgs := make([]TypedExpression, len(lit.Arguments))
	for i, arg := range lit.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding field
		// Use checkTypeCompatibilityCore to allow nullable coercions (i64 -> i64?, null -> T?)
		if i < len(structType.Fields) {
			fieldType := structType.Fields[i].Type
			a.checkTypeCompatibilityCore(fieldType, typedArgs[i].GetType(), typedArgs[i], arg.Pos(), contextAssignment)
		}
	}

	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    lit.LeftBrace, // Use left brace position since there's no type name
		LeftBrace:  lit.LeftBrace,
		Args:       typedArgs,
		RightBrace: lit.RightBrace,
	}
}

// analyzeAnonStructLiteralNamed analyzes an anonymous struct literal with named arguments
func (a *Analyzer) analyzeAnonStructLiteralNamed(lit *ast.AnonStructLiteral, structType StructType) TypedExpression {
	result := a.analyzeNamedLiteral(structType, structType.Name, lit.NamedArguments, lit.LeftBrace, lit.RightBrace)
	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    lit.LeftBrace, // Use left brace position since there's no type name
		LeftBrace:  lit.LeftBrace,
		Args:       result.args,
		RightBrace: lit.RightBrace,
	}
}

// analyzeFieldAccessExpr analyzes a field access expression (e.g., p.x, rect.topLeft.x)
func (a *Analyzer) analyzeFieldAccessExpr(expr *ast.FieldAccessExpr) TypedExpression {
	// Check if this is accessing a package namespace (e.g., config.db_port)
	if ident, ok := expr.Object.(*ast.IdentifierExpr); ok {
		if varInfo, found := a.currentScope.lookup(ident.Name); found {
			if nsType, isNs := varInfo.Type.(PackageNamespaceType); isNs {
				return a.analyzeNamespaceFieldAccess(nsType.Namespace, expr)
			}
		}
	}

	// Analyze the object expression
	typedObject := a.analyzeExpression(expr.Object)
	objectType := typedObject.GetType()

	// Auto-dereference owned pointers: Own<StructType> -> StructType
	// This allows p.x where p is Own<Point> to access the Point's x field
	if ownedType, isOwned := objectType.(OwnedPointerType); isOwned {
		objectType = ownedType.ElementType
	}

	// Auto-dereference reference pointers: Ref<StructType> -> StructType
	// This allows p.x where p is Ref<Point>/MutRef<Point> to access the Point's x field
	// Track if we're accessing through a ref for mutability propagation
	var accessingThroughImmutableRef bool
	var accessingThroughMutableRef bool
	if refType, isRef := objectType.(RefPointerType); isRef {
		objectType = refType.ElementType
		accessingThroughImmutableRef = true
	} else if mutRefType, isMutRef := objectType.(MutRefPointerType); isMutRef {
		objectType = mutRefType.ElementType
		accessingThroughMutableRef = true
	}

	// Check that the object is a struct or class type
	var fields []StructFieldInfo
	var typeName string

	if structType, isStruct := objectType.(StructType); isStruct {
		fields = structType.Fields
		typeName = structType.Name
	} else if classType, isClass := objectType.(ClassType); isClass {
		fields = classType.Fields
		typeName = classType.Name
	} else {
		if _, isErr := objectType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("cannot access field '%s' on non-struct type '%s'", expr.Field, objectType.String()),
				expr.Dot, expr.FieldPos,
			)
		}
		return &TypedFieldAccessExpr{
			Type:     TypeError,
			Object:   typedObject,
			Dot:      expr.Dot,
			Field:    expr.Field,
			FieldPos: expr.FieldPos,
			Mutable:  false,
		}
	}

	// Look up the field
	var fieldInfo StructFieldInfo
	var found bool
	for _, f := range fields {
		if f.Name == expr.Field {
			fieldInfo = f
			found = true
			break
		}
	}
	if !found {
		a.addError(
			fmt.Sprintf("type '%s' has no field '%s'", typeName, expr.Field),
			expr.FieldPos, expr.FieldPos,
		)
		return &TypedFieldAccessExpr{
			Type:     TypeError,
			Object:   typedObject,
			Dot:      expr.Dot,
			Field:    expr.Field,
			FieldPos: expr.FieldPos,
			Mutable:  false,
		}
	}

	// Determine the result type and mutability
	resultType := fieldInfo.Type
	resultMutable := fieldInfo.Mutable

	// When accessing through an immutable Ref, apply special rules:
	if accessingThroughImmutableRef {
		// 1. Own<T> fields become Ref<T> (borrowing doesn't transfer ownership, immutable)
		if ownedType, isOwned := fieldInfo.Type.(OwnedPointerType); isOwned {
			resultType = RefPointerType{ElementType: ownedType.ElementType}
		}
		// 2. Field is not mutable through immutable ref
		resultMutable = false
	}

	// When accessing through a mutable MutRef, apply special rules:
	if accessingThroughMutableRef {
		// 1. Own<T> fields become MutRef<T> if field is mutable, else Ref<T>
		if ownedType, isOwned := fieldInfo.Type.(OwnedPointerType); isOwned {
			if fieldInfo.Mutable {
				resultType = MutRefPointerType{ElementType: ownedType.ElementType}
			} else {
				resultType = RefPointerType{ElementType: ownedType.ElementType}
			}
		}
		// 2. Mutability is determined by field's mutability
		resultMutable = fieldInfo.Mutable
	}

	return &TypedFieldAccessExpr{
		Type:     resultType,
		Object:   typedObject,
		Dot:      expr.Dot,
		Field:    expr.Field,
		FieldPos: expr.FieldPos,
		Mutable:  resultMutable,
	}
}

// analyzeMethodCallExpr analyzes a method call expression (e.g., p.copy(), instance.method())
func (a *Analyzer) analyzeMethodCallExpr(expr *ast.MethodCallExpr) TypedExpression {
	// Check if this is a call on a package namespace (e.g., math.add(1, 2))
	if ident, ok := expr.Object.(*ast.IdentifierExpr); ok && !expr.SafeNavigation {
		if varInfo, found := a.currentScope.lookup(ident.Name); found {
			if nsType, isNs := varInfo.Type.(PackageNamespaceType); isNs {
				return a.analyzeNamespaceCall(nsType.Namespace, expr)
			}
		}
	}

	// Check if this is a static method call on a class or object name
	// (Safe navigation doesn't apply to static method calls)
	if ident, ok := expr.Object.(*ast.IdentifierExpr); ok && !expr.SafeNavigation {
		// Check if it's a class name
		if classType, isClass := a.TypeRegistry.LookupClass(ident.Name); isClass {
			return a.analyzeStaticMethodCall(&classType, ident.Name, expr)
		}
		// Check if it's an object name
		if objectType, isObject := a.TypeRegistry.LookupObject(ident.Name); isObject {
			return a.analyzeObjectStaticMethodCall(&objectType, ident.Name, expr)
		}
	}

	// Analyze the object expression
	typedObject := a.analyzeExpression(expr.Object)
	objectType := typedObject.GetType()

	// Handle safe navigation: receiver must be nullable, unwrap it for method resolution
	unwrappedType := objectType
	if expr.SafeNavigation {
		if nullableType, isNullable := objectType.(NullableType); isNullable {
			unwrappedType = nullableType.InnerType
		} else if _, isErr := objectType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("safe navigation '?.' can only be used on nullable types, got '%s'", objectType.String()),
				expr.Dot, expr.Dot,
			)
			// Still type the arguments to avoid nil TypedExpressions
			typedArgs := make([]TypedExpression, len(expr.Arguments))
			for i, arg := range expr.Arguments {
				typedArgs[i] = a.analyzeExpression(arg)
			}
			return &TypedMethodCallExpr{
				Type:           TypeError,
				Object:         typedObject,
				Dot:            expr.Dot,
				Method:         expr.Method,
				MethodPos:      expr.MethodPos,
				LeftParen:      expr.LeftParen,
				Arguments:      typedArgs,
				RightParen:     expr.RightParen,
				SafeNavigation: true,
			}
		}
	}

	// Type arguments
	typedArgs := make([]TypedExpression, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)
	}

	// Check if this is a .copy() call on an owned pointer
	if expr.Method == "copy" {
		if ownedType, isOwned := unwrappedType.(OwnedPointerType); isOwned {
			// .copy() on Own<T> returns a new Own<T> (deep copy)
			if len(expr.Arguments) != 0 {
				a.addError(
					fmt.Sprintf("copy() takes no arguments, got %d", len(expr.Arguments)),
					expr.LeftParen, expr.RightParen,
				)
			}
			resultType := Type(ownedType)
			// For safe navigation, wrap result in nullable
			if expr.SafeNavigation {
				resultType = NullableType{InnerType: ownedType}
			}
			return &TypedMethodCallExpr{
				Type:           resultType,
				Object:         typedObject,
				Dot:            expr.Dot,
				Method:         expr.Method,
				MethodPos:      expr.MethodPos,
				LeftParen:      expr.LeftParen,
				Arguments:      typedArgs,
				RightParen:     expr.RightParen,
				SafeNavigation: expr.SafeNavigation,
			}
		}
	}

	// Check if this is an instance method call on a class type
	if result := a.tryAnalyzeInstanceMethodCall(typedObject, unwrappedType, expr, typedArgs); result != nil {
		return result
	}

	// Unknown method call
	if _, isErr := unwrappedType.(ErrorType); !isErr {
		a.addError(
			fmt.Sprintf("unknown method '%s' on type '%s'", expr.Method, unwrappedType.String()),
			expr.MethodPos, expr.MethodPos,
		)
	}

	return &TypedMethodCallExpr{
		Type:           TypeError,
		Object:         typedObject,
		Dot:            expr.Dot,
		Method:         expr.Method,
		MethodPos:      expr.MethodPos,
		LeftParen:      expr.LeftParen,
		Arguments:      typedArgs,
		RightParen:     expr.RightParen,
		SafeNavigation: expr.SafeNavigation,
	}
}

// analyzeNamespaceCall analyzes a function call on a package namespace (e.g., math.add(1, 2)).
func (a *Analyzer) analyzeNamespaceCall(ns *PackageNamespace, expr *ast.MethodCallExpr) TypedExpression {
	// Type arguments
	typedArgs := make([]TypedExpression, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)
	}

	// Look up the function in the package's exports
	export, found := ns.Exports[expr.Method]
	if !found {
		a.addError(
			fmt.Sprintf("package '%s' has no declaration '%s'", ns.Path, expr.Method),
			expr.MethodPos, expr.MethodPos,
		)
		return &TypedCallExpr{
			Type:       TypeError,
			Name:       ns.Path + "." + expr.Method,
			NamePos:    expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}
	}

	// Must be a function export
	fnType, isFn := export.Type.(FunctionType)
	if !isFn {
		a.addError(
			fmt.Sprintf("'%s.%s' is not a function", ns.Path, expr.Method),
			expr.MethodPos, expr.MethodPos,
		)
		return &TypedCallExpr{
			Type:       TypeError,
			Name:       ns.Path + "." + expr.Method,
			NamePos:    expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}
	}

	// Check argument count
	if len(typedArgs) != len(fnType.ParamTypes) {
		a.addError(
			fmt.Sprintf("function '%s.%s' expects %d arguments, got %d", ns.Path, expr.Method, len(fnType.ParamTypes), len(typedArgs)),
			expr.LeftParen, expr.RightParen,
		)
	} else {
		// Check argument types
		for i, arg := range typedArgs {
			if !arg.GetType().Equals(fnType.ParamTypes[i]) {
				if _, isErr := arg.GetType().(ErrorType); !isErr {
					a.addError(
						fmt.Sprintf("argument %d to '%s.%s': expected '%s', got '%s'", i+1, ns.Path, expr.Method, fnType.ParamTypes[i].String(), arg.GetType().String()),
						arg.Pos(), arg.End(),
					)
				}
			}
		}
	}

	returnType := fnType.ReturnType
	if returnType == nil {
		returnType = TypeVoid
	}

	return &TypedCallExpr{
		Type:       returnType,
		Name:       ns.Path + "." + expr.Method,
		NamePos:    expr.MethodPos,
		LeftParen:  expr.LeftParen,
		Arguments:  typedArgs,
		RightParen: expr.RightParen,
	}
}

// analyzeStaticMethodCall analyzes a static method call on a class (e.g., ClassName.method())
func (a *Analyzer) analyzeStaticMethodCall(classType *ClassType, className string, expr *ast.MethodCallExpr) TypedExpression {
	// Type arguments
	typedArgs := make([]TypedExpression, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)
	}

	// Create a typed identifier for the class name
	typedObject := &TypedIdentifierExpr{
		Type:     *classType,
		Name:     className,
		StartPos: expr.Object.Pos(),
		EndPos:   expr.Object.End(),
	}

	// Look up the method
	methodInfos, found := classType.Methods[expr.Method]
	if !found || len(methodInfos) == 0 {
		a.addError(
			fmt.Sprintf("undefined method '%s' on class '%s'", expr.Method, className),
			expr.MethodPos, expr.MethodPos,
		)
		return &TypedMethodCallExpr{
			Type:       TypeError,
			Object:     typedObject,
			Dot:        expr.Dot,
			Method:     expr.Method,
			MethodPos:  expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}
	}

	// Resolve the best static overload
	methodInfo := a.resolveStaticOverload(methodInfos, typedArgs, expr.Method, className, expr.MethodPos)
	if methodInfo == nil {
		// Error already reported by resolveStaticOverload
		return &TypedMethodCallExpr{
			Type:       TypeError,
			Object:     typedObject,
			Dot:        expr.Dot,
			Method:     expr.Method,
			MethodPos:  expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}
	}

	return &TypedMethodCallExpr{
		Type:           methodInfo.ReturnType,
		Object:         typedObject,
		Dot:            expr.Dot,
		Method:         expr.Method,
		MethodPos:      expr.MethodPos,
		LeftParen:      expr.LeftParen,
		Arguments:      typedArgs,
		RightParen:     expr.RightParen,
		ResolvedMethod: methodInfo,
	}
}

// analyzeObjectStaticMethodCall analyzes a static method call on an object (e.g., Math.max())
func (a *Analyzer) analyzeObjectStaticMethodCall(objectType *ObjectType, objectName string, expr *ast.MethodCallExpr) TypedExpression {
	// Type arguments
	typedArgs := make([]TypedExpression, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)
	}

	// Create a typed identifier for the object name
	typedObject := &TypedIdentifierExpr{
		Type:     *objectType,
		Name:     objectName,
		StartPos: expr.Object.Pos(),
		EndPos:   expr.Object.End(),
	}

	// Look up the method
	methodInfos, found := objectType.Methods[expr.Method]
	if !found || len(methodInfos) == 0 {
		a.addError(
			fmt.Sprintf("undefined method '%s' on object '%s'", expr.Method, objectName),
			expr.MethodPos, expr.MethodPos,
		)
		return &TypedMethodCallExpr{
			Type:       TypeError,
			Object:     typedObject,
			Dot:        expr.Dot,
			Method:     expr.Method,
			MethodPos:  expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}
	}

	// Resolve the best overload (object methods are always static)
	methodInfo := a.resolveObjectOverload(methodInfos, typedArgs, expr.Method, objectName, expr.MethodPos)
	if methodInfo == nil {
		// Error already reported by resolveObjectOverload
		return &TypedMethodCallExpr{
			Type:       TypeError,
			Object:     typedObject,
			Dot:        expr.Dot,
			Method:     expr.Method,
			MethodPos:  expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}
	}

	return &TypedMethodCallExpr{
		Type:           methodInfo.ReturnType,
		Object:         typedObject,
		Dot:            expr.Dot,
		Method:         expr.Method,
		MethodPos:      expr.MethodPos,
		LeftParen:      expr.LeftParen,
		Arguments:      typedArgs,
		RightParen:     expr.RightParen,
		ResolvedMethod: methodInfo,
	}
}

// tryAnalyzeInstanceMethodCall attempts to analyze an instance method call
// Returns nil if the object type doesn't support instance methods
func (a *Analyzer) tryAnalyzeInstanceMethodCall(typedObject TypedExpression, objectType Type, expr *ast.MethodCallExpr, typedArgs []TypedExpression) TypedExpression {
	// Get the class type from the object type
	var classType *ClassType

	// Direct class type (stack-allocated instance)
	if ct, ok := objectType.(ClassType); ok {
		classType = &ct
	}

	// Owned pointer to class
	if owned, ok := objectType.(OwnedPointerType); ok {
		if ct, ok := owned.ElementType.(ClassType); ok {
			classType = &ct
		}
	}

	// Immutable reference to class
	if ref, ok := objectType.(RefPointerType); ok {
		if ct, ok := ref.ElementType.(ClassType); ok {
			classType = &ct
		}
	}

	// Mutable reference to class
	if mutRef, ok := objectType.(MutRefPointerType); ok {
		if ct, ok := mutRef.ElementType.(ClassType); ok {
			classType = &ct
		}
	}

	if classType == nil {
		return nil // Not a class type, let caller handle it
	}

	// Look up the method
	methodInfos, found := classType.Methods[expr.Method]
	if !found || len(methodInfos) == 0 {
		a.addError(
			fmt.Sprintf("undefined method '%s' on class '%s'", expr.Method, classType.Name),
			expr.MethodPos, expr.MethodPos,
		)
		return &TypedMethodCallExpr{
			Type:           TypeError,
			Object:         typedObject,
			Dot:            expr.Dot,
			Method:         expr.Method,
			MethodPos:      expr.MethodPos,
			LeftParen:      expr.LeftParen,
			Arguments:      typedArgs,
			RightParen:     expr.RightParen,
			SafeNavigation: expr.SafeNavigation,
		}
	}

	// Resolve the best instance overload
	methodInfo := a.resolveInstanceOverload(methodInfos, typedArgs, objectType, expr.Method, classType.Name, expr.MethodPos)
	if methodInfo == nil {
		// Error already reported by resolveInstanceOverload
		return &TypedMethodCallExpr{
			Type:           TypeError,
			Object:         typedObject,
			Dot:            expr.Dot,
			Method:         expr.Method,
			MethodPos:      expr.MethodPos,
			LeftParen:      expr.LeftParen,
			Arguments:      typedArgs,
			RightParen:     expr.RightParen,
			SafeNavigation: expr.SafeNavigation,
		}
	}

	// Check that the receiver type is compatible with the self parameter type
	selfType := methodInfo.ParamTypes[0]
	if !a.checkReceiverCompatibility(objectType, selfType, expr.Object.Pos()) {
		// Error already reported by checkReceiverCompatibility
	}

	// Determine result type: for safe navigation, wrap non-void return types in nullable
	resultType := methodInfo.ReturnType
	if expr.SafeNavigation {
		// For safe navigation, if method returns a non-void type, wrap in nullable
		if _, isVoid := resultType.(VoidType); !isVoid {
			resultType = NullableType{InnerType: resultType}
		}
	}

	return &TypedMethodCallExpr{
		Type:           resultType,
		Object:         typedObject,
		Dot:            expr.Dot,
		Method:         expr.Method,
		MethodPos:      expr.MethodPos,
		LeftParen:      expr.LeftParen,
		Arguments:      typedArgs,
		RightParen:     expr.RightParen,
		ResolvedMethod: methodInfo,
		SafeNavigation: expr.SafeNavigation,
	}
}

// checkReceiverCompatibility checks if an object type is compatible with a self parameter type
func (a *Analyzer) checkReceiverCompatibility(objectType Type, selfType Type, pos ast.Position) bool {
	// Get the class from the self type
	var selfClassType ClassType
	var isOwnedRef bool

	switch st := selfType.(type) {
	case RefPointerType:
		if ct, ok := st.ElementType.(ClassType); ok {
			selfClassType = ct
		}
	case MutRefPointerType:
		// Mutable reference - auto-borrow logic applies
		if ct, ok := st.ElementType.(ClassType); ok {
			selfClassType = ct
		}
	case OwnedPointerType:
		isOwnedRef = true
		if ct, ok := st.ElementType.(ClassType); ok {
			selfClassType = ct
		}
	default:
		return false
	}

	// For owned reference methods, the caller must pass an owned pointer
	if isOwnedRef {
		if _, isOwned := objectType.(OwnedPointerType); !isOwned {
			a.addError(
				fmt.Sprintf("method requires ownership (*%s), but receiver is '%s'", selfClassType.Name, objectType.String()),
				pos, pos,
			).WithHint("use 'new' to create an owned instance")
			return false
		}
	}

	// For mutable reference methods, we need auto-borrow logic
	// Stack values and owned pointers can be auto-borrowed
	// (This is a simplified check - full implementation would track borrows)

	return true
}

// overloadCallKind distinguishes between different method call contexts for overload resolution
type overloadCallKind int

const (
	overloadCallStatic   overloadCallKind = iota // static method call on class
	overloadCallInstance                         // instance method call on class
	overloadCallObject                           // method call on object (always static)
)

// resolveStaticOverload finds the best matching static method overload for the given arguments.
// Returns the resolved method, or nil if no match is found (error already reported).
func (a *Analyzer) resolveStaticOverload(methodInfos []*MethodInfo, typedArgs []TypedExpression, methodName, className string, pos ast.Position) *MethodInfo {
	return a.resolveOverloadCore(methodInfos, typedArgs, nil, methodName, className, pos, overloadCallStatic)
}

// resolveInstanceOverload finds the best matching instance method overload for the given arguments.
// Returns the resolved method, or nil if no match is found (error already reported).
func (a *Analyzer) resolveInstanceOverload(methodInfos []*MethodInfo, typedArgs []TypedExpression, objectType Type, methodName, className string, pos ast.Position) *MethodInfo {
	return a.resolveOverloadCore(methodInfos, typedArgs, objectType, methodName, className, pos, overloadCallInstance)
}

// resolveObjectOverload finds the best matching method overload for an object (all methods are static).
// Returns the resolved method, or nil if no match is found (error already reported).
func (a *Analyzer) resolveObjectOverload(methodInfos []*MethodInfo, typedArgs []TypedExpression, methodName, objectName string, pos ast.Position) *MethodInfo {
	return a.resolveOverloadCore(methodInfos, typedArgs, nil, methodName, objectName, pos, overloadCallObject)
}

// resolveOverloadCore is the shared implementation for overload resolution.
// objectType is only used for instance method calls (to check receiver compatibility).
func (a *Analyzer) resolveOverloadCore(methodInfos []*MethodInfo, typedArgs []TypedExpression, objectType Type, methodName, ownerName string, pos ast.Position, callKind overloadCallKind) *MethodInfo {
	// Determine self offset based on call kind
	selfOffset := 0
	if callKind == overloadCallInstance {
		selfOffset = 1
	}

	// Collect all applicable overloads
	var applicable []*MethodInfo
	for _, mi := range methodInfos {
		switch callKind {
		case overloadCallStatic:
			// Only consider static methods
			if !mi.IsStatic {
				continue
			}
		case overloadCallInstance:
			// Only consider instance methods with compatible receiver
			if mi.IsStatic {
				continue
			}
			if len(mi.ParamTypes) == 0 || !a.isReceiverCompatible(objectType, mi.ParamTypes[0]) {
				continue
			}
		case overloadCallObject:
			// Object methods are all static, no filtering needed
		}

		if a.isOverloadApplicable(mi, typedArgs, selfOffset) {
			applicable = append(applicable, mi)
		}
	}

	if len(applicable) == 0 {
		// No applicable overload found - report best error
		a.reportNoApplicableOverloadCore(methodInfos, typedArgs, methodName, ownerName, pos, callKind)
		return nil
	}

	if len(applicable) == 1 {
		return applicable[0]
	}

	// Multiple applicable - find most specific
	return a.selectMostSpecificCore(applicable, selfOffset, methodName, ownerName, pos, callKind)
}

// selectMostSpecificCore selects the most specific overload from multiple applicable overloads.
// Returns the most specific method, or reports ambiguity and returns nil.
func (a *Analyzer) selectMostSpecificCore(applicable []*MethodInfo, selfOffset int, methodName, ownerName string, pos ast.Position, callKind overloadCallKind) *MethodInfo {
	if len(applicable) == 0 {
		return nil
	}

	// Find the most specific overload
	best := applicable[0]
	for i := 1; i < len(applicable); i++ {
		candidate := applicable[i]
		if a.isMoreSpecificThan(candidate, best, selfOffset) {
			best = candidate
		}
	}

	// Verify that 'best' is more specific than all others
	for _, other := range applicable {
		if other == best {
			continue
		}
		if !a.isMoreSpecificThan(best, other, selfOffset) {
			// Ambiguous overload - format error message based on call kind
			var msg string
			if callKind == overloadCallObject {
				msg = fmt.Sprintf("ambiguous method call '%s' on object '%s': multiple overloads match", methodName, ownerName)
			} else {
				msg = fmt.Sprintf("ambiguous method call '%s' on '%s': multiple overloads match", methodName, ownerName)
			}
			a.addError(msg, pos, pos).WithHint("provide explicit types to disambiguate")
			return nil
		}
	}

	return best
}

// reportNoApplicableOverloadCore reports an error when no overload matches the arguments.
func (a *Analyzer) reportNoApplicableOverloadCore(methodInfos []*MethodInfo, typedArgs []TypedExpression, methodName, ownerName string, pos ast.Position, callKind overloadCallKind) {
	// For class methods, filter by static/instance kind
	var candidates []*MethodInfo
	selfOffset := 0

	switch callKind {
	case overloadCallStatic:
		for _, mi := range methodInfos {
			if mi.IsStatic {
				candidates = append(candidates, mi)
			}
		}
	case overloadCallInstance:
		selfOffset = 1
		for _, mi := range methodInfos {
			if !mi.IsStatic {
				candidates = append(candidates, mi)
			}
		}
	case overloadCallObject:
		candidates = methodInfos // All object methods are static
	}

	// Check for static/instance mismatch (only for class methods)
	if callKind != overloadCallObject && len(candidates) == 0 {
		if callKind == overloadCallStatic {
			a.addError(
				fmt.Sprintf("method '%s' on class '%s' is not a static method", methodName, ownerName),
				pos, pos,
			).WithHint("static methods are called on the class name, instance methods on an instance")
		} else {
			a.addError(
				fmt.Sprintf("method '%s' on class '%s' is a static method", methodName, ownerName),
				pos, pos,
			).WithHint("instance methods require 'self' as first parameter; call static methods on the class name")
		}
		return
	}

	// Check if it's an argument count issue
	expectedCounts := make(map[int]bool)
	for _, mi := range candidates {
		expectedCounts[len(mi.ParamTypes)-selfOffset] = true
	}

	// Format owner type label
	ownerLabel := ownerName
	if callKind == overloadCallObject {
		ownerLabel = "object '" + ownerName + "'"
	}

	if len(expectedCounts) == 1 {
		// All overloads have the same arg count
		var expected int
		for c := range expectedCounts {
			expected = c
		}
		if len(typedArgs) != expected {
			a.addError(
				fmt.Sprintf("method '%s' expects %d argument(s), got %d", methodName, expected, len(typedArgs)),
				pos, pos,
			)
		} else {
			// Same arg count but type mismatch
			a.addError(
				fmt.Sprintf("no matching overload for method '%s' on %s", methodName, ownerLabel),
				pos, pos,
			).WithHint("check argument types")
		}
	} else {
		// Multiple arg counts possible
		a.addError(
			fmt.Sprintf("no matching overload for method '%s' on %s: got %d argument(s)", methodName, ownerLabel, len(typedArgs)),
			pos, pos,
		)
	}
}

// isReceiverCompatible checks if an object type is compatible with a self parameter type
// without reporting errors (unlike checkReceiverCompatibility)
func (a *Analyzer) isReceiverCompatible(objectType Type, selfType Type) bool {
	// For owned pointers, check if object is also owned
	if _, isSelfOwned := selfType.(OwnedPointerType); isSelfOwned {
		_, isObjOwned := objectType.(OwnedPointerType)
		return isObjOwned
	}

	// For ref/mutref pointers, owned pointers can auto-borrow
	return true
}

// isOverloadApplicable checks if a method can be called with the given arguments.
// selfOffset is 0 for static methods, 1 for instance methods (to skip self).
func (a *Analyzer) isOverloadApplicable(mi *MethodInfo, typedArgs []TypedExpression, selfOffset int) bool {
	expectedArgs := len(mi.ParamTypes) - selfOffset
	if len(typedArgs) != expectedArgs {
		return false
	}

	for i := 0; i < len(typedArgs); i++ {
		argType := typedArgs[i].GetType()
		paramType := mi.ParamTypes[i+selfOffset]
		if !IsAssignableTo(argType, paramType) {
			return false
		}
	}

	return true
}

// isMoreSpecificThan returns true if m1 is more specific than m2.
// A method is more specific if all its parameter types are at least as specific,
// and at least one is strictly more specific.
// Specificity rules: non-nullable > nullable, exact type > compatible type
func (a *Analyzer) isMoreSpecificThan(m1, m2 *MethodInfo, selfOffset int) bool {
	if len(m1.ParamTypes) != len(m2.ParamTypes) {
		return false // Different arity, can't compare
	}

	hasStrictlyMore := false
	for i := selfOffset; i < len(m1.ParamTypes); i++ {
		t1 := m1.ParamTypes[i]
		t2 := m2.ParamTypes[i]

		cmp := a.compareTypeSpecificity(t1, t2)
		if cmp < 0 {
			return false // t1 is less specific, m1 is not more specific
		}
		if cmp > 0 {
			hasStrictlyMore = true
		}
	}

	return hasStrictlyMore
}

// compareTypeSpecificity compares two types for specificity.
// Returns: 1 if t1 is more specific, -1 if t2 is more specific, 0 if equal.
func (a *Analyzer) compareTypeSpecificity(t1, t2 Type) int {
	// Non-nullable is more specific than nullable
	t1Nullable, t1IsNullable := t1.(NullableType)
	t2Nullable, t2IsNullable := t2.(NullableType)

	if !t1IsNullable && t2IsNullable {
		return 1 // t1 is more specific (non-nullable vs nullable)
	}
	if t1IsNullable && !t2IsNullable {
		return -1 // t2 is more specific
	}
	if t1IsNullable && t2IsNullable {
		// Both nullable, compare inner types
		return a.compareTypeSpecificity(t1Nullable.InnerType, t2Nullable.InnerType)
	}

	// If types are equal, return 0
	if t1.Equals(t2) {
		return 0
	}

	// Different non-nullable types - no clear specificity ordering
	return 0
}

// analyzeNewExpr analyzes a 'new' expression (e.g., new Point{ 10, 20 })
func (a *Analyzer) analyzeNewExpr(expr *ast.NewExpr) TypedExpression {
	typedOperand := a.analyzeExpression(expr.Operand)

	operandType := typedOperand.GetType()
	if _, isErr := operandType.(ErrorType); isErr {
		return &TypedNewExpr{
			Type:    TypeError,
			NewPos:  expr.NewPos,
			Operand: typedOperand,
		}
	}

	resultType := OwnedPointerType{ElementType: operandType}
	return &TypedNewExpr{
		Type:    resultType,
		NewPos:  expr.NewPos,
		Operand: typedOperand,
	}
}

// analyzeSafeCallExpr analyzes a safe call expression (e.g., person?.address)
func (a *Analyzer) analyzeSafeCallExpr(expr *ast.SafeCallExpr) TypedExpression {
	// Analyze the object expression
	typedObject := a.analyzeExpression(expr.Object)
	objectType := typedObject.GetType()

	// Object must be nullable
	innerType, isNullable := UnwrapNullable(objectType)
	if !isNullable {
		if _, isErr := objectType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("safe call '?.' used on non-nullable type '%s'", objectType.String()),
				expr.SafeCallPos, expr.SafeCallPos,
			).WithHint("use '.' for non-nullable types")
		}
		return &TypedSafeCallExpr{
			Type:           TypeError,
			Object:         typedObject,
			SafeCallPos:    expr.SafeCallPos,
			Field:          expr.Field,
			FieldPos:       expr.FieldPos,
			FieldOffset:    0,
			InnerType:      TypeError,
			ThroughPointer: false,
		}
	}

	// Auto-dereference owned/ref pointers: Own<T>? or Ref<T>? -> access T's fields
	// This allows node?.value where node is Own<Node>? to work
	throughPointer := false
	if ownedType, isOwned := innerType.(OwnedPointerType); isOwned {
		innerType = ownedType.ElementType
		throughPointer = true
	} else if refType, isRef := innerType.(RefPointerType); isRef {
		innerType = refType.ElementType
		throughPointer = true
	} else if mutRefType, isMutRef := innerType.(MutRefPointerType); isMutRef {
		innerType = mutRefType.ElementType
		throughPointer = true
	}

	// Re-lookup struct from registry to get the version with resolved fields
	// This is needed because pointer types may store a stale copy from before fields were resolved
	if st, isStruct := innerType.(StructType); isStruct {
		if resolved, ok := a.TypeRegistry.LookupStruct(st.Name); ok {
			innerType = resolved
		}
	}

	// The inner type must be a struct
	structType, isStruct := innerType.(StructType)
	if !isStruct {
		if _, isErr := innerType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("cannot access field '%s' on non-struct type '%s'", expr.Field, innerType.String()),
				expr.SafeCallPos, expr.FieldPos,
			)
		}
		return &TypedSafeCallExpr{
			Type:           TypeError,
			Object:         typedObject,
			SafeCallPos:    expr.SafeCallPos,
			Field:          expr.Field,
			FieldPos:       expr.FieldPos,
			FieldOffset:    0,
			InnerType:      innerType,
			ThroughPointer: throughPointer,
		}
	}

	// Look up the field
	fieldInfo, found := structType.GetField(expr.Field)
	if !found {
		a.addError(
			fmt.Sprintf("struct '%s' has no field '%s'", structType.Name, expr.Field),
			expr.FieldPos, expr.FieldPos,
		)
		return &TypedSafeCallExpr{
			Type:           TypeError,
			Object:         typedObject,
			SafeCallPos:    expr.SafeCallPos,
			Field:          expr.Field,
			FieldPos:       expr.FieldPos,
			FieldOffset:    0,
			InnerType:      innerType,
			ThroughPointer: throughPointer,
		}
	}

	// Result type is always nullable (field value or null)
	resultType := MakeNullable(fieldInfo.Type)
	fieldOffset := structType.FieldOffset(expr.Field)

	return &TypedSafeCallExpr{
		Type:           resultType,
		Object:         typedObject,
		SafeCallPos:    expr.SafeCallPos,
		Field:          expr.Field,
		FieldPos:       expr.FieldPos,
		FieldOffset:    fieldOffset,
		InnerType:      innerType,
		ThroughPointer: throughPointer,
	}
}

// isAcceptedType checks if argType matches any of the accepted types
func isAcceptedType(argType Type, acceptedTypes []Type) bool {
	for _, accepted := range acceptedTypes {
		if accepted.Equals(argType) {
			return true
		}
		// Also allow compatible integer types when s64 is accepted
		if _, isS64 := accepted.(S64Type); isS64 && IsIntegerType(argType) {
			return true
		}
	}
	return false
}

// formatAcceptedTypes returns a human-readable list of accepted types
func formatAcceptedTypes(types []Type) string {
	if len(types) == 0 {
		return "none"
	}
	if len(types) == 1 {
		return types[0].String()
	}
	result := ""
	for i, t := range types {
		if i > 0 {
			if i == len(types)-1 {
				result += " or "
			} else {
				result += ", "
			}
		}
		result += t.String()
	}
	return result
}

// isCompatibleIntegerType checks if argType can be passed to paramType
// This allows generic integer literals to be passed to sized integer parameters
func isCompatibleIntegerType(paramType, argType Type) bool {
	// If param expects s64, allow any integer type
	if _, isS64 := paramType.(S64Type); isS64 {
		return IsIntegerType(argType)
	}
	return false
}

// analyzeIdentifier analyzes an identifier (variable reference)
func (a *Analyzer) analyzeIdentifier(ident *ast.IdentifierExpr) TypedExpression {
	// Look up the variable in the current scope
	info, found := a.currentScope.lookup(ident.Name)
	var typ Type
	if !found {
		a.addError(
			fmt.Sprintf("undefined variable '%s'", ident.Name),
			ident.StartPos, ident.EndPos,
		).WithHint("did you forget to declare it with 'val' or 'var'?")
		typ = TypeError
	} else {
		typ = info.Type
		// Check for package namespace misuse — namespaces can only be used with dot-access
		if nsType, isNs := typ.(PackageNamespaceType); isNs {
			a.addError(
				fmt.Sprintf("cannot use package '%s' as a value", nsType.Namespace.Path),
				ident.StartPos, ident.EndPos,
			).WithHint(fmt.Sprintf("use '%s.<name>' to access a declaration from the package", ident.Name))
			return &TypedIdentifierExpr{
				Type:     TypeError,
				Name:     ident.Name,
				StartPos: ident.StartPos,
				EndPos:   ident.EndPos,
			}
		}
		// Check for use-after-move
		if ownerInfo, tracked := a.ownershipScope.lookup(ident.Name); tracked {
			if ownerInfo.State == StateMoved {
				err := a.addError(
					fmt.Sprintf("use of moved value '%s'", ident.Name),
					ident.StartPos, ident.EndPos,
				)
				if ownerInfo.MoveInfo.MovedTo != "" {
					if ownerInfo.MoveInfo.MovedTo == "<param>" {
						err.WithHint("value was moved when passed as function argument")
					} else {
						err.WithHint(fmt.Sprintf("value was moved to '%s'", ownerInfo.MoveInfo.MovedTo))
					}
				}
			}
		}
	}

	return &TypedIdentifierExpr{
		Type:     typ,
		Name:     ident.Name,
		StartPos: ident.StartPos,
		EndPos:   ident.EndPos,
	}
}

// analyzeSelfExpr analyzes the 'self' expression within a method body
func (a *Analyzer) analyzeSelfExpr(self *ast.SelfExpr) TypedExpression {
	// 'self' is only valid within a method body
	if a.currentClass == nil {
		a.addError("'self' can only be used within a method body", self.SelfPos, self.SelfPos)
		return &TypedSelfExpr{
			Type:    TypeError,
			SelfPos: self.SelfPos,
		}
	}

	// Look up 'self' in the current scope (it was added as a parameter)
	info, found := a.currentScope.lookup("self")
	if !found {
		// This shouldn't happen if we're in a class method that has 'self' param
		a.addError("'self' is not available in this context", self.SelfPos, self.SelfPos)
		return &TypedSelfExpr{
			Type:    TypeError,
			SelfPos: self.SelfPos,
		}
	}

	return &TypedSelfExpr{
		Type:    info.Type,
		SelfPos: self.SelfPos,
	}
}

// checkAndRecordMove checks if an expression is a variable being moved and records the move
func (a *Analyzer) checkAndRecordMove(expr ast.Expression, movedTo string, location ast.Position) {
	if expr == nil {
		return
	}

	// Handle identifier expressions - direct variable moves
	if ident, ok := expr.(*ast.IdentifierExpr); ok {
		// Look up the variable type to see if it's move-only
		info, found := a.currentScope.lookup(ident.Name)
		if found && IsMoveOnly(info.Type) {
			// Check if already moved
			if ownerInfo, tracked := a.ownershipScope.lookup(ident.Name); tracked {
				if ownerInfo.State == StateMoved {
					// Already reported as use-after-move in analyzeIdentifier
					return
				}
			}

			// Check if we're inside a loop - moves inside loops are not allowed
			// because the loop might execute multiple times, causing double-move
			if a.ownershipScope.isInLoop() {
				a.addError(
					fmt.Sprintf("cannot move '%s' inside a loop", ident.Name),
					ident.StartPos, ident.EndPos,
				).WithHint("moves inside loops would cause double-move on second iteration; consider using .copy() or restructuring")
				return
			}

			// Mark as moved
			a.ownershipScope.markMoved(ident.Name, movedTo, location)
		}
		return
	}

	// Handle field access - moving a field out of a struct
	// For now, we don't allow moving fields out of structs
	// (would require more complex tracking)
}

// analyzeLiteral analyzes a literal expression
func (a *Analyzer) analyzeLiteral(lit *ast.LiteralExpr) TypedExpression {
	var typ Type
	switch lit.Kind {
	case ast.LiteralTypeInteger:
		typ = TypeInteger
	case ast.LiteralTypeFloat:
		typ = TypeFloat64 // default float type is f64
	case ast.LiteralTypeString:
		typ = TypeString
	case ast.LiteralTypeBoolean:
		typ = TypeBoolean
	case ast.LiteralTypeNull:
		typ = TypeNothing // null has type Nothing, assignable to any T?
	default:
		a.addError(fmt.Sprintf("unknown literal type: %v", lit.Kind), lit.StartPos, lit.EndPos)
		typ = TypeError
	}

	return &TypedLiteralExpr{
		Type:     typ,
		LitType:  lit.Kind,
		Value:    lit.Value,
		StartPos: lit.StartPos,
		EndPos:   lit.EndPos,
	}
}

// analyzeBinaryExpression analyzes a binary expression
func (a *Analyzer) analyzeBinaryExpression(expr *ast.BinaryExpr) TypedExpression {
	// Short-circuit operators (&&, ||) need special handling for ownership:
	// The right operand might not be evaluated, so moves in the right operand
	// are conditional and should be treated like moves in an if branch.
	if expr.Op == "&&" || expr.Op == "||" {
		return a.analyzeShortCircuitExpression(expr)
	}

	// Analyze left and right operands
	left := a.analyzeExpression(expr.Left)
	right := a.analyzeExpression(expr.Right)

	leftType := left.GetType()
	rightType := right.GetType()

	// Determine the result type and check type compatibility
	resultType := a.checkBinaryOperation(expr.Op, leftType, rightType, expr.LeftPos, expr.RightPos)

	return &TypedBinaryExpr{
		Type:     resultType,
		Left:     left,
		Op:       expr.Op,
		Right:    right,
		LeftPos:  expr.LeftPos,
		OpPos:    expr.OpPos,
		RightPos: expr.RightPos,
	}
}

// analyzeShortCircuitExpression handles && and || with proper ownership tracking.
// The right operand of short-circuit operators may not be evaluated:
// - For &&: right only evaluates if left is true
// - For ||: right only evaluates if left is false
// Any moves in the right operand should be treated as conditional moves.
func (a *Analyzer) analyzeShortCircuitExpression(expr *ast.BinaryExpr) TypedExpression {
	// Analyze left operand (always evaluated)
	left := a.analyzeExpression(expr.Left)
	leftType := left.GetType()

	// Snapshot ownership state before evaluating right operand
	beforeSnapshot := a.snapshotOwnershipState()

	// Analyze right operand (conditionally evaluated)
	right := a.analyzeExpression(expr.Right)
	rightType := right.GetType()

	// Collect any moves that occurred in the right operand
	rightMoves := a.collectBranchMoves(beforeSnapshot)

	// If there were moves in the right operand, they are conditional.
	// We treat this like an if statement where one branch has the moves
	// and the other branch has no moves (short-circuit case).
	// Result: the variable is considered "possibly moved" after the expression.
	if len(rightMoves) > 0 {
		// Merge with empty moves from the "short-circuit" branch
		var emptyMoves []moveRecord
		a.mergeConditionalMoves(rightMoves, emptyMoves)
	}

	// Determine the result type and check type compatibility
	resultType := a.checkBinaryOperation(expr.Op, leftType, rightType, expr.LeftPos, expr.RightPos)

	return &TypedBinaryExpr{
		Type:     resultType,
		Left:     left,
		Op:       expr.Op,
		Right:    right,
		LeftPos:  expr.LeftPos,
		OpPos:    expr.OpPos,
		RightPos: expr.RightPos,
	}
}

// analyzeUnaryExpression analyzes a unary expression (e.g., !x)
func (a *Analyzer) analyzeUnaryExpression(expr *ast.UnaryExpr) TypedExpression {
	operand := a.analyzeExpression(expr.Operand)
	operandType := operand.GetType()

	if expr.Op == "!" {
		// ! requires boolean operand
		if _, isBool := operandType.(BooleanType); !isBool {
			if _, isErr := operandType.(ErrorType); !isErr {
				a.addError(
					fmt.Sprintf("operator '!' requires boolean operand, got '%s'", operandType.String()),
					expr.OperandPos, expr.OperandEnd,
				).WithHint("logical NOT only works with boolean values")
			}
			return &TypedUnaryExpr{
				Type:       TypeError,
				Op:         expr.Op,
				Operand:    operand,
				OpPos:      expr.OpPos,
				OperandEnd: expr.OperandEnd,
			}
		}

		return &TypedUnaryExpr{
			Type:       TypeBoolean,
			Op:         expr.Op,
			Operand:    operand,
			OpPos:      expr.OpPos,
			OperandEnd: expr.OperandEnd,
		}
	}

	if expr.Op == "-" {
		// Unary minus requires integer operand
		if !IsIntegerType(operandType) {
			if _, isErr := operandType.(ErrorType); !isErr {
				a.addError(
					fmt.Sprintf("operator '-' requires integer operand, got '%s'", operandType.String()),
					expr.OperandPos, expr.OperandEnd,
				).WithHint("unary minus only works with integer types")
			}
			return &TypedUnaryExpr{
				Type:       TypeError,
				Op:         expr.Op,
				Operand:    operand,
				OpPos:      expr.OpPos,
				OperandEnd: expr.OperandEnd,
			}
		}

		return &TypedUnaryExpr{
			Type:       operandType,
			Op:         expr.Op,
			Operand:    operand,
			OpPos:      expr.OpPos,
			OperandEnd: expr.OperandEnd,
		}
	}

	// Unknown unary operator
	a.addError(fmt.Sprintf("unknown operator '%s'", expr.Op), expr.OpPos, expr.OpPos)
	return &TypedUnaryExpr{
		Type:       TypeError,
		Op:         expr.Op,
		Operand:    operand,
		OpPos:      expr.OpPos,
		OperandEnd: expr.OperandEnd,
	}
}

// checkBinaryOperation checks if a binary operation is valid and returns the result type
func (a *Analyzer) checkBinaryOperation(op string, leftType, rightType Type, leftPos, rightPos ast.Position) Type {
	// Check for error types - propagate them
	if _, ok := leftType.(ErrorType); ok {
		return TypeError
	}
	if _, ok := rightType.(ErrorType); ok {
		return TypeError
	}

	// Elvis operator: left ?: right
	// Left must be nullable T?, right must be compatible with T, result is T
	if op == "?:" {
		innerType, isNullable := UnwrapNullable(leftType)
		if !isNullable {
			a.addError(
				fmt.Sprintf("operator '?:' requires nullable type on left, got '%s'", leftType.String()),
				leftPos, leftPos,
			).WithHint("elvis operator provides a default value for nullable types")
			return TypeError
		}

		// Right operand must be compatible with unwrapped inner type
		if !rightType.Equals(innerType) && !IsAssignableTo(rightType, innerType) {
			a.addError(
				fmt.Sprintf("operator '?:' requires right operand of type '%s', got '%s'",
					innerType.String(), rightType.String()),
				rightPos, rightPos,
			)
			return TypeError
		}

		return innerType
	}

	// Logical operators: &&, ||
	// These require boolean operands and return boolean
	if op == "&&" || op == "||" {
		// Check left operand is boolean
		if _, isBool := leftType.(BooleanType); !isBool {
			a.addError(
				fmt.Sprintf("operator '%s' requires boolean operands, got '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("logical operators only work with boolean values")
			return TypeError
		}

		// Check right operand is boolean
		if _, isBool := rightType.(BooleanType); !isBool {
			a.addError(
				fmt.Sprintf("operator '%s' requires boolean operands, got '%s'", op, rightType.String()),
				rightPos, rightPos,
			).WithHint("logical operators only work with boolean values")
			return TypeError
		}

		return TypeBoolean
	}

	// Arithmetic operators: +, -, *, /, %
	// These require matching numeric types (strict type matching)
	if op == "+" || op == "-" || op == "*" || op == "/" || op == "%" {
		// Check left operand is numeric
		if !IsIntegerType(leftType) && !IsFloatType(leftType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires numeric operands, but left operand has type '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("arithmetic operators only work with numeric types")
			return TypeError
		}

		// Check right operand is numeric
		if !IsIntegerType(rightType) && !IsFloatType(rightType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires numeric operands, but right operand has type '%s'", op, rightType.String()),
				rightPos, rightPos,
			).WithHint("arithmetic operators only work with numeric types")
			return TypeError
		}

		// Type matching: both operands must have the same type, or one must widen to the other
		if !leftType.Equals(rightType) {
			if IntegerWidensTo(leftType, rightType) {
				// left widens to right — use right's type as result
				leftType = rightType
			} else if IntegerWidensTo(rightType, leftType) {
				// right widens to left — use left's type as result
			} else {
				a.addError(
					fmt.Sprintf("operator '%s' requires operands of the same type, but got '%s' and '%s'",
						op, leftType.String(), rightType.String()),
					leftPos, rightPos,
				).WithHint("both operands must have the same type (no implicit conversion)")
				return TypeError
			}
		}

		// Modulo only works with integers
		if op == "%" && IsFloatType(leftType) {
			a.addError(
				fmt.Sprintf("operator '%%' is not supported for floating point types"),
				leftPos, rightPos,
			).WithHint("modulo only works with integer types")
			return TypeError
		}

		return leftType
	}

	// Comparison operators: ==, !=, <, >, <=, >=
	// These require matching numeric types and return bool
	if op == "==" || op == "!=" || op == "<" || op == ">" || op == "<=" || op == ">=" {
		// Special case: null comparison (x == null, x != null)
		// These are only allowed for nullable types and return bool
		if op == "==" || op == "!=" {
			_, leftIsNothing := leftType.(NothingType)
			_, rightIsNothing := rightType.(NothingType)

			// One side is null
			if leftIsNothing || rightIsNothing {
				// Get the non-null type
				var otherType Type
				if leftIsNothing {
					otherType = rightType
				} else {
					otherType = leftType
				}

				// The other side must be nullable or also null
				_, otherIsNothing := otherType.(NothingType)
				if otherIsNothing || IsNullable(otherType) {
					return TypeBoolean // null == null or T? == null returns bool
				}

				// Cannot compare non-nullable with null
				a.addError(
					fmt.Sprintf("cannot compare non-nullable type '%s' with null", otherType.String()),
					leftPos, rightPos,
				).WithHint("only nullable types can be compared with null")
				return TypeError
			}

			// Both sides are nullable - comparing two T? values
			if IsNullable(leftType) && IsNullable(rightType) {
				leftInner, _ := UnwrapNullable(leftType)
				rightInner, _ := UnwrapNullable(rightType)
				if leftInner.Equals(rightInner) {
					return TypeBoolean
				}
			}
		}

		// Boolean equality: == and != work on booleans
		if op == "==" || op == "!=" {
			_, leftIsBool := leftType.(BooleanType)
			_, rightIsBool := rightType.(BooleanType)
			if leftIsBool && rightIsBool {
				return TypeBoolean
			}
		}

		// String equality: == and != work on strings
		if op == "==" || op == "!=" {
			_, leftIsStr := leftType.(StringType)
			_, rightIsStr := rightType.(StringType)
			if leftIsStr && rightIsStr {
				return TypeBoolean
			}
		}

		// Check left operand is numeric
		if !IsIntegerType(leftType) && !IsFloatType(leftType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires numeric operands, but left operand has type '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("comparison operators only work with numeric types")
			return TypeError
		}

		// Check right operand is numeric
		if !IsIntegerType(rightType) && !IsFloatType(rightType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires numeric operands, but right operand has type '%s'", op, rightType.String()),
				rightPos, rightPos,
			).WithHint("comparison operators only work with numeric types")
			return TypeError
		}

		// Type matching: both operands must have the same type, or one must widen to the other
		if !leftType.Equals(rightType) {
			if !IntegerWidensTo(leftType, rightType) && !IntegerWidensTo(rightType, leftType) {
				a.addError(
					fmt.Sprintf("operator '%s' requires operands of the same type, but got '%s' and '%s'",
						op, leftType.String(), rightType.String()),
					leftPos, rightPos,
				).WithHint("both operands must have the same type (no implicit conversion)")
				return TypeError
			}
		}

		// Comparison result is boolean
		return TypeBoolean
	}

	// Unknown operator
	a.addError(fmt.Sprintf("unknown operator '%s'", op), leftPos, rightPos)
	return TypeError
}

// addError adds a compiler error to the error list
func (a *Analyzer) addError(message string, startPos, endPos ast.Position) *errors.CompilerError {
	err := errors.NewErrorWithSpan(message, a.filename, toErrorPos(startPos), toErrorPos(endPos), "semantic")
	err.Tool = errors.ToolSL
	a.errors = append(a.errors, err)
	return err
}

// checkIntegerBounds checks if an integer literal fits in the declared type.
// This delegates to the data-driven bounds checking in bounds.go.
func (a *Analyzer) checkIntegerBounds(value string, targetType Type, pos ast.Position) bool {
	errMsg := checkIntegerBoundsCore(value, targetType)
	if errMsg != "" {
		a.addError(errMsg, pos, pos)
		return false
	}
	return true
}

// checkFloatBounds checks if a float literal can be represented in the target type
func (a *Analyzer) checkFloatBounds(value string, targetType Type, pos ast.Position) bool {
	_, err := strconv.ParseFloat(value, 64)
	if err != nil {
		a.addError(fmt.Sprintf("invalid float literal: %s", value), pos, pos)
		return false
	}

	// For now, we don't do strict float bounds checking
	// f32 and f64 can represent most reasonable literals
	return true
}

// allPathsReturn checks if a list of statements guarantees a return on all code paths.
// This is used to verify that non-void functions return a value.
func allPathsReturn(stmts []TypedStatement) bool {
	for _, stmt := range stmts {
		if statementReturns(stmt) {
			return true
		}
	}
	return false
}

// statementReturns checks if a single statement guarantees a return.
func statementReturns(stmt TypedStatement) bool {
	switch s := stmt.(type) {
	case *TypedReturnStmt:
		return true
	case *TypedIfStmt:
		// If statement only guarantees return if both branches exist and both return
		if s.ElseBranch == nil {
			return false
		}
		thenReturns := blockReturns(s.ThenBranch)
		elseReturns := branchReturns(s.ElseBranch)
		return thenReturns && elseReturns
	case *TypedWhenExpr:
		// When guarantees return if exhaustive and all branches return
		if !whenIsExhaustive(s) {
			return false
		}
		for _, wcase := range s.Cases {
			if !branchReturns(wcase.Body) {
				return false
			}
		}
		return true
	case *TypedBlockStmt:
		return allPathsReturn(s.Statements)
	default:
		return false
	}
}

// blockReturns checks if a block statement guarantees a return.
func blockReturns(block *TypedBlockStmt) bool {
	return allPathsReturn(block.Statements)
}

// branchReturns checks if an else branch (which can be a block or another if/when) returns.
func branchReturns(branch TypedStatement) bool {
	switch b := branch.(type) {
	case *TypedBlockStmt:
		return allPathsReturn(b.Statements)
	case *TypedIfStmt:
		// else if: recursively check
		return statementReturns(b)
	case *TypedWhenExpr:
		// when expression: recursively check
		return statementReturns(b)
	default:
		return false
	}
}

// whenIsExhaustive checks if a when expression covers all cases
// A when is exhaustive if it has an else branch or a literal `true` condition
func whenIsExhaustive(when *TypedWhenExpr) bool {
	for _, wcase := range when.Cases {
		if wcase.IsElse {
			return true
		}
		// Check for literal `true` condition (always executes)
		if wcase.Condition != nil {
			if lit, ok := wcase.Condition.(*TypedLiteralExpr); ok {
				if lit.LitType == ast.LiteralTypeBoolean && lit.Value == "true" {
					return true
				}
			}
		}
	}
	return false
}

// analyzeWhenStatement analyzes a when statement (statement context, no result type)
func (a *Analyzer) analyzeWhenStatement(when *ast.WhenExpr) TypedStatement {
	return a.analyzeWhen(when, false)
}

// analyzeWhenExpression analyzes a when expression (expression context, needs result type)
func (a *Analyzer) analyzeWhenExpression(when *ast.WhenExpr) TypedExpression {
	return a.analyzeWhen(when, true)
}

// analyzeWhen is the core when analysis, handling both statement and expression contexts
func (a *Analyzer) analyzeWhen(when *ast.WhenExpr, isExpression bool) *TypedWhenExpr {
	// Snapshot ownership state before any cases for conditional move tracking
	beforeSnapshot := a.snapshotOwnershipState()

	// Analyze cases
	typedCases := make([]TypedWhenCase, len(when.Cases))
	var hasElse bool
	var hasTrueCondition bool
	var branchTypes []Type  // For expression type checking
	var allMoves []moveRecord // Collect moves from all branches

	for i, wcase := range when.Cases {
		typedCase := a.analyzeWhenCase(wcase, isExpression)
		typedCases[i] = typedCase

		// Collect moves from this case
		caseMoves := a.collectBranchMoves(beforeSnapshot)
		allMoves = append(allMoves, caseMoves...)

		if wcase.IsElse {
			hasElse = true
		}

		// Check for literal `true` condition
		if typedCase.Condition != nil {
			if lit, ok := typedCase.Condition.(*TypedLiteralExpr); ok {
				if lit.LitType == ast.LiteralTypeBoolean && lit.Value == "true" {
					hasTrueCondition = true
				}
			}
		}

		if isExpression {
			branchTypes = append(branchTypes, a.getWhenCaseResultType(typedCase.Body))
		}
	}

	// Merge moves from all cases: if a variable was moved in ANY case,
	// it must be considered moved after the when expression
	// We use empty moves for the "no case matched" scenario (implicit else)
	var emptyMoves []moveRecord
	a.mergeConditionalMoves(allMoves, emptyMoves)

	// Exhaustiveness checking
	a.checkWhenExhaustiveness(when, hasElse, hasTrueCondition)

	// Type checking for expressions
	var resultType Type = TypeVoid
	if isExpression {
		resultType = a.checkWhenBranchTypeConsistency(branchTypes, when.WhenKeyword, when.RightBrace)
	}

	return &TypedWhenExpr{
		WhenKeyword: when.WhenKeyword,
		Cases:       typedCases,
		RightBrace:  when.RightBrace,
		ResultType:  resultType,
	}
}

// analyzeWhenCase analyzes a single when case
func (a *Analyzer) analyzeWhenCase(wcase ast.WhenCase, isExpression bool) TypedWhenCase {
	var typedCondition TypedExpression

	if !wcase.IsElse {
		typedCondition = a.analyzeExpression(wcase.Condition)
		conditionType := typedCondition.GetType()

		// Condition must be boolean
		if _, isBool := conditionType.(BooleanType); !isBool {
			if _, isErr := conditionType.(ErrorType); !isErr {
				a.addError(
					fmt.Sprintf("when case condition must be boolean, got '%s'", conditionType.String()),
					wcase.ConditionPos, wcase.Condition.End(),
				).WithHint("use a comparison or boolean expression")
			}
		}
	}

	// Analyze body
	var typedBody TypedStatement
	switch body := wcase.Body.(type) {
	case *ast.BlockStmt:
		a.enterScope()
		if isExpression {
			typedBlock := a.analyzeBlockStmtForExpression(body)
			typedBody = typedBlock
			// Check that block ends with an expression
			if len(body.Statements) > 0 {
				resultType := a.getBlockResultType(typedBlock)
				if _, isVoid := resultType.(VoidType); isVoid {
					lastStmt := body.Statements[len(body.Statements)-1]
					a.addError(
						"when expression block must end with an expression",
						lastStmt.Pos(), lastStmt.End(),
					).WithHint("the last statement in a when expression block must produce a value")
				}
			} else {
				a.addError(
					"when expression block cannot be empty",
					body.LeftBrace, body.RightBrace,
				).WithHint("add an expression that produces a value")
			}
		} else {
			typedBody = a.analyzeBlockStmt(body)
		}
		a.exitScope()
	case *ast.ExprStmt:
		typedBody = a.analyzeExprStatement(body)
	case *ast.AssignStmt:
		if isExpression {
			a.addError(
				"when expression branches must contain expressions, not statements",
				body.NamePos, body.Value.End(),
			).WithHint("assignment statements don't produce a value; use a block or expression instead")
		}
		typedBody = a.analyzeStatement(wcase.Body)
	default:
		if isExpression {
			a.addError(
				"when expression branches must contain expressions, not statements",
				wcase.Body.Pos(), wcase.Body.End(),
			).WithHint("use an expression that produces a value")
		}
		typedBody = a.analyzeStatement(wcase.Body)
	}

	return TypedWhenCase{
		Condition:    typedCondition,
		ConditionPos: wcase.ConditionPos,
		Arrow:        wcase.Arrow,
		Body:         typedBody,
		IsElse:       wcase.IsElse,
	}
}

// checkWhenExhaustiveness verifies that at least one branch is guaranteed to execute
func (a *Analyzer) checkWhenExhaustiveness(when *ast.WhenExpr, hasElse bool, hasTrueCondition bool) {
	if hasElse || hasTrueCondition {
		return // exhaustive: else covers everything, or a `true` condition always executes
	}

	a.addError(
		"when is not exhaustive: no branch is guaranteed to execute",
		when.WhenKeyword, when.RightBrace,
	).WithHint("add 'else -> ...' or use 'true -> ...' to ensure a branch always executes")
}

// checkWhenBranchTypeConsistency ensures all branches have the same type
func (a *Analyzer) checkWhenBranchTypeConsistency(types []Type, startPos, endPos ast.Position) Type {
	if len(types) == 0 {
		return TypeVoid
	}

	firstType := types[0]
	for i := 1; i < len(types); i++ {
		if _, isErr := types[i].(ErrorType); isErr {
			continue
		}
		if _, isErr := firstType.(ErrorType); isErr {
			firstType = types[i]
			continue
		}
		if !firstType.Equals(types[i]) {
			a.addError(
				fmt.Sprintf("when branches have different types: '%s' and '%s'",
					firstType.String(), types[i].String()),
				startPos, endPos,
			).WithHint("all branches must evaluate to the same type")
			return TypeError
		}
	}
	return firstType
}

// getWhenCaseResultType extracts the result type from a when case body
func (a *Analyzer) getWhenCaseResultType(body TypedStatement) Type {
	switch b := body.(type) {
	case *TypedBlockStmt:
		return a.getBlockResultType(b)
	case *TypedExprStmt:
		return b.Expr.GetType()
	default:
		return TypeVoid
	}
}

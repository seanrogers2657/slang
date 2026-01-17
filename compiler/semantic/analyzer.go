package semantic

import (
	"fmt"
	"math/big"
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
	structs           map[string]StructType   // struct registry
	currentReturnType Type                    // return type of current function being analyzed
	loopDepth         int                     // tracks nested loop depth for break/continue validation
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(filename string) *Analyzer {
	return &Analyzer{
		filename:          filename,
		errors:            make([]*errors.CompilerError, 0),
		currentScope:      newScope(nil),           // global scope
		ownershipScope:    newOwnershipScope(nil),  // global ownership scope
		functions:         make(map[string]FunctionInfo),
		structs:           make(map[string]StructType),
		currentReturnType: nil,
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

// Analyze performs semantic analysis on a program
func (a *Analyzer) Analyze(program *ast.Program) ([]*errors.CompilerError, *TypedProgram) {
	typedProgram := &TypedProgram{
		Declarations: make([]TypedDeclaration, 0),
		Statements:   make([]TypedStatement, 0),
		StartPos:     program.StartPos,
		EndPos:       program.EndPos,
	}

	// Handle declaration-based programs
	if len(program.Declarations) > 0 {
		// First pass: register all struct names (so forward references work)
		for _, decl := range program.Declarations {
			if structDecl, ok := decl.(*ast.StructDecl); ok {
				a.registerStructName(structDecl)
			}
		}

		// Second pass: resolve struct field types (now all struct names are known)
		for _, decl := range program.Declarations {
			if structDecl, ok := decl.(*ast.StructDecl); ok {
				a.resolveStructFields(structDecl)
			}
		}

		// Third pass: collect all function signatures
		hasMain := false
		for _, decl := range program.Declarations {
			if fnDecl, ok := decl.(*ast.FunctionDecl); ok {
				a.registerFunction(fnDecl)
				if fnDecl.Name == "main" {
					hasMain = true
				}
			}
		}

		if !hasMain {
			// Add error if no main function found - point to end of file
			a.addError("program must have a 'main' function", program.EndPos, program.EndPos)
		}

		// Third pass: analyze all declarations
		for _, decl := range program.Declarations {
			typedDecl := a.analyzeDeclaration(decl)
			typedProgram.Declarations = append(typedProgram.Declarations, typedDecl)
		}
	} else {
		// Handle legacy statement-based programs
		for _, stmt := range program.Statements {
			typedStmt := a.analyzeStatement(stmt)
			typedProgram.Statements = append(typedProgram.Statements, typedStmt)
		}
	}

	return a.errors, typedProgram
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
	if _, exists := a.structs[s.Name]; exists {
		a.addError(fmt.Sprintf("struct '%s' is already declared", s.Name), s.NamePos, s.NamePos)
		return
	}

	// Register the struct with empty fields (will be resolved in second pass)
	a.structs[s.Name] = StructType{
		Name:   s.Name,
		Fields: nil, // Placeholder until resolveStructFields is called
	}
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

	// Update the struct with resolved fields
	a.structs[s.Name] = StructType{
		Name:   s.Name,
		Fields: fields,
	}
}

// registerStruct registers a struct type in the struct registry (legacy - for testing)
func (a *Analyzer) registerStruct(s *ast.StructDecl) {
	// Check for duplicate struct
	if _, exists := a.structs[s.Name]; exists {
		a.addError(fmt.Sprintf("struct '%s' is already declared", s.Name), s.NamePos, s.NamePos)
		return
	}

	// Convert field types
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

	a.structs[s.Name] = StructType{
		Name:   s.Name,
		Fields: fields,
	}
}

// resolveTypeName converts a type name string to a Type, checking both primitive types and structs
func (a *Analyzer) resolveTypeName(name string, pos ast.Position) Type {
	// Check for nullable type T?
	if strings.HasSuffix(name, "?") {
		innerName := name[:len(name)-1]
		innerType := a.resolveTypeName(innerName, pos)
		if _, isErr := innerType.(ErrorType); isErr {
			return TypeError
		}
		// Check for nested nullables (T??) - defense-in-depth
		// The parser already catches this at parse time, but this check remains
		// for programmatically constructed ASTs (e.g., in tests)
		if IsNullable(innerType) {
			a.addError("nested nullable types are not allowed", pos, pos)
			return TypeError
		}
		return NullableType{InnerType: innerType}
	}

	// Check for symbol-based pointer syntax: *T, &&T, &T
	// IMPORTANT: Check "&&" before "&" to avoid wrong prefix match

	// Check for &&T syntax (mutable borrow)
	if strings.HasPrefix(name, "&&") {
		elementTypeName := strings.TrimPrefix(name, "&&")
		elementType := a.resolveTypeName(elementTypeName, pos)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// &&T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			a.addError("&&T cannot contain nullable type; use &&T? for nullable mutable references", pos, pos)
			return TypeError
		}
		return MutRefPointerType{ElementType: elementType}
	}

	// Check for &T syntax (immutable borrow)
	if strings.HasPrefix(name, "&") {
		elementTypeName := strings.TrimPrefix(name, "&")
		elementType := a.resolveTypeName(elementTypeName, pos)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// &T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			a.addError("&T cannot contain nullable type; use &T? for nullable references", pos, pos)
			return TypeError
		}
		return RefPointerType{ElementType: elementType}
	}

	// Check for *T syntax (owned pointer)
	if strings.HasPrefix(name, "*") {
		elementTypeName := strings.TrimPrefix(name, "*")
		elementType := a.resolveTypeName(elementTypeName, pos)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// *T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			a.addError("*T cannot contain nullable type; use *T? for nullable owned pointers", pos, pos)
			return TypeError
		}
		return OwnedPointerType{ElementType: elementType}
	}

	// Check for Array<T> syntax
	if strings.HasPrefix(name, "Array<") && strings.HasSuffix(name, ">") {
		elementTypeName := name[6 : len(name)-1] // extract T from Array<T>
		elementType := a.resolveTypeName(elementTypeName, pos)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// Return ArrayType with unknown size (will be inferred from literal)
		return ArrayType{ElementType: elementType, Size: ArraySizeUnknown}
	}

	// Try primitive types first
	t := TypeFromName(name)
	if _, isErr := t.(ErrorType); !isErr {
		return t
	}

	// Try struct types
	if structType, ok := a.structs[name]; ok {
		return structType
	}

	// Unknown type
	a.addError(fmt.Sprintf("unknown type '%s'", name), pos, pos)
	return TypeError
}

// resolveTypeNameNoError converts a type name string to a Type without adding errors
// (used when caller wants to handle errors itself)
func (a *Analyzer) resolveTypeNameNoError(name string) Type {
	// Check for nullable type T?
	if strings.HasSuffix(name, "?") {
		innerName := name[:len(name)-1]
		innerType := a.resolveTypeNameNoError(innerName)
		if _, isErr := innerType.(ErrorType); isErr {
			return TypeError
		}
		// Nested nullables are error - defense-in-depth (parser catches this at parse time)
		if IsNullable(innerType) {
			return TypeError
		}
		return NullableType{InnerType: innerType}
	}

	// Check for symbol-based pointer syntax: *T, &&T, &T
	// IMPORTANT: Check "&&" before "&" to avoid wrong prefix match

	// Check for &&T syntax (mutable borrow)
	if strings.HasPrefix(name, "&&") {
		elementTypeName := strings.TrimPrefix(name, "&&")
		elementType := a.resolveTypeNameNoError(elementTypeName)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// &&T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			return TypeError
		}
		return MutRefPointerType{ElementType: elementType}
	}

	// Check for &T syntax (immutable borrow)
	if strings.HasPrefix(name, "&") {
		elementTypeName := strings.TrimPrefix(name, "&")
		elementType := a.resolveTypeNameNoError(elementTypeName)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// &T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			return TypeError
		}
		return RefPointerType{ElementType: elementType}
	}

	// Check for *T syntax (owned pointer)
	if strings.HasPrefix(name, "*") {
		elementTypeName := strings.TrimPrefix(name, "*")
		elementType := a.resolveTypeNameNoError(elementTypeName)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// *T? is invalid - the element type cannot be nullable
		if IsNullable(elementType) {
			return TypeError
		}
		return OwnedPointerType{ElementType: elementType}
	}

	// Check for Array<T> syntax
	if strings.HasPrefix(name, "Array<") && strings.HasSuffix(name, ">") {
		elementTypeName := name[6 : len(name)-1] // extract T from Array<T>
		elementType := a.resolveTypeNameNoError(elementTypeName)
		if _, isErr := elementType.(ErrorType); isErr {
			return TypeError
		}
		// Return ArrayType with unknown size (will be inferred from literal)
		return ArrayType{ElementType: elementType, Size: ArraySizeUnknown}
	}

	// Try primitive types first
	t := TypeFromName(name)
	if _, isErr := t.(ErrorType); !isErr {
		return t
	}

	// Try struct types
	if structType, ok := a.structs[name]; ok {
		return structType
	}

	return TypeError
}

// analyzeDeclaration performs semantic analysis on a declaration
func (a *Analyzer) analyzeDeclaration(decl ast.Declaration) TypedDeclaration {
	switch d := decl.(type) {
	case *ast.FunctionDecl:
		return a.analyzeFunctionDecl(d)
	case *ast.StructDecl:
		return a.analyzeStructDecl(d)
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
	structType := a.structs[s.Name]

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

// analyzeFunctionDecl analyzes a function declaration
func (a *Analyzer) analyzeFunctionDecl(fn *ast.FunctionDecl) TypedDeclaration {
	// Get function info
	fnInfo := a.functions[fn.Name]

	// Set current return type for return statement checking
	prevReturnType := a.currentReturnType
	a.currentReturnType = fnInfo.ReturnType

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

	// Restore previous return type
	a.currentReturnType = prevReturnType

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
// The last statement, if it's an IfStmt, is analyzed as an expression.
func (a *Analyzer) analyzeBlockStmtForExpression(block *ast.BlockStmt) *TypedBlockStmt {
	typedStmts := make([]TypedStatement, 0, len(block.Statements))

	for i, stmt := range block.Statements {
		var typedStmt TypedStatement
		// For the last statement, check if it's an IfStmt that should be an expression
		if i == len(block.Statements)-1 {
			if ifStmt, ok := stmt.(*ast.IfStmt); ok {
				// Analyze as expression to get proper type
				typedStmt = a.analyzeIfExpression(ifStmt).(*TypedIfStmt)
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

	// Check that the object is a struct type
	structType, isStruct := objectType.(StructType)
	if !isStruct {
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
	fieldInfo, found := structType.GetField(stmt.Field)
	if !found {
		a.addError(
			fmt.Sprintf("struct '%s' has no field '%s'", structType.Name, stmt.Field),
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
			a.addError(
				fmt.Sprintf("cannot assign to element of immutable array '%s'", ident.Name),
				stmt.LeftBracket, stmt.Equals,
			).WithHint("consider using 'var' instead of 'val' if you need to modify elements")
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
	if structType, ok := a.structs[call.Name]; ok {
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

// analyzeLenBuiltin analyzes a len() call on an array
func (a *Analyzer) analyzeLenBuiltin(call *ast.CallExpr) TypedExpression {
	// Check argument count
	if len(call.Arguments) != 1 {
		a.addError(
			fmt.Sprintf("len() takes exactly 1 argument, got %d", len(call.Arguments)),
			call.LeftParen, call.RightParen,
		)
		// Return error typed expression
		return &TypedLenExpr{
			Type:       TypeI64,
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

	// Check that argument is an array type
	arrayType, isArray := argType.(ArrayType)
	if !isArray {
		if _, isErr := argType.(ErrorType); !isErr {
			a.addError(
				fmt.Sprintf("len() argument must be an array, got '%s'", argType.String()),
				call.Arguments[0].Pos(), call.Arguments[0].End(),
			)
		}
		return &TypedLenExpr{
			Type:       TypeI64,
			Array:      typedArg,
			ArraySize:  ArraySizeUnknown,
			NamePos:    call.NamePos,
			LeftParen:  call.LeftParen,
			RightParen: call.RightParen,
		}
	}

	return &TypedLenExpr{
		Type:       TypeI64,
		Array:      typedArg,
		ArraySize:  arrayType.Size,
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
func (a *Analyzer) analyzeStructLiteralExpr(lit *ast.StructLiteral) TypedExpression {
	// Check if this is a known struct type
	structType, ok := a.structs[lit.Name]
	if !ok {
		a.addError(
			fmt.Sprintf("undefined struct '%s'", lit.Name),
			lit.NamePos, lit.NamePos,
		)
		return &TypedLiteralExpr{Type: ErrorType{}}
	}

	// Handle named arguments
	if lit.HasNamedArguments() {
		return a.analyzeStructLiteralExprNamed(lit, structType)
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
		TypePos:    lit.NamePos,
		LeftBrace:  lit.LeftBrace,
		Args:       typedArgs,
		RightBrace: lit.RightBrace,
	}
}

// analyzeStructLiteralExprNamed analyzes a struct literal with named arguments (e.g., Point { x: 10, y: 20 })
func (a *Analyzer) analyzeStructLiteralExprNamed(lit *ast.StructLiteral, structType StructType) TypedExpression {
	// Build a map of field name -> index for quick lookup
	fieldIndex := make(map[string]int)
	for i, field := range structType.Fields {
		fieldIndex[field.Name] = i
	}

	// Check argument count matches field count
	if len(lit.NamedArguments) != len(structType.Fields) {
		a.addError(
			fmt.Sprintf("struct '%s' has %d field(s), but %d argument(s) were provided",
				structType.Name, len(structType.Fields), len(lit.NamedArguments)),
			lit.LeftBrace, lit.RightBrace,
		)
	}

	// Track which fields have been provided (for duplicate detection)
	providedFields := make(map[string]ast.Position)

	// Create typed arguments array in field order
	typedArgs := make([]TypedExpression, len(structType.Fields))

	for _, namedArg := range lit.NamedArguments {
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
			if len(lit.NamedArguments) == len(structType.Fields) {
				a.addError(
					fmt.Sprintf("missing field '%s' in struct '%s'", field.Name, structType.Name),
					lit.LeftBrace, lit.RightBrace,
				)
			}
		}
	}

	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    lit.NamePos,
		LeftBrace:  lit.LeftBrace,
		Args:       typedArgs,
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
	// Build a map of field name -> index for quick lookup
	fieldIndex := make(map[string]int)
	for i, field := range structType.Fields {
		fieldIndex[field.Name] = i
	}

	// Check argument count matches field count
	if len(lit.NamedArguments) != len(structType.Fields) {
		a.addError(
			fmt.Sprintf("struct '%s' has %d field(s), but %d argument(s) were provided",
				structType.Name, len(structType.Fields), len(lit.NamedArguments)),
			lit.LeftBrace, lit.RightBrace,
		)
	}

	// Track which fields have been provided (for duplicate detection)
	providedFields := make(map[string]ast.Position)

	// Create typed arguments array in field order
	typedArgs := make([]TypedExpression, len(structType.Fields))

	for _, namedArg := range lit.NamedArguments {
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
			if len(lit.NamedArguments) == len(structType.Fields) {
				a.addError(
					fmt.Sprintf("missing field '%s' in struct '%s'", field.Name, structType.Name),
					lit.LeftBrace, lit.RightBrace,
				)
			}
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

// analyzeFieldAccessExpr analyzes a field access expression (e.g., p.x, rect.topLeft.x)
func (a *Analyzer) analyzeFieldAccessExpr(expr *ast.FieldAccessExpr) TypedExpression {
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

	// Check that the object is a struct type
	structType, isStruct := objectType.(StructType)
	if !isStruct {
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
	fieldInfo, found := structType.GetField(expr.Field)
	if !found {
		a.addError(
			fmt.Sprintf("struct '%s' has no field '%s'", structType.Name, expr.Field),
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

// analyzeMethodCallExpr analyzes a method call expression (e.g., Heap.new(x), p.copy())
func (a *Analyzer) analyzeMethodCallExpr(expr *ast.MethodCallExpr) TypedExpression {
	// Check if this is a Heap.new() call
	if ident, ok := expr.Object.(*ast.IdentifierExpr); ok && ident.Name == "Heap" {
		return a.analyzeHeapMethodCall(expr)
	}

	// Analyze the object expression
	typedObject := a.analyzeExpression(expr.Object)
	objectType := typedObject.GetType()

	// Type arguments
	typedArgs := make([]TypedExpression, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)
	}

	// Check if this is a .copy() call on an owned pointer
	if expr.Method == "copy" {
		if ownedType, isOwned := objectType.(OwnedPointerType); isOwned {
			// .copy() on Own<T> returns a new Own<T> (deep copy)
			if len(expr.Arguments) != 0 {
				a.addError(
					fmt.Sprintf("copy() takes no arguments, got %d", len(expr.Arguments)),
					expr.LeftParen, expr.RightParen,
				)
			}
			return &TypedMethodCallExpr{
				Type:       ownedType, // returns same type
				Object:     typedObject,
				Dot:        expr.Dot,
				Method:     expr.Method,
				MethodPos:  expr.MethodPos,
				LeftParen:  expr.LeftParen,
				Arguments:  typedArgs,
				RightParen: expr.RightParen,
			}
		}
	}

	// Unknown method call
	if _, isErr := objectType.(ErrorType); !isErr {
		a.addError(
			fmt.Sprintf("unknown method '%s' on type '%s'", expr.Method, objectType.String()),
			expr.MethodPos, expr.MethodPos,
		)
	}

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

// analyzeHeapMethodCall analyzes Heap.new() and other Heap methods
func (a *Analyzer) analyzeHeapMethodCall(expr *ast.MethodCallExpr) TypedExpression {
	// Type arguments
	typedArgs := make([]TypedExpression, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)
	}

	// Create a dummy typed object for Heap (it's a pseudo-singleton)
	typedObject := &TypedIdentifierExpr{
		Type:     TypeVoid, // Heap doesn't have a real type
		Name:     "Heap",
		StartPos: expr.Object.Pos(),
		EndPos:   expr.Object.End(),
	}

	switch expr.Method {
	case "new":
		// Heap.new(expr) allocates expr on the heap and returns Own<T>
		if len(expr.Arguments) != 1 {
			a.addError(
				fmt.Sprintf("Heap.new() takes exactly 1 argument, got %d", len(expr.Arguments)),
				expr.LeftParen, expr.RightParen,
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

		// Infer the type from the argument
		argType := typedArgs[0].GetType()
		if _, isErr := argType.(ErrorType); isErr {
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

		// Return Own<T> where T is the argument type
		resultType := OwnedPointerType{ElementType: argType}
		return &TypedMethodCallExpr{
			Type:       resultType,
			Object:     typedObject,
			Dot:        expr.Dot,
			Method:     expr.Method,
			MethodPos:  expr.MethodPos,
			LeftParen:  expr.LeftParen,
			Arguments:  typedArgs,
			RightParen: expr.RightParen,
		}

	default:
		a.addError(
			fmt.Sprintf("Heap has no method '%s'", expr.Method),
			expr.MethodPos, expr.MethodPos,
		).WithHint("available methods: new()")
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
		if resolved, ok := a.structs[st.Name]; ok {
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
		// Also allow compatible integer types when i64 is accepted
		if _, isI64 := accepted.(I64Type); isI64 && IsIntegerType(argType) {
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
	// If param expects i64, allow any integer type
	if _, isI64 := paramType.(I64Type); isI64 {
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

		// Strict type matching: both operands must have the same type
		if !leftType.Equals(rightType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires operands of the same type, but got '%s' and '%s'",
					op, leftType.String(), rightType.String()),
				leftPos, rightPos,
			).WithHint("both operands must have the same type (no implicit conversion)")
			return TypeError
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

		// Strict type matching: both operands must have the same type
		if !leftType.Equals(rightType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires operands of the same type, but got '%s' and '%s'",
					op, leftType.String(), rightType.String()),
				leftPos, rightPos,
			).WithHint("both operands must have the same type (no implicit conversion)")
			return TypeError
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

// Type bounds for integer types
var (
	minI8, _   = big.NewInt(0).SetString("-128", 10)
	maxI8, _   = big.NewInt(0).SetString("127", 10)
	minI16, _  = big.NewInt(0).SetString("-32768", 10)
	maxI16, _  = big.NewInt(0).SetString("32767", 10)
	minI32, _  = big.NewInt(0).SetString("-2147483648", 10)
	maxI32, _  = big.NewInt(0).SetString("2147483647", 10)
	minI64, _  = big.NewInt(0).SetString("-9223372036854775808", 10)
	maxI64, _  = big.NewInt(0).SetString("9223372036854775807", 10)
	minI128, _ = big.NewInt(0).SetString("-170141183460469231731687303715884105728", 10)
	maxI128, _ = big.NewInt(0).SetString("170141183460469231731687303715884105727", 10)

	maxU8, _   = big.NewInt(0).SetString("255", 10)
	maxU16, _  = big.NewInt(0).SetString("65535", 10)
	maxU32, _  = big.NewInt(0).SetString("4294967295", 10)
	maxU64, _  = big.NewInt(0).SetString("18446744073709551615", 10)
	maxU128, _ = big.NewInt(0).SetString("340282366920938463463374607431768211455", 10)
)

// checkIntegerBounds checks if an integer literal fits in the declared type
func (a *Analyzer) checkIntegerBounds(value string, targetType Type, pos ast.Position) bool {
	val, ok := big.NewInt(0).SetString(value, 10)
	if !ok {
		a.addError(fmt.Sprintf("invalid integer literal: %s", value), pos, pos)
		return false
	}

	zero := big.NewInt(0)

	switch targetType.(type) {
	case I8Type:
		if val.Cmp(minI8) < 0 || val.Cmp(maxI8) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for i8 (-128 to 127)", value), pos, pos)
			return false
		}
	case I16Type:
		if val.Cmp(minI16) < 0 || val.Cmp(maxI16) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for i16 (-32768 to 32767)", value), pos, pos)
			return false
		}
	case I32Type:
		if val.Cmp(minI32) < 0 || val.Cmp(maxI32) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for i32", value), pos, pos)
			return false
		}
	case I64Type:
		if val.Cmp(minI64) < 0 || val.Cmp(maxI64) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for i64", value), pos, pos)
			return false
		}
	case I128Type:
		if val.Cmp(minI128) < 0 || val.Cmp(maxI128) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for i128", value), pos, pos)
			return false
		}
	case U8Type:
		if val.Cmp(zero) < 0 || val.Cmp(maxU8) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for u8 (0 to 255)", value), pos, pos)
			return false
		}
	case U16Type:
		if val.Cmp(zero) < 0 || val.Cmp(maxU16) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for u16 (0 to 65535)", value), pos, pos)
			return false
		}
	case U32Type:
		if val.Cmp(zero) < 0 || val.Cmp(maxU32) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for u32", value), pos, pos)
			return false
		}
	case U64Type:
		if val.Cmp(zero) < 0 || val.Cmp(maxU64) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for u64", value), pos, pos)
			return false
		}
	case U128Type:
		if val.Cmp(zero) < 0 || val.Cmp(maxU128) > 0 {
			a.addError(fmt.Sprintf("integer literal %s out of range for u128", value), pos, pos)
			return false
		}
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

package semantic

import (
	"fmt"
	"math/big"
	"strconv"

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
	functions         map[string]FunctionInfo // function registry
	structs           map[string]StructType   // struct registry
	currentReturnType Type                    // return type of current function being analyzed
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(filename string) *Analyzer {
	return &Analyzer{
		filename:          filename,
		errors:            make([]*errors.CompilerError, 0),
		currentScope:      newScope(nil), // global scope
		functions:         make(map[string]FunctionInfo),
		structs:           make(map[string]StructType),
		currentReturnType: nil,
	}
}

// enterScope creates a new nested scope
func (a *Analyzer) enterScope() {
	a.currentScope = newScope(a.currentScope)
}

// exitScope returns to the parent scope
func (a *Analyzer) exitScope() {
	if a.currentScope.parent != nil {
		a.currentScope = a.currentScope.parent
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
		// First pass: register all struct types (needed for function signatures)
		for _, decl := range program.Declarations {
			if structDecl, ok := decl.(*ast.StructDecl); ok {
				a.registerStruct(structDecl)
			}
		}

		// Second pass: collect all function signatures
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
		paramTypes[i] = a.resolveTypeName(param.TypeName, param.TypePos)
	}

	// Convert return type (supports both primitive and struct types)
	returnType := a.resolveTypeName(fn.ReturnType, fn.ReturnPos)

	a.functions[fn.Name] = FunctionInfo{
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}
}

// registerStruct registers a struct type in the struct registry
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
			FnKeyword:  decl.Pos(),
			Name:       "error",
			NamePos:    decl.Pos(),
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
		StructKeyword: s.StructKeyword,
		Name:          s.Name,
		NamePos:       s.NamePos,
		LeftParen:     s.LeftParen,
		StructType:    structType,
		RightParen:    s.RightParen,
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
		FnKeyword:  fn.FnKeyword,
		Name:       fn.Name,
		NamePos:    fn.NamePos,
		LeftParen:  fn.LeftParen,
		Parameters: typedParams,
		RightParen: fn.RightParen,
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
	case *ast.ReturnStmt:
		return a.analyzeReturnStatement(s)
	case *ast.IfStmt:
		return a.analyzeIfStatement(s)
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
	// Analyze the initializer expression
	typedInit := a.analyzeExpression(stmt.Initializer)
	initType := typedInit.GetType()

	// Determine the declared type
	var declaredType Type
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
	}

	// Check for duplicate declaration in the current scope
	if !a.currentScope.declare(stmt.Name, declaredType, stmt.Mutable) {
		a.addError(
			fmt.Sprintf("variable '%s' is already declared in this scope", stmt.Name),
			stmt.NamePos, stmt.NamePos,
		)
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

// checkTypeCompatibility checks if an initializer is compatible with the declared type
func (a *Analyzer) checkTypeCompatibility(declaredType, initType Type, typedInit TypedExpression, pos ast.Position) bool {
	// If types are exactly equal, always ok
	if declaredType.Equals(initType) {
		return true
	}

	// Check for literal bounds when assigning to a specific type
	if litExpr, ok := typedInit.(*TypedLiteralExpr); ok {
		// Integer literal -> any integer type (with bounds check)
		if litExpr.LitType == ast.LiteralTypeInteger && isIntegerType(declaredType) {
			return a.checkIntegerBounds(litExpr.Value, declaredType, pos)
		}

		// Float literal -> any float type (with bounds check)
		if litExpr.LitType == ast.LiteralTypeFloat && isFloatType(declaredType) {
			return a.checkFloatBounds(litExpr.Value, declaredType, pos)
		}

		// Integer literal cannot be assigned to float type
		if litExpr.LitType == ast.LiteralTypeInteger && isFloatType(declaredType) {
			a.addError(
				fmt.Sprintf("cannot assign integer literal to %s", declaredType.String()),
				pos, pos,
			).WithHint("use a float literal like 42.0 instead")
			return false
		}

		// Float literal cannot be assigned to integer type
		if litExpr.LitType == ast.LiteralTypeFloat && isIntegerType(declaredType) {
			a.addError(
				fmt.Sprintf("cannot assign float literal to %s", declaredType.String()),
				pos, pos,
			)
			return false
		}
	}

	// Types don't match and no special conversion allowed
	a.addError(
		fmt.Sprintf("cannot assign %s to variable of type %s", initType.String(), declaredType.String()),
		pos, pos,
	)
	return false
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

	// Analyze the value expression
	typedValue := a.analyzeExpression(stmt.Value)
	valueType := typedValue.GetType()

	// Type check: value must match variable type (skip if value is already error type)
	if _, isErr := valueType.(ErrorType); !isErr && !info.Type.Equals(valueType) {
		a.addError(
			fmt.Sprintf("cannot assign %s to variable '%s' of type %s",
				valueType.String(), stmt.Name, info.Type.String()),
			stmt.Value.Pos(), stmt.Value.End(),
		)
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

	// Check field mutability
	if !fieldInfo.Mutable {
		a.addError(
			fmt.Sprintf("cannot assign to immutable field '%s'", stmt.Field),
			stmt.FieldPos, stmt.Equals,
		).WithHint("consider using 'var' instead of 'val' in the struct definition")
	}

	// Analyze the value expression
	typedValue := a.analyzeExpression(stmt.Value)
	valueType := typedValue.GetType()

	// Type check: value must match field type
	if _, isErr := valueType.(ErrorType); !isErr && !fieldInfo.Type.Equals(valueType) {
		a.addError(
			fmt.Sprintf("cannot assign %s to field '%s' of type %s",
				valueType.String(), stmt.Field, fieldInfo.Type.String()),
			stmt.Value.Pos(), stmt.Value.End(),
		)
	}

	return &TypedFieldAssignStmt{
		Object:   typedObject,
		Dot:      stmt.Dot,
		Field:    stmt.Field,
		FieldPos: stmt.FieldPos,
		Equals:   stmt.Equals,
		Value:    typedValue,
	}
}

// analyzeReturnStatement analyzes a return statement
func (a *Analyzer) analyzeReturnStatement(stmt *ast.ReturnStmt) TypedStatement {
	// Check if we're in a function
	if a.currentReturnType == nil {
		a.addError("return statement outside of function", stmt.Keyword, stmt.Keyword)
		return &TypedReturnStmt{
			Keyword: stmt.Keyword,
			Value:   nil,
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
		} else if _, isErr := valueType.(ErrorType); !isErr && !a.currentReturnType.Equals(valueType) {
			a.addError(
				fmt.Sprintf("return type mismatch: expected %s, got %s",
					a.currentReturnType.String(), valueType.String()),
				stmt.Value.Pos(), stmt.Value.End(),
			)
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
		Keyword: stmt.Keyword,
		Value:   typedValue,
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

	// Analyze the then branch (with its own scope)
	a.enterScope()
	typedThenBranch := a.analyzeBlockStmt(stmt.ThenBranch)
	a.exitScope()

	// Analyze the else branch if present
	var typedElseBranch TypedStatement
	if stmt.ElseBranch != nil {
		switch elseBranch := stmt.ElseBranch.(type) {
		case *ast.IfStmt:
			// else if: recursively analyze (no extra scope needed, the if will create its own)
			typedElseBranch = a.analyzeIfStatement(elseBranch)
		case *ast.BlockStmt:
			// else block: create scope
			a.enterScope()
			typedElseBranch = a.analyzeBlockStmt(elseBranch)
			a.exitScope()
		default:
			a.addError("unexpected else branch type", stmt.ElseBranch.Pos(), stmt.ElseBranch.End())
		}
	}

	return &TypedIfStmt{
		IfKeyword:   stmt.IfKeyword,
		Condition:   typedCond,
		ThenBranch:  typedThenBranch,
		ElseKeyword: stmt.ElseKeyword,
		ElseBranch:  typedElseBranch,
	}
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

	// Analyze the then branch (with its own scope)
	a.enterScope()
	typedThenBranch := a.analyzeBlockStmtForExpression(stmt.ThenBranch)
	thenType := a.getBlockResultType(typedThenBranch)
	a.exitScope()

	// Analyze the else branch
	var typedElseBranch TypedStatement
	var elseType Type

	switch elseBranch := stmt.ElseBranch.(type) {
	case *ast.IfStmt:
		// else if: recursively analyze as expression
		typedElseExpr := a.analyzeIfExpression(elseBranch)
		typedElseBranch = typedElseExpr.(*TypedIfStmt)
		elseType = typedElseExpr.GetType()
	case *ast.BlockStmt:
		// else block: create scope
		a.enterScope()
		typedBlock := a.analyzeBlockStmtForExpression(elseBranch)
		typedElseBranch = typedBlock
		elseType = a.getBlockResultType(typedBlock)
		a.exitScope()
	default:
		a.addError("unexpected else branch type", stmt.ElseBranch.Pos(), stmt.ElseBranch.End())
		elseType = TypeError
	}

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
	case *ast.FieldAccessExpr:
		return a.analyzeFieldAccessExpr(e)
	case *ast.GroupingExpr:
		// Grouping is purely syntactic for precedence; just analyze the inner expression
		return a.analyzeExpression(e.Expr)
	case *ast.IfStmt:
		// If can be used as an expression
		return a.analyzeIfExpression(e)
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
	typedArgs := make([]TypedExpression, len(call.Arguments))
	for i, arg := range call.Arguments {
		typedArgs[i] = a.analyzeExpression(arg)

		// Type check if we have a corresponding parameter
		if i < len(fnInfo.ParamTypes) {
			argType := typedArgs[i].GetType()
			paramType := fnInfo.ParamTypes[i]
			if _, isErr := argType.(ErrorType); !isErr && !paramType.Equals(argType) {
				a.addError(
					fmt.Sprintf("argument %d: expected %s, got %s",
						i+1, paramType.String(), argType.String()),
					arg.Pos(), arg.End(),
				)
			}
		}
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
		if i < len(structType.Fields) {
			argType := typedArgs[i].GetType()
			fieldType := structType.Fields[i].Type
			fieldName := structType.Fields[i].Name
			if _, isErr := argType.(ErrorType); !isErr && !fieldType.Equals(argType) {
				a.addError(
					fmt.Sprintf("field '%s': expected %s, got %s",
						fieldName, fieldType.String(), argType.String()),
					arg.Pos(), arg.End(),
				)
			}
		}
	}

	return &TypedStructLiteralExpr{
		Type:       structType,
		TypePos:    call.NamePos,
		LeftParen:  call.LeftParen,
		Args:       typedArgs,
		RightParen: call.RightParen,
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

		// Type check
		argType := typedArg.GetType()
		fieldType := structType.Fields[idx].Type
		if _, isErr := argType.(ErrorType); !isErr && !fieldType.Equals(argType) {
			a.addError(
				fmt.Sprintf("field '%s': expected %s, got %s",
					namedArg.Name, fieldType.String(), argType.String()),
				namedArg.Value.Pos(), namedArg.Value.End(),
			)
		}
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
		LeftParen:  call.LeftParen,
		Args:       typedArgs,
		RightParen: call.RightParen,
	}
}

// analyzeFieldAccessExpr analyzes a field access expression (e.g., p.x, rect.topLeft.x)
func (a *Analyzer) analyzeFieldAccessExpr(expr *ast.FieldAccessExpr) TypedExpression {
	// Analyze the object expression
	typedObject := a.analyzeExpression(expr.Object)
	objectType := typedObject.GetType()

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

	return &TypedFieldAccessExpr{
		Type:     fieldInfo.Type,
		Object:   typedObject,
		Dot:      expr.Dot,
		Field:    expr.Field,
		FieldPos: expr.FieldPos,
		Mutable:  fieldInfo.Mutable,
	}
}

// isAcceptedType checks if argType matches any of the accepted types
func isAcceptedType(argType Type, acceptedTypes []Type) bool {
	for _, accepted := range acceptedTypes {
		if accepted.Equals(argType) {
			return true
		}
		// Also allow compatible integer types when i64 is accepted
		if _, isI64 := accepted.(I64Type); isI64 && isIntegerType(argType) {
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
		return isIntegerType(argType)
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
	}

	return &TypedIdentifierExpr{
		Type:     typ,
		Name:     ident.Name,
		StartPos: ident.StartPos,
		EndPos:   ident.EndPos,
	}
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
		if !isIntegerType(leftType) && !isFloatType(leftType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires numeric operands, but left operand has type '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("arithmetic operators only work with numeric types")
			return TypeError
		}

		// Check right operand is numeric
		if !isIntegerType(rightType) && !isFloatType(rightType) {
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
		if op == "%" && isFloatType(leftType) {
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
		// Check left operand is numeric
		if !isIntegerType(leftType) && !isFloatType(leftType) {
			a.addError(
				fmt.Sprintf("operator '%s' requires numeric operands, but left operand has type '%s'", op, leftType.String()),
				leftPos, leftPos,
			).WithHint("comparison operators only work with numeric types")
			return TypeError
		}

		// Check right operand is numeric
		if !isIntegerType(rightType) && !isFloatType(rightType) {
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
	case I64Type, IntegerType:
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

// isIntegerType checks if a type is any integer type
func isIntegerType(t Type) bool {
	switch t.(type) {
	case IntegerType, I8Type, I16Type, I32Type, I64Type, I128Type,
		U8Type, U16Type, U32Type, U64Type, U128Type:
		return true
	}
	return false
}

// isFloatType checks if a type is any float type
func isFloatType(t Type) bool {
	switch t.(type) {
	case F32Type, F64Type:
		return true
	}
	return false
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

// branchReturns checks if an else branch (which can be a block or another if) returns.
func branchReturns(branch TypedStatement) bool {
	switch b := branch.(type) {
	case *TypedBlockStmt:
		return allPathsReturn(b.Statements)
	case *TypedIfStmt:
		// else if: recursively check
		return statementReturns(b)
	default:
		return false
	}
}

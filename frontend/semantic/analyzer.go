package semantic

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/seanrogers2657/slang/errors"
	"github.com/seanrogers2657/slang/frontend/ast"
)

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
	currentReturnType Type                    // return type of current function being analyzed
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(filename string) *Analyzer {
	return &Analyzer{
		filename:          filename,
		errors:            make([]*errors.CompilerError, 0),
		currentScope:      newScope(nil), // global scope
		functions:         make(map[string]FunctionInfo),
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

	// Handle function-based programs
	if len(program.Declarations) > 0 {
		// First pass: collect all function signatures
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
			// Add error if no main function found
			a.addError("program must have a 'main' function", program.StartPos, program.EndPos)
		}

		// Second pass: analyze function bodies
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

	// Convert parameter types
	paramTypes := make([]Type, len(fn.Parameters))
	for i, param := range fn.Parameters {
		paramTypes[i] = TypeFromName(param.TypeName)
		if _, isErr := paramTypes[i].(ErrorType); isErr {
			a.addError(fmt.Sprintf("unknown type '%s'", param.TypeName), param.TypePos, param.TypePos)
		}
	}

	// Convert return type
	returnType := TypeFromName(fn.ReturnType)
	if _, isErr := returnType.(ErrorType); isErr {
		a.addError(fmt.Sprintf("unknown type '%s'", fn.ReturnType), fn.ReturnPos, fn.ReturnPos)
	}

	a.functions[fn.Name] = FunctionInfo{
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}
}

// analyzeDeclaration performs semantic analysis on a declaration
func (a *Analyzer) analyzeDeclaration(decl ast.Declaration) TypedDeclaration {
	switch d := decl.(type) {
	case *ast.FunctionDecl:
		return a.analyzeFunctionDecl(d)
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

// analyzeStatement performs semantic analysis on a statement
func (a *Analyzer) analyzeStatement(stmt ast.Statement) TypedStatement {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return a.analyzeExprStatement(s)
	case *ast.PrintStmt:
		return a.analyzePrintStatement(s)
	case *ast.BlockStmt:
		return a.analyzeBlockStmt(s)
	case *ast.VarDeclStmt:
		return a.analyzeVarDeclStatement(s)
	case *ast.AssignStmt:
		return a.analyzeAssignStatement(s)
	case *ast.ReturnStmt:
		return a.analyzeReturnStatement(s)
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
		// Explicit type annotation
		declaredType = TypeFromName(stmt.TypeName)
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

// analyzeExprStatement analyzes an expression statement
func (a *Analyzer) analyzeExprStatement(stmt *ast.ExprStmt) TypedStatement {
	typedExpr := a.analyzeExpression(stmt.Expr)
	return &TypedExprStmt{
		Expr: typedExpr,
	}
}

// analyzePrintStatement analyzes a print statement
func (a *Analyzer) analyzePrintStatement(stmt *ast.PrintStmt) TypedStatement {
	typedExpr := a.analyzeExpression(stmt.Expr)

	// Print can handle any type, so no type checking needed here
	return &TypedPrintStmt{
		Keyword: stmt.Keyword,
		Expr:    typedExpr,
	}
}

// analyzeExpression performs semantic analysis on an expression
func (a *Analyzer) analyzeExpression(expr ast.Expression) TypedExpression {
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return a.analyzeLiteral(e)
	case *ast.BinaryExpr:
		return a.analyzeBinaryExpression(e)
	case *ast.IdentifierExpr:
		return a.analyzeIdentifier(e)
	case *ast.CallExpr:
		return a.analyzeCallExpr(e)
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

// checkBinaryOperation checks if a binary operation is valid and returns the result type
func (a *Analyzer) checkBinaryOperation(op string, leftType, rightType Type, leftPos, rightPos ast.Position) Type {
	// Check for error types - propagate them
	if _, ok := leftType.(ErrorType); ok {
		return TypeError
	}
	if _, ok := rightType.(ErrorType); ok {
		return TypeError
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
	// These require matching numeric types and return i64 (0 or 1)
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

		// Comparison result is an integer (0 or 1)
		return TypeInteger
	}

	// Unknown operator
	a.addError(fmt.Sprintf("unknown binary operator: '%s'", op), leftPos, rightPos)
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

package codegen

import (
	"fmt"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// LiteralInfo holds information about a collected literal.
type LiteralInfo struct {
	Value  string
	Label  string
	Length int  // for strings
	IsF64  bool // for floats
	Kind   ast.LiteralType
}

// ProgramInfo holds collected information about a program.
type ProgramInfo struct {
	FloatLiterals  map[string]LiteralInfo // label -> info
	StringLiterals []LiteralInfo
	HasPrint       bool
	HasBoolPrint   bool // true if print() is called with a boolean argument
}

// NewProgramInfo creates an empty ProgramInfo.
func NewProgramInfo() *ProgramInfo {
	return &ProgramInfo{
		FloatLiterals:  make(map[string]LiteralInfo),
		StringLiterals: make([]LiteralInfo, 0),
	}
}

// CollectFromTypedFunction scans a typed function for literals and print calls.
func (info *ProgramInfo) CollectFromTypedFunction(fn *semantic.TypedFunctionDecl) {
	floatIndex := len(info.FloatLiterals)
	stringIndex := len(info.StringLiterals)

	for _, stmt := range fn.Body.Statements {
		info.collectFromTypedStatement(stmt, &floatIndex, &stringIndex)
	}
}

func (info *ProgramInfo) collectFromTypedStatement(stmt semantic.TypedStatement, floatIdx, stringIdx *int) {
	switch s := stmt.(type) {
	case *semantic.TypedExprStmt:
		info.collectFromTypedExpr(s.Expr, floatIdx, stringIdx)
	case *semantic.TypedVarDeclStmt:
		info.collectFromTypedExpr(s.Initializer, floatIdx, stringIdx)
	case *semantic.TypedAssignStmt:
		info.collectFromTypedExpr(s.Value, floatIdx, stringIdx)
	case *semantic.TypedFieldAssignStmt:
		info.collectFromTypedExpr(s.Object, floatIdx, stringIdx)
		info.collectFromTypedExpr(s.Value, floatIdx, stringIdx)
	case *semantic.TypedReturnStmt:
		if s.Value != nil {
			info.collectFromTypedExpr(s.Value, floatIdx, stringIdx)
		}
	case *semantic.TypedIfStmt:
		// Collect from condition
		info.collectFromTypedExpr(s.Condition, floatIdx, stringIdx)
		// Collect from then branch
		for _, bodyStmt := range s.ThenBranch.Statements {
			info.collectFromTypedStatement(bodyStmt, floatIdx, stringIdx)
		}
		// Collect from else branch if present
		if s.ElseBranch != nil {
			info.collectFromTypedStatement(s.ElseBranch, floatIdx, stringIdx)
		}
	case *semantic.TypedBlockStmt:
		for _, bodyStmt := range s.Statements {
			info.collectFromTypedStatement(bodyStmt, floatIdx, stringIdx)
		}
	case *semantic.TypedForStmt:
		// Collect from init
		if s.Init != nil {
			info.collectFromTypedStatement(s.Init, floatIdx, stringIdx)
		}
		// Collect from condition
		if s.Condition != nil {
			info.collectFromTypedExpr(s.Condition, floatIdx, stringIdx)
		}
		// Collect from update
		if s.Update != nil {
			info.collectFromTypedStatement(s.Update, floatIdx, stringIdx)
		}
		// Collect from body
		for _, bodyStmt := range s.Body.Statements {
			info.collectFromTypedStatement(bodyStmt, floatIdx, stringIdx)
		}
	case *semantic.TypedBreakStmt:
		// Nothing to collect from break
	case *semantic.TypedContinueStmt:
		// Nothing to collect from continue
	default:
		// Unknown statement type - panic to catch missing cases during development
		panic(fmt.Sprintf("collectFromTypedStatement: unhandled statement type %T", stmt))
	}
}

func (info *ProgramInfo) collectFromTypedExpr(expr semantic.TypedExpression, floatIdx, stringIdx *int) {
	switch e := expr.(type) {
	case *semantic.TypedLiteralExpr:
		switch e.LitType {
		case ast.LiteralTypeFloat:
			label := fmt.Sprintf("float_%d", *floatIdx)
			(*floatIdx)++
			_, isF64 := e.Type.(semantic.F64Type)
			info.FloatLiterals[label] = LiteralInfo{
				Value: e.Value,
				Label: label,
				IsF64: isF64,
				Kind:  ast.LiteralTypeFloat,
			}
		case ast.LiteralTypeString:
			label := fmt.Sprintf("str_%d", *stringIdx)
			info.StringLiterals = append(info.StringLiterals, LiteralInfo{
				Value:  e.Value,
				Label:  label,
				Length: len(e.Value),
				Kind:   ast.LiteralTypeString,
			})
			(*stringIdx)++
		}

	case *semantic.TypedBinaryExpr:
		info.collectFromTypedExpr(e.Left, floatIdx, stringIdx)
		info.collectFromTypedExpr(e.Right, floatIdx, stringIdx)

	case *semantic.TypedCallExpr:
		if e.Name == "print" {
			info.HasPrint = true
			// Check if printing a boolean
			if len(e.Arguments) > 0 {
				if _, isBool := e.Arguments[0].GetType().(semantic.BooleanType); isBool {
					info.HasBoolPrint = true
				}
			}
		}
		for _, arg := range e.Arguments {
			info.collectFromTypedExpr(arg, floatIdx, stringIdx)
		}

	case *semantic.TypedUnaryExpr:
		info.collectFromTypedExpr(e.Operand, floatIdx, stringIdx)

	case *semantic.TypedIdentifierExpr:
		// Variable references don't contain literals - nothing to collect

	case *semantic.TypedIfStmt:
		// If expression: collect from condition and branches
		info.collectFromTypedExpr(e.Condition, floatIdx, stringIdx)
		for _, bodyStmt := range e.ThenBranch.Statements {
			info.collectFromTypedStatement(bodyStmt, floatIdx, stringIdx)
		}
		if e.ElseBranch != nil {
			info.collectFromTypedStatement(e.ElseBranch, floatIdx, stringIdx)
		}

	case *semantic.TypedStructLiteralExpr:
		// Collect literals from struct arguments
		for _, arg := range e.Args {
			info.collectFromTypedExpr(arg, floatIdx, stringIdx)
		}

	case *semantic.TypedFieldAccessExpr:
		// Collect from the object expression (though usually just identifiers)
		info.collectFromTypedExpr(e.Object, floatIdx, stringIdx)

	default:
		// Unknown expression type - panic to catch missing cases during development
		panic(fmt.Sprintf("collectFromTypedExpr: unhandled expression type %T", expr))
	}
}

// CountTypedVariables counts variable declarations in a typed statement list.
func CountTypedVariables(stmts []semantic.TypedStatement) int {
	count := 0
	for _, stmt := range stmts {
		count += countTypedVarsInStmt(stmt)
	}
	return count
}

// countStructSlots counts the total number of stack slots needed for a struct type
// (recursively counting nested struct fields)
func countStructSlots(structType semantic.StructType) int {
	count := 0
	for _, field := range structType.Fields {
		if nestedStruct, ok := field.Type.(semantic.StructType); ok {
			count += countStructSlots(nestedStruct)
		} else {
			count++
		}
	}
	return count
}

// countTypedVarsInStmt counts variables in a single statement (recursive)
// For struct types, counts the number of fields (each takes a stack slot)
func countTypedVarsInStmt(stmt semantic.TypedStatement) int {
	switch s := stmt.(type) {
	case *semantic.TypedVarDeclStmt:
		// Struct types take one slot per field (recursively counting nested structs)
		if structType, ok := s.DeclaredType.(semantic.StructType); ok {
			return countStructSlots(structType)
		}
		return 1
	case *semantic.TypedIfStmt:
		count := 0
		// Count in then branch
		for _, bodyStmt := range s.ThenBranch.Statements {
			count += countTypedVarsInStmt(bodyStmt)
		}
		// Count in else branch if present
		if s.ElseBranch != nil {
			count += countTypedVarsInStmt(s.ElseBranch)
		}
		return count
	case *semantic.TypedBlockStmt:
		count := 0
		for _, bodyStmt := range s.Statements {
			count += countTypedVarsInStmt(bodyStmt)
		}
		return count
	case *semantic.TypedForStmt:
		count := 0
		// Count in init
		if s.Init != nil {
			count += countTypedVarsInStmt(s.Init)
		}
		// Count in body
		for _, bodyStmt := range s.Body.Statements {
			count += countTypedVarsInStmt(bodyStmt)
		}
		return count
	default:
		return 0
	}
}

// FindStringLiteral finds a string literal by value in the collected literals.
func (info *ProgramInfo) FindStringLiteral(value string) (LiteralInfo, bool) {
	for _, lit := range info.StringLiterals {
		if lit.Value == value {
			return lit, true
		}
	}
	return LiteralInfo{}, false
}

// FindFloatLiteral finds a float literal by value in the collected literals.
func (info *ProgramInfo) FindFloatLiteral(value string) (string, LiteralInfo, bool) {
	for label, lit := range info.FloatLiterals {
		if lit.Value == value {
			return label, lit, true
		}
	}
	return "", LiteralInfo{}, false
}

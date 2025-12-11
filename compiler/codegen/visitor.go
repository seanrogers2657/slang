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
	case *semantic.TypedReturnStmt:
		if s.Value != nil {
			info.collectFromTypedExpr(s.Value, floatIdx, stringIdx)
		}
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
	}
}

// CountTypedVariables counts variable declarations in a typed statement list.
func CountTypedVariables(stmts []semantic.TypedStatement) int {
	count := 0
	for _, stmt := range stmts {
		if _, ok := stmt.(*semantic.TypedVarDeclStmt); ok {
			count++
		}
	}
	return count
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

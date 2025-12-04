package codegen

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/semantic"
)

// VariableInfo holds metadata about a variable's stack location and type.
type VariableInfo struct {
	Offset int
	Type   semantic.Type // nil for untyped codegen
}

// BaseContext provides shared context functionality for code generation.
// It tracks variable allocations, stack offsets, and source line information.
type BaseContext struct {
	variables   map[string]VariableInfo
	stackOffset int
	sourceLines []string
	stringMap   map[*ast.LiteralExpr]string // for AST-based codegen
}

// NewBaseContext creates a new code generation context.
func NewBaseContext(sourceLines []string) *BaseContext {
	return &BaseContext{
		variables:   make(map[string]VariableInfo),
		stackOffset: 0,
		sourceLines: sourceLines,
		stringMap:   make(map[*ast.LiteralExpr]string),
	}
}

// DeclareVariable allocates stack space for a variable and records its type.
// Returns the stack offset (positive, relative to frame pointer).
func (ctx *BaseContext) DeclareVariable(name string, typ semantic.Type) int {
	ctx.stackOffset += StackAlignment
	ctx.variables[name] = VariableInfo{Offset: ctx.stackOffset, Type: typ}
	return ctx.stackOffset
}

// GetVariable returns the variable info for a given name.
func (ctx *BaseContext) GetVariable(name string) (VariableInfo, bool) {
	v, ok := ctx.variables[name]
	return v, ok
}

// GetVariableOffset returns just the stack offset for a variable (for untyped codegen).
func (ctx *BaseContext) GetVariableOffset(name string) (int, bool) {
	v, ok := ctx.variables[name]
	return v.Offset, ok
}

// GetSourceLineComment returns a comment with the source line for a given position.
func (ctx *BaseContext) GetSourceLineComment(pos ast.Position) string {
	if ctx.sourceLines == nil || pos.Line <= 0 || pos.Line > len(ctx.sourceLines) {
		return ""
	}
	line := strings.TrimSpace(ctx.sourceLines[pos.Line-1])
	if line == "" {
		return ""
	}
	return fmt.Sprintf("// %d: %s\n", pos.Line, line)
}

// SetStringMap sets the string literal map for AST-based codegen.
func (ctx *BaseContext) SetStringMap(m map[*ast.LiteralExpr]string) {
	ctx.stringMap = m
}

// GetStringLabel returns the label for a string literal.
func (ctx *BaseContext) GetStringLabel(lit *ast.LiteralExpr) (string, bool) {
	label, ok := ctx.stringMap[lit]
	return label, ok
}

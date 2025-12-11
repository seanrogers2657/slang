package codegen

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// VariableInfo holds metadata about a variable's stack location and type.
type VariableInfo struct {
	Offset int
	Type   semantic.Type
}

// BaseContext provides shared context functionality for code generation.
// It tracks variable allocations, stack offsets, and source line information.
type BaseContext struct {
	variables    map[string]VariableInfo
	stackOffset  int
	sourceLines  []string
	labelCounter int // counter for generating unique labels
}

// NewBaseContext creates a new code generation context.
func NewBaseContext(sourceLines []string) *BaseContext {
	return &BaseContext{
		variables:   make(map[string]VariableInfo),
		stackOffset: 0,
		sourceLines: sourceLines,
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

// NextLabel generates a unique label with the given prefix.
// Used for generating branch targets in control flow (short-circuit evaluation, etc.)
// Note: Labels use underscore prefix (not .L) for slasm compatibility.
func (ctx *BaseContext) NextLabel(prefix string) string {
	ctx.labelCounter++
	return fmt.Sprintf("_%s_%d", prefix, ctx.labelCounter)
}

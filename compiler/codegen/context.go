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

// loopLabels holds the labels for break and continue statements within a loop.
type loopLabels struct {
	continueLabel string // label to jump to for continue (before update)
	breakLabel    string // label to jump to for break (after loop)
}

// BaseContext provides shared context functionality for code generation.
// It tracks variable allocations, stack offsets, and source line information.
type BaseContext struct {
	variables    map[string]VariableInfo
	stackOffset  int
	sourceLines  []string
	labelCounter int          // counter for generating unique labels
	loopStack    []loopLabels // stack of active loop labels for break/continue
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

// NextLabelID returns the next unique label ID without a prefix.
// Use this when you need multiple related labels to share the same ID.
func (ctx *BaseContext) NextLabelID() int {
	ctx.labelCounter++
	return ctx.labelCounter
}

// PushLoop pushes new loop labels onto the stack for break/continue support.
func (ctx *BaseContext) PushLoop(continueLabel, breakLabel string) {
	ctx.loopStack = append(ctx.loopStack, loopLabels{continueLabel, breakLabel})
}

// PopLoop removes the innermost loop from the stack.
func (ctx *BaseContext) PopLoop() {
	if len(ctx.loopStack) > 0 {
		ctx.loopStack = ctx.loopStack[:len(ctx.loopStack)-1]
	}
}

// CurrentLoop returns the labels for the innermost loop.
func (ctx *BaseContext) CurrentLoop() (continueLabel, breakLabel string, ok bool) {
	if len(ctx.loopStack) == 0 {
		return "", "", false
	}
	loop := ctx.loopStack[len(ctx.loopStack)-1]
	return loop.continueLabel, loop.breakLabel, true
}

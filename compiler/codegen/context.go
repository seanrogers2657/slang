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

// OwnedVarInfo tracks an owned pointer variable for cleanup.
type OwnedVarInfo struct {
	Name      string        // variable name
	Offset    int           // stack offset where pointer is stored
	AllocSize int           // size of heap allocation (for munmap)
	ElemType  semantic.Type // the element type (T in Own<T>)
}

// loopLabels holds the labels for break and continue statements within a loop.
type loopLabels struct {
	continueLabel string // label to jump to for continue (before update)
	breakLabel    string // label to jump to for break (after loop)
}

// BaseContext provides shared context functionality for code generation.
// It tracks variable allocations, stack offsets, and source line information.
type BaseContext struct {
	variables        map[string]VariableInfo
	stackOffset      int
	sourceLines      []string
	labelCounter     int            // local counter (used if sharedCounter is nil)
	sharedCounter    *int           // pointer to shared counter across methods (if not nil)
	loopStack        []loopLabels   // stack of active loop labels for break/continue
	ownedVars        []OwnedVarInfo // owned pointers that need cleanup (in declaration order)
	classReturnType  semantic.Type  // if non-nil, function returns a class by value (x8 has dest addr)
	x8SavedOffset    int            // stack offset where x8 (return destination) is saved
}

// NewBaseContext creates a new code generation context.
func NewBaseContext(sourceLines []string) *BaseContext {
	return &BaseContext{
		variables:   make(map[string]VariableInfo),
		stackOffset: 0,
		sourceLines: sourceLines,
	}
}

// NewBaseContextWithSharedCounter creates a context that shares a label counter
// with other contexts (used for class methods to avoid duplicate labels).
func NewBaseContextWithSharedCounter(sourceLines []string, sharedCounter *int) *BaseContext {
	return &BaseContext{
		variables:     make(map[string]VariableInfo),
		stackOffset:   0,
		sourceLines:   sourceLines,
		sharedCounter: sharedCounter,
	}
}

// DeclareVariable allocates stack space for a variable and records its type.
// Returns the stack offset (positive, relative to frame pointer).
// For nullable primitive types, allocates 16 bytes (tag + value).
// For nullable reference types, allocates 8 bytes (null pointer).
func (ctx *BaseContext) DeclareVariable(name string, typ semantic.Type) int {
	// Determine size needed for this type
	slots := 1 // default: 1 slot (8 bytes)
	if nullableType, isNullable := typ.(semantic.NullableType); isNullable {
		if !semantic.IsReferenceType(nullableType.InnerType) {
			// Nullable primitives need 2 slots: tag (8 bytes) + value (8 bytes)
			slots = 2
		}
		// Nullable references just need 1 slot (null pointer)
	}

	ctx.stackOffset += StackAlignment * slots
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
	if ctx.sharedCounter != nil {
		*ctx.sharedCounter++
		return fmt.Sprintf("_%s_%d", prefix, *ctx.sharedCounter)
	}
	ctx.labelCounter++
	return fmt.Sprintf("_%s_%d", prefix, ctx.labelCounter)
}

// NextLabelID returns the next unique label ID without a prefix.
// Use this when you need multiple related labels to share the same ID.
func (ctx *BaseContext) NextLabelID() int {
	if ctx.sharedCounter != nil {
		*ctx.sharedCounter++
		return *ctx.sharedCounter
	}
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

// RegisterOwnedVar registers an owned pointer variable for cleanup at scope exit.
// allocSize is the size of the heap allocation (for munmap).
func (ctx *BaseContext) RegisterOwnedVar(name string, offset int, allocSize int, elemType semantic.Type) {
	ctx.ownedVars = append(ctx.ownedVars, OwnedVarInfo{
		Name:      name,
		Offset:    offset,
		AllocSize: allocSize,
		ElemType:  elemType,
	})
}

// GetOwnedVars returns all owned pointer variables in declaration order.
func (ctx *BaseContext) GetOwnedVars() []OwnedVarInfo {
	return ctx.ownedVars
}

// HasOwnedVars returns true if there are any owned pointer variables to clean up.
func (ctx *BaseContext) HasOwnedVars() bool {
	return len(ctx.ownedVars) > 0
}

// MarkOwnedVarMoved marks an owned variable as moved, so it won't be deallocated.
// This is used when ownership is transferred (e.g., returned from function).
func (ctx *BaseContext) MarkOwnedVarMoved(name string) {
	for i := range ctx.ownedVars {
		if ctx.ownedVars[i].Name == name {
			// Remove from list by marking with empty name (or we could remove it)
			ctx.ownedVars[i].Name = ""
			return
		}
	}
}

// SetClassReturnType sets the return type for methods/functions returning class by value.
// The offset is where x8 (caller's destination address) is saved on the stack.
func (ctx *BaseContext) SetClassReturnType(typ semantic.Type, x8Offset int) {
	ctx.classReturnType = typ
	ctx.x8SavedOffset = x8Offset
}

// GetClassReturnType returns the class return type and x8 saved offset, if set.
func (ctx *BaseContext) GetClassReturnType() (semantic.Type, int, bool) {
	if ctx.classReturnType == nil {
		return nil, 0, false
	}
	return ctx.classReturnType, ctx.x8SavedOffset, true
}

// ReturnsClassByValue returns true if this context is for a function returning a class by value.
func (ctx *BaseContext) ReturnsClassByValue() bool {
	return ctx.classReturnType != nil
}

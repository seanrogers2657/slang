package codegen

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

func TestNewBaseContext(t *testing.T) {
	ctx := NewBaseContext(nil)

	if ctx.variables == nil {
		t.Error("variables map should be initialized")
	}
	if ctx.stackOffset != 0 {
		t.Errorf("stackOffset should be 0, got %d", ctx.stackOffset)
	}
}

func TestNewBaseContext_WithSourceLines(t *testing.T) {
	lines := []string{"line 1", "line 2"}
	ctx := NewBaseContext(lines)

	if len(ctx.sourceLines) != 2 {
		t.Errorf("expected 2 source lines, got %d", len(ctx.sourceLines))
	}
}

func TestBaseContext_DeclareVariable(t *testing.T) {
	ctx := NewBaseContext(nil)

	offset1 := ctx.DeclareVariable("x", semantic.TypeI64)
	if offset1 != 16 {
		t.Errorf("first variable should be at offset 16, got %d", offset1)
	}

	offset2 := ctx.DeclareVariable("y", semantic.TypeI32)
	if offset2 != 32 {
		t.Errorf("second variable should be at offset 32, got %d", offset2)
	}

	offset3 := ctx.DeclareVariable("z", nil)
	if offset3 != 48 {
		t.Errorf("third variable should be at offset 48, got %d", offset3)
	}
}

func TestBaseContext_GetVariable(t *testing.T) {
	ctx := NewBaseContext(nil)
	ctx.DeclareVariable("x", semantic.TypeI64)

	info, found := ctx.GetVariable("x")
	if !found {
		t.Error("variable x should be found")
	}
	if info.Offset != 16 {
		t.Errorf("expected offset 16, got %d", info.Offset)
	}
	if info.Type != semantic.TypeI64 {
		t.Errorf("expected type I64, got %v", info.Type)
	}

	_, found = ctx.GetVariable("notfound")
	if found {
		t.Error("variable notfound should not be found")
	}
}

func TestBaseContext_GetSourceLineComment(t *testing.T) {
	lines := []string{
		"fn main() {",
		"    val x = 42",
		"    print(x)",
		"}",
	}
	ctx := NewBaseContext(lines)

	// Valid line
	comment := ctx.GetSourceLineComment(ast.Position{Line: 2})
	if !strings.Contains(comment, "val x = 42") {
		t.Errorf("expected source line comment, got %q", comment)
	}
	if !strings.Contains(comment, "// 2:") {
		t.Errorf("expected line number prefix, got %q", comment)
	}

	// Line 0 (invalid)
	comment = ctx.GetSourceLineComment(ast.Position{Line: 0})
	if comment != "" {
		t.Errorf("expected empty comment for line 0, got %q", comment)
	}

	// Negative line (invalid)
	comment = ctx.GetSourceLineComment(ast.Position{Line: -1})
	if comment != "" {
		t.Errorf("expected empty comment for negative line, got %q", comment)
	}

	// Out of bounds
	comment = ctx.GetSourceLineComment(ast.Position{Line: 100})
	if comment != "" {
		t.Errorf("expected empty comment for out of bounds line, got %q", comment)
	}
}

func TestBaseContext_GetSourceLineComment_EmptyLine(t *testing.T) {
	lines := []string{
		"fn main() {",
		"",
		"    print(x)",
	}
	ctx := NewBaseContext(lines)

	// Empty line should return empty string
	comment := ctx.GetSourceLineComment(ast.Position{Line: 2})
	if comment != "" {
		t.Errorf("expected empty comment for empty line, got %q", comment)
	}
}

func TestBaseContext_GetSourceLineComment_WhitespaceOnly(t *testing.T) {
	lines := []string{
		"fn main() {",
		"   \t   ",
		"    print(x)",
	}
	ctx := NewBaseContext(lines)

	// Whitespace-only line should return empty string
	comment := ctx.GetSourceLineComment(ast.Position{Line: 2})
	if comment != "" {
		t.Errorf("expected empty comment for whitespace line, got %q", comment)
	}
}

func TestBaseContext_GetSourceLineComment_NilSourceLines(t *testing.T) {
	ctx := NewBaseContext(nil)

	comment := ctx.GetSourceLineComment(ast.Position{Line: 1})
	if comment != "" {
		t.Errorf("expected empty comment when sourceLines is nil, got %q", comment)
	}
}

func TestBaseContext_MultipleVariables(t *testing.T) {
	ctx := NewBaseContext(nil)

	// Declare multiple variables with different types
	ctx.DeclareVariable("a", semantic.TypeI8)
	ctx.DeclareVariable("b", semantic.TypeI16)
	ctx.DeclareVariable("c", semantic.TypeI32)
	ctx.DeclareVariable("d", semantic.TypeI64)
	ctx.DeclareVariable("e", semantic.TypeFloat32)
	ctx.DeclareVariable("f", semantic.TypeFloat64)

	// All should be found
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		_, found := ctx.GetVariable(name)
		if !found {
			t.Errorf("variable %s should be found", name)
		}
	}

	// Check offsets are incrementing
	aInfo, _ := ctx.GetVariable("a")
	fInfo, _ := ctx.GetVariable("f")

	if aInfo.Offset >= fInfo.Offset {
		t.Error("later variables should have larger offsets")
	}
}

func TestBaseContext_NextLabel(t *testing.T) {
	ctx := NewBaseContext(nil)

	label1 := ctx.NextLabel("and_end")
	if label1 != "_and_end_1" {
		t.Errorf("expected _and_end_1, got %s", label1)
	}

	label2 := ctx.NextLabel("or_true")
	if label2 != "_or_true_2" {
		t.Errorf("expected _or_true_2, got %s", label2)
	}

	label3 := ctx.NextLabel("and_end")
	if label3 != "_and_end_3" {
		t.Errorf("expected _and_end_3, got %s", label3)
	}
}

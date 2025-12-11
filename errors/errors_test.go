package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompilerError(t *testing.T) {
	t.Run("creates error with message", func(t *testing.T) {
		err := NewError("test error", "test.sl", Position{Line: 1, Column: 5}, "parser")

		if err.Message != "test error" {
			t.Errorf("expected message 'test error', got %q", err.Message)
		}
		if err.Filename != "test.sl" {
			t.Errorf("expected filename 'test.sl', got %q", err.Filename)
		}
		if err.Position.Line != 1 {
			t.Errorf("expected line 1, got %d", err.Position.Line)
		}
		if err.Position.Column != 5 {
			t.Errorf("expected column 5, got %d", err.Position.Column)
		}
		if err.Stage != "parser" {
			t.Errorf("expected stage 'parser', got %q", err.Stage)
		}
		if err.Kind != ErrorKindError {
			t.Errorf("expected error kind, got %v", err.Kind)
		}
	})

	t.Run("creates error with span", func(t *testing.T) {
		err := NewErrorWithSpan("test error", "test.sl",
			Position{Line: 1, Column: 5},
			Position{Line: 1, Column: 10},
			"semantic")

		if err.EndPos.Column != 10 {
			t.Errorf("expected end column 10, got %d", err.EndPos.Column)
		}
	})

	t.Run("creates warning", func(t *testing.T) {
		warn := NewWarning("test warning", "test.sl", Position{Line: 2, Column: 3}, "semantic")

		if warn.Kind != ErrorKindWarning {
			t.Errorf("expected warning kind, got %v", warn.Kind)
		}
	})

	t.Run("adds hint to error", func(t *testing.T) {
		err := NewError("test error", "test.sl", Position{Line: 1, Column: 1}, "parser").
			WithHint("try using parentheses")

		if err.Hint != "try using parentheses" {
			t.Errorf("expected hint, got %q", err.Hint)
		}
	})

	t.Run("adds tool to error", func(t *testing.T) {
		err := NewError("test error", "test.sl", Position{Line: 1, Column: 1}, "parser").
			WithTool(ToolSL)

		if err.Tool != ToolSL {
			t.Errorf("expected tool %q, got %q", ToolSL, err.Tool)
		}
	})

	t.Run("error implements error interface", func(t *testing.T) {
		err := NewError("test error", "test.sl", Position{Line: 1, Column: 1}, "parser")

		if err.Error() != "test error" {
			t.Errorf("expected Error() to return message, got %q", err.Error())
		}
	})
}

func TestErrorKindString(t *testing.T) {
	tests := []struct {
		kind     ErrorKind
		expected string
	}{
		{ErrorKindWarning, "Warning"},
		{ErrorKindError, "Error"},
		{ErrorKindFatal, "Fatal"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.kind.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.kind.String())
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	// Disable colors for testing
	DisableColors = true
	defer func() { DisableColors = false }()

	sourceLines := []string{
		"print 5 + 3",
		"print 10 * 2",
		"print \"hello\"",
	}

	t.Run("formats error with source context", func(t *testing.T) {
		err := NewError("unexpected token", "test.sl",
			Position{Line: 2, Column: 10}, "lexer")

		formatted := FormatError(err, sourceLines)
		t.Logf("Formatted output:\n%s", formatted)

		// Check for key components
		if !strings.Contains(formatted, "Error: unexpected token") {
			t.Errorf("formatted error should contain error message, got:\n%s", formatted)
		}
		if !strings.Contains(formatted, "test.sl:2:10") {
			t.Error("formatted error should contain file location")
		}
		if !strings.Contains(formatted, "print 10 * 2") {
			t.Error("formatted error should contain source line")
		}
		if !strings.Contains(formatted, "^") {
			t.Error("formatted error should contain error pointer")
		}
	})

	t.Run("formats error with tool name", func(t *testing.T) {
		err := NewError("type mismatch", "test.sl",
			Position{Line: 1, Column: 7}, "semantic").
			WithTool(ToolSL)

		formatted := FormatError(err, sourceLines)
		t.Logf("Formatted output:\n%s", formatted)

		if !strings.Contains(formatted, "sl:") {
			t.Error("formatted error should contain tool name")
		}
		if !strings.Contains(formatted, "(semantic)") {
			t.Error("formatted error should contain stage")
		}
	})

	t.Run("formats error with slasm tool", func(t *testing.T) {
		err := NewError("invalid instruction", "test.s",
			Position{Line: 3, Column: 5}, "assemble").
			WithTool(ToolSlasm)

		formatted := FormatError(err, sourceLines)
		t.Logf("Formatted output:\n%s", formatted)

		if !strings.Contains(formatted, "slasm:") {
			t.Error("formatted error should contain tool name 'slasm'")
		}
		if !strings.Contains(formatted, "(assemble)") {
			t.Error("formatted error should contain stage 'assemble'")
		}
	})

	t.Run("formats error with span", func(t *testing.T) {
		err := NewErrorWithSpan("type mismatch", "test.sl",
			Position{Line: 1, Column: 7},
			Position{Line: 1, Column: 11},
			"semantic")

		formatted := FormatError(err, sourceLines)

		// Should have multiple carets for span
		if !strings.Contains(formatted, "^^^^^") {
			t.Error("formatted error should show span with multiple carets")
		}
	})

	t.Run("formats error with hint", func(t *testing.T) {
		err := NewError("invalid syntax", "test.sl",
			Position{Line: 3, Column: 7}, "parser").
			WithHint("strings must be enclosed in double quotes")

		formatted := FormatError(err, sourceLines)

		if !strings.Contains(formatted, "help: strings must be enclosed in double quotes") {
			t.Error("formatted error should contain hint")
		}
	})

	t.Run("formats warning with different color", func(t *testing.T) {
		warn := NewWarning("unused variable", "test.sl",
			Position{Line: 1, Column: 1}, "semantic")

		formatted := FormatError(warn, sourceLines)

		if !strings.Contains(formatted, "Warning:") {
			t.Error("formatted warning should be labeled as Warning")
		}
	})
}

func TestFormatErrors(t *testing.T) {
	// Disable colors for testing
	DisableColors = true
	defer func() { DisableColors = false }()

	sourceLines := []string{
		"print 5 + 3",
		"print 10 * 2",
	}

	t.Run("formats multiple errors", func(t *testing.T) {
		errors := []*CompilerError{
			NewError("error 1", "test.sl", Position{Line: 1, Column: 1}, "lexer"),
			NewError("error 2", "test.sl", Position{Line: 2, Column: 1}, "parser"),
		}

		formatted := FormatErrors(errors, sourceLines)

		if !strings.Contains(formatted, "error 1") {
			t.Error("should contain first error")
		}
		if !strings.Contains(formatted, "error 2") {
			t.Error("should contain second error")
		}
		if !strings.Contains(formatted, "Compilation failed with 2 error(s)") {
			t.Error("should contain error count summary")
		}
	})

	t.Run("formats errors and warnings together", func(t *testing.T) {
		errors := []*CompilerError{
			NewError("error 1", "test.sl", Position{Line: 1, Column: 1}, "semantic"),
			NewWarning("warning 1", "test.sl", Position{Line: 2, Column: 1}, "semantic"),
		}

		formatted := FormatErrors(errors, sourceLines)

		if !strings.Contains(formatted, "1 error(s) and 1 warning(s)") {
			t.Error("should contain both error and warning count")
		}
	})

	t.Run("formats only warnings", func(t *testing.T) {
		errors := []*CompilerError{
			NewWarning("warning 1", "test.sl", Position{Line: 1, Column: 1}, "semantic"),
		}

		formatted := FormatErrors(errors, sourceLines)

		if !strings.Contains(formatted, "Compilation succeeded with 1 warning(s)") {
			t.Error("should indicate success with warnings")
		}
	})
}

func TestHandler(t *testing.T) {
	// Disable colors for testing
	DisableColors = true
	defer func() { DisableColors = false }()

	t.Run("creates handler with tool", func(t *testing.T) {
		handler := NewHandler(ToolSL)

		if handler.Tool != ToolSL {
			t.Errorf("expected tool %q, got %q", ToolSL, handler.Tool)
		}
	})

	t.Run("wraps error with tool", func(t *testing.T) {
		handler := NewHandler(ToolSlasm)
		err := handler.Wrap(fmt.Errorf("test error"), "assemble")

		if err.Tool != ToolSlasm {
			t.Errorf("expected tool %q, got %q", ToolSlasm, err.Tool)
		}
		if err.Stage != "assemble" {
			t.Errorf("expected stage 'assemble', got %q", err.Stage)
		}
	})

	t.Run("wraps error with position", func(t *testing.T) {
		handler := NewHandler(ToolSL)
		pos := Position{Line: 5, Column: 10}
		err := handler.WrapWithPos(fmt.Errorf("test error"), "test.sl", pos, "parser")

		if err.Position.Line != 5 {
			t.Errorf("expected line 5, got %d", err.Position.Line)
		}
		if err.Filename != "test.sl" {
			t.Errorf("expected filename 'test.sl', got %q", err.Filename)
		}
	})

	t.Run("returns 0 for no errors", func(t *testing.T) {
		handler := NewHandler(ToolSL)
		exitCode := handler.Handle([]*CompilerError{})

		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})
}

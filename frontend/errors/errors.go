package errors

import (
	"github.com/seanrogers2657/slang/frontend/ast"
)

// ErrorKind represents the severity of a compiler error
type ErrorKind int

const (
	ErrorKindWarning ErrorKind = iota
	ErrorKindError
	ErrorKindFatal
)

func (k ErrorKind) String() string {
	switch k {
	case ErrorKindWarning:
		return "Warning"
	case ErrorKindError:
		return "Error"
	case ErrorKindFatal:
		return "Fatal"
	default:
		return "Unknown"
	}
}

// CompilerError represents a detailed error with source location
type CompilerError struct {
	Message  string       // Error description
	Filename string       // Source file path
	Position ast.Position // Where the error occurred
	EndPos   ast.Position // End position (for multi-character spans)
	Stage    string       // "lexer", "parser", "semantic", "codegen"
	Kind     ErrorKind    // Warning, Error, Fatal
	Hint     string       // Optional suggestion for fixing the error
}

// Error implements the error interface
func (e *CompilerError) Error() string {
	return e.Message
}

// NewError creates a new compiler error
func NewError(message string, filename string, pos ast.Position, stage string) *CompilerError {
	return &CompilerError{
		Message:  message,
		Filename: filename,
		Position: pos,
		EndPos:   pos,
		Stage:    stage,
		Kind:     ErrorKindError,
	}
}

// NewErrorWithSpan creates a new compiler error with a position span
func NewErrorWithSpan(message string, filename string, startPos, endPos ast.Position, stage string) *CompilerError {
	return &CompilerError{
		Message:  message,
		Filename: filename,
		Position: startPos,
		EndPos:   endPos,
		Stage:    stage,
		Kind:     ErrorKindError,
	}
}

// NewWarning creates a new compiler warning
func NewWarning(message string, filename string, pos ast.Position, stage string) *CompilerError {
	return &CompilerError{
		Message:  message,
		Filename: filename,
		Position: pos,
		EndPos:   pos,
		Stage:    stage,
		Kind:     ErrorKindWarning,
	}
}

// WithHint adds a hint to the error
func (e *CompilerError) WithHint(hint string) *CompilerError {
	e.Hint = hint
	return e
}

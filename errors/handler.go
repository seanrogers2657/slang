package errors

import (
	"fmt"
	"os"
)

// Handler provides consistent error handling for CLI tools
type Handler struct {
	Tool        Tool
	SourceCache map[string][]string
}

// NewHandler creates a new error handler for a specific tool
func NewHandler(tool Tool) *Handler {
	return &Handler{
		Tool:        tool,
		SourceCache: make(map[string][]string),
	}
}

// Handle processes and displays errors, returns an appropriate exit code
// Returns 0 if no errors, 1 if there are errors
func (h *Handler) Handle(errs []*CompilerError) int {
	if len(errs) == 0 {
		return 0
	}

	// Ensure all errors have the tool set
	for _, err := range errs {
		if err.Tool == "" {
			err.Tool = h.Tool
		}
	}

	// Get source lines for each unique file
	for _, err := range errs {
		if err.Filename != "" && err.Position.Line > 0 {
			if _, ok := h.SourceCache[err.Filename]; !ok {
				lines, readErr := ReadSourceLines(err.Filename)
				if readErr == nil {
					h.SourceCache[err.Filename] = lines
				}
			}
		}
	}

	// Format and print each error
	for i, err := range errs {
		lines := h.SourceCache[err.Filename]
		fmt.Fprint(os.Stderr, FormatError(err, lines))
		if i < len(errs)-1 {
			fmt.Fprintln(os.Stderr)
		}
	}

	// Print summary
	errorCount := 0
	warningCount := 0
	for _, err := range errs {
		if err.Kind == ErrorKindWarning {
			warningCount++
		} else {
			errorCount++
		}
	}

	fmt.Fprintln(os.Stderr)
	if errorCount > 0 {
		fmt.Fprintf(os.Stderr, "%s%s: %d error(s)",
			color(colorBold), h.Tool, errorCount)
		if warningCount > 0 {
			fmt.Fprintf(os.Stderr, " and %d warning(s)", warningCount)
		}
		fmt.Fprintf(os.Stderr, "%s\n", color(colorReset))
		return 1
	} else if warningCount > 0 {
		fmt.Fprintf(os.Stderr, "%s%s: %d warning(s)%s\n",
			color(colorYellow), h.Tool, warningCount, color(colorReset))
	}

	return 0
}

// Wrap converts a plain error to a CompilerError with the handler's tool
func (h *Handler) Wrap(err error, stage string) *CompilerError {
	return &CompilerError{
		Tool:    h.Tool,
		Stage:   stage,
		Message: err.Error(),
		Kind:    ErrorKindError,
	}
}

// WrapWithFile converts a plain error to a CompilerError with file info
func (h *Handler) WrapWithFile(err error, filename string, stage string) *CompilerError {
	return &CompilerError{
		Tool:     h.Tool,
		Stage:    stage,
		Message:  err.Error(),
		Filename: filename,
		Kind:     ErrorKindError,
	}
}

// WrapWithPos converts a plain error to a CompilerError with full position info
func (h *Handler) WrapWithPos(err error, filename string, pos Position, stage string) *CompilerError {
	return &CompilerError{
		Tool:     h.Tool,
		Stage:    stage,
		Message:  err.Error(),
		Filename: filename,
		Position: pos,
		EndPos:   pos,
		Kind:     ErrorKindError,
	}
}

// NewError creates a new error with the handler's tool
func (h *Handler) NewError(message string, filename string, pos Position, stage string) *CompilerError {
	return &CompilerError{
		Tool:     h.Tool,
		Stage:    stage,
		Message:  message,
		Filename: filename,
		Position: pos,
		EndPos:   pos,
		Kind:     ErrorKindError,
	}
}

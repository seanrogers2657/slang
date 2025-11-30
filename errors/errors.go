package errors

// Tool identifies which tool generated an error
type Tool string

const (
	ToolSL    Tool = "sl"
	ToolSlasm Tool = "slasm"
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
	Tool     Tool     // which tool generated this error (sl, slasm)
	Stage    string   // "lexer", "parser", "semantic", "codegen", "assemble", "link"
	Message  string   // error description
	Filename string   // source file path
	Position Position // where the error occurred
	EndPos   Position // end position (for multi-character spans)
	Kind     ErrorKind
	Hint     string // optional suggestion for fixing the error
}

// Error implements the error interface
func (e *CompilerError) Error() string {
	return e.Message
}

// NewError creates a new compiler error
func NewError(message string, filename string, pos Position, stage string) *CompilerError {
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
func NewErrorWithSpan(message string, filename string, startPos, endPos Position, stage string) *CompilerError {
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
func NewWarning(message string, filename string, pos Position, stage string) *CompilerError {
	return &CompilerError{
		Message:  message,
		Filename: filename,
		Position: pos,
		EndPos:   pos,
		Stage:    stage,
		Kind:     ErrorKindWarning,
	}
}

// WithTool sets the tool that generated this error
func (e *CompilerError) WithTool(tool Tool) *CompilerError {
	e.Tool = tool
	return e
}

// WithHint adds a hint to the error
func (e *CompilerError) WithHint(hint string) *CompilerError {
	e.Hint = hint
	return e
}

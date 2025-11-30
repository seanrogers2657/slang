package errors

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// DisableColors disables color output for testing
var DisableColors = false

func color(code string) string {
	if DisableColors {
		return ""
	}
	return code
}

// FormatError formats a single compiler error with source context
func FormatError(err *CompilerError, sourceLines []string) string {
	var builder strings.Builder

	// Error header with tool name and color
	// Format: tool: Kind: message
	errorColor := colorRed
	if err.Kind == ErrorKindWarning {
		errorColor = colorYellow
	}

	// Include tool name if set
	if err.Tool != "" {
		builder.WriteString(fmt.Sprintf("%s%s%s: %s%s%s: %s\n",
			color(colorBold), err.Tool, color(colorReset),
			color(colorBold), color(errorColor), err.Kind.String(), color(colorReset)))
		builder.WriteString(fmt.Sprintf("  %s\n", err.Message))
	} else {
		builder.WriteString(fmt.Sprintf("%s%s%s%s: %s\n",
			color(colorBold), color(errorColor), err.Kind.String(), color(colorReset), err.Message))
	}

	// Location info with stage
	// Format:   --> file:line:column (stage)
	if err.Filename != "" {
		if err.Position.Line > 0 {
			builder.WriteString(fmt.Sprintf("%s  --> %s:%d:%d",
				color(colorCyan), err.Filename, err.Position.Line, err.Position.Column))
		} else {
			builder.WriteString(fmt.Sprintf("%s  --> %s",
				color(colorCyan), err.Filename))
		}
	} else if err.Stage != "" {
		builder.WriteString(fmt.Sprintf("%s  -->", color(colorCyan)))
	}

	if err.Stage != "" {
		builder.WriteString(fmt.Sprintf(" (%s)", err.Stage))
	}
	builder.WriteString(fmt.Sprintf("%s\n", color(colorReset)))

	// Source context
	if err.Position.Line > 0 && err.Position.Line <= len(sourceLines) {
		lineNum := err.Position.Line
		sourceLine := sourceLines[lineNum-1]

		// Line number gutter width (for alignment)
		gutterWidth := len(fmt.Sprintf("%d", lineNum))

		// Separator line
		builder.WriteString(fmt.Sprintf("%s%*s |%s\n",
			color(colorCyan), gutterWidth, "", color(colorReset)))

		// Source line with line number
		builder.WriteString(fmt.Sprintf("%s%*d |%s %s\n",
			color(colorCyan), gutterWidth, lineNum, color(colorReset), sourceLine))

		// Error pointer line
		builder.WriteString(fmt.Sprintf("%s%*s |%s ",
			color(colorCyan), gutterWidth, "", color(colorReset)))

		// Calculate spacing and underlining
		if err.Position.Column > 0 {
			// Add spaces to align with error position
			for i := 1; i < err.Position.Column; i++ {
				builder.WriteString(" ")
			}

			// Add carets to underline the error
			spanLength := 1
			if err.EndPos.Line == err.Position.Line && err.EndPos.Column > err.Position.Column {
				spanLength = err.EndPos.Column - err.Position.Column + 1
			}

			builder.WriteString(color(errorColor))
			for i := 0; i < spanLength; i++ {
				builder.WriteString("^")
			}
			builder.WriteString(color(colorReset))
		}

		builder.WriteString("\n")
	}

	// Hint
	if err.Hint != "" {
		if err.Position.Line > 0 {
			builder.WriteString(fmt.Sprintf("%s%*s |%s %shelp:%s %s\n",
				color(colorCyan), len(fmt.Sprintf("%d", err.Position.Line)), "", color(colorReset),
				color(colorCyan), color(colorReset), err.Hint))
		} else {
			builder.WriteString(fmt.Sprintf("  %shelp:%s %s\n",
				color(colorCyan), color(colorReset), err.Hint))
		}
	}

	return builder.String()
}

// FormatErrors formats multiple compiler errors
func FormatErrors(errors []*CompilerError, sourceLines []string) string {
	var builder strings.Builder

	for i, err := range errors {
		builder.WriteString(FormatError(err, sourceLines))
		if i < len(errors)-1 {
			builder.WriteString("\n")
		}
	}

	// Summary
	errorCount := 0
	warningCount := 0
	for _, err := range errors {
		if err.Kind == ErrorKindWarning {
			warningCount++
		} else {
			errorCount++
		}
	}

	builder.WriteString("\n")
	if errorCount > 0 {
		builder.WriteString(fmt.Sprintf("%sCompilation failed with %d error(s)",
			color(colorBold), errorCount))
		if warningCount > 0 {
			builder.WriteString(fmt.Sprintf(" and %d warning(s)", warningCount))
		}
		builder.WriteString(color(colorReset))
	} else if warningCount > 0 {
		builder.WriteString(fmt.Sprintf("%sCompilation succeeded with %d warning(s)%s",
			color(colorYellow), warningCount, color(colorReset)))
	}
	builder.WriteString("\n")

	return builder.String()
}

// ReadSourceLines reads a source file and returns its lines
func ReadSourceLines(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	return lines, nil
}

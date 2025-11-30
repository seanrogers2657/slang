package slasm

import (
	"fmt"
	"io"
	"os"
)

// Logger provides configurable logging for the assembler pipeline
type Logger struct {
	enabled bool
	writer  io.Writer
}

// NewLogger creates a new logger
func NewLogger(enabled bool, writer io.Writer) *Logger {
	if writer == nil {
		writer = os.Stderr
	}
	return &Logger{
		enabled: enabled,
		writer:  writer,
	}
}

// NewDefaultLogger creates a logger that writes to stderr
func NewDefaultLogger(enabled bool) *Logger {
	return NewLogger(enabled, os.Stderr)
}

// NewSilentLogger creates a logger that discards all output
func NewSilentLogger() *Logger {
	return NewLogger(false, io.Discard)
}

// Printf formats and prints a message if logging is enabled
func (l *Logger) Printf(format string, args ...any) {
	if l.enabled {
		fmt.Fprintf(l.writer, format, args...)
	}
}

// Println prints a message with a newline if logging is enabled
func (l *Logger) Println(args ...any) {
	if l.enabled {
		fmt.Fprintln(l.writer, args...)
	}
}

// Header prints a formatted header section
func (l *Logger) Header(title string) {
	if l.enabled {
		l.Printf("\n%s\n", title)
		l.Printf("%s\n", repeat("=", len(title)))
	}
}

// Section prints a formatted section header
func (l *Logger) Section(title string) {
	if l.enabled {
		l.Printf("\n%s\n", title)
		l.Printf("%s\n", repeat("-", len(title)))
	}
}

// Enabled returns whether logging is enabled
func (l *Logger) Enabled() bool {
	return l.enabled
}

// SetEnabled enables or disables logging
func (l *Logger) SetEnabled(enabled bool) {
	l.enabled = enabled
}

// SetWriter changes the output writer
func (l *Logger) SetWriter(writer io.Writer) {
	if writer != nil {
		l.writer = writer
	}
}

// Helper function to repeat a string
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

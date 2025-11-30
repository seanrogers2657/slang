package errors

// Position represents a location in source code
type Position struct {
	Line   int // line number (1-indexed)
	Column int // column number (1-indexed)
	Offset int // byte offset (0-indexed)
}

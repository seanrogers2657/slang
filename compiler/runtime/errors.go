// Package runtime provides runtime support for Slang programs,
// including panic handling and error codes.
package runtime

// RuntimeError represents a runtime error code
type RuntimeError int

const (
	// Signed integer overflow
	ErrOverflowAddSigned RuntimeError = 1 // integer overflow: addition
	ErrOverflowSubSigned RuntimeError = 2 // integer overflow: subtraction
	ErrOverflowMulSigned RuntimeError = 3 // integer overflow: multiplication

	// Unsigned integer overflow/underflow
	ErrOverflowAddUnsigned  RuntimeError = 4 // unsigned overflow: addition
	ErrUnderflowSubUnsigned RuntimeError = 5 // unsigned underflow: subtraction
	ErrOverflowMulUnsigned  RuntimeError = 6 // unsigned overflow: multiplication

	// Division errors
	ErrDivByZero RuntimeError = 7 // division by zero
	ErrModByZero RuntimeError = 8 // modulo by zero

	// Array errors
	ErrIndexOutOfBounds RuntimeError = 9 // array index out of bounds

	// Reserved for future use
	// ErrNilPointer       RuntimeError = 10
	// ErrStackOverflow    RuntimeError = 11
)

// ErrorMessages maps error codes to human-readable messages
var ErrorMessages = map[RuntimeError]string{
	ErrOverflowAddSigned:    "integer overflow: addition",
	ErrOverflowSubSigned:    "integer overflow: subtraction",
	ErrOverflowMulSigned:    "integer overflow: multiplication",
	ErrOverflowAddUnsigned:  "unsigned overflow: addition",
	ErrUnderflowSubUnsigned: "unsigned underflow: subtraction",
	ErrOverflowMulUnsigned:  "unsigned overflow: multiplication",
	ErrDivByZero:            "division by zero",
	ErrModByZero:            "modulo by zero",
	ErrIndexOutOfBounds:     "array index out of bounds",
}

// String returns the human-readable message for an error code
func (e RuntimeError) String() string {
	if msg, ok := ErrorMessages[e]; ok {
		return msg
	}
	return "unknown runtime error"
}

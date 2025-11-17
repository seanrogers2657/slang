package semantic

// Type represents a type in the Slang type system
type Type interface {
	String() string          // Human-readable name
	Equals(other Type) bool  // Type equality check
}

// IntegerType represents the integer type
type IntegerType struct{}

func (t IntegerType) String() string {
	return "integer"
}

func (t IntegerType) Equals(other Type) bool {
	_, ok := other.(IntegerType)
	return ok
}

// StringType represents the string type
type StringType struct{}

func (t StringType) String() string {
	return "string"
}

func (t StringType) Equals(other Type) bool {
	_, ok := other.(StringType)
	return ok
}

// BooleanType represents the boolean type (for comparison results)
type BooleanType struct{}

func (t BooleanType) String() string {
	return "boolean"
}

func (t BooleanType) Equals(other Type) bool {
	_, ok := other.(BooleanType)
	return ok
}

// VoidType represents no type (for statements)
type VoidType struct{}

func (t VoidType) String() string {
	return "void"
}

func (t VoidType) Equals(other Type) bool {
	_, ok := other.(VoidType)
	return ok
}

// ErrorType represents a type error
type ErrorType struct{}

func (t ErrorType) String() string {
	return "<error>"
}

func (t ErrorType) Equals(other Type) bool {
	_, ok := other.(ErrorType)
	return ok
}

// Common type instances
var (
	TypeInteger = IntegerType{}
	TypeString  = StringType{}
	TypeBoolean = BooleanType{}
	TypeVoid    = VoidType{}
	TypeError   = ErrorType{}
)

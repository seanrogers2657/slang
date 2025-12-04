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

// FunctionType represents a function type with parameter and return types
type FunctionType struct {
	ParamTypes []Type
	ReturnType Type
}

func (t FunctionType) String() string {
	params := ""
	for i, p := range t.ParamTypes {
		if i > 0 {
			params += ", "
		}
		params += p.String()
	}
	return "fn(" + params + "): " + t.ReturnType.String()
}

func (t FunctionType) Equals(other Type) bool {
	o, ok := other.(FunctionType)
	if !ok {
		return false
	}
	if len(t.ParamTypes) != len(o.ParamTypes) {
		return false
	}
	for i, pt := range t.ParamTypes {
		if !pt.Equals(o.ParamTypes[i]) {
			return false
		}
	}
	return t.ReturnType.Equals(o.ReturnType)
}

// Common type instances
var (
	TypeInteger = IntegerType{}
	TypeString  = StringType{}
	TypeBoolean = BooleanType{}
	TypeVoid    = VoidType{}
	TypeError   = ErrorType{}
)

// TypeFromName converts a type name string to a Type
func TypeFromName(name string) Type {
	switch name {
	case "int":
		return TypeInteger
	case "string":
		return TypeString
	case "bool":
		return TypeBoolean
	case "void":
		return TypeVoid
	default:
		return TypeError
	}
}

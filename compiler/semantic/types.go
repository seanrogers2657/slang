package semantic

// Type represents a type in the Slang type system
type Type interface {
	String() string         // Human-readable name
	Equals(other Type) bool // Type equality check
}

// NumericType provides bit width info for numeric types
type NumericType interface {
	Type
	BitWidth() int  // Size in bits
	IsSigned() bool // Whether the type is signed
	IsFloat() bool  // Whether the type is a floating point type
}

// Signed integer types
type I8Type struct{}

func (t I8Type) String() string         { return "i8" }
func (t I8Type) BitWidth() int          { return 8 }
func (t I8Type) IsSigned() bool         { return true }
func (t I8Type) IsFloat() bool          { return false }
func (t I8Type) Equals(other Type) bool { _, ok := other.(I8Type); return ok }

type I16Type struct{}

func (t I16Type) String() string         { return "i16" }
func (t I16Type) BitWidth() int          { return 16 }
func (t I16Type) IsSigned() bool         { return true }
func (t I16Type) IsFloat() bool          { return false }
func (t I16Type) Equals(other Type) bool { _, ok := other.(I16Type); return ok }

type I32Type struct{}

func (t I32Type) String() string         { return "i32" }
func (t I32Type) BitWidth() int          { return 32 }
func (t I32Type) IsSigned() bool         { return true }
func (t I32Type) IsFloat() bool          { return false }
func (t I32Type) Equals(other Type) bool { _, ok := other.(I32Type); return ok }

type I64Type struct{}

func (t I64Type) String() string { return "i64" }
func (t I64Type) BitWidth() int  { return 64 }
func (t I64Type) IsSigned() bool { return true }
func (t I64Type) IsFloat() bool  { return false }
func (t I64Type) Equals(other Type) bool {
	_, ok := other.(I64Type)
	return ok
}

type I128Type struct{}

func (t I128Type) String() string         { return "i128" }
func (t I128Type) BitWidth() int          { return 128 }
func (t I128Type) IsSigned() bool         { return true }
func (t I128Type) IsFloat() bool          { return false }
func (t I128Type) Equals(other Type) bool { _, ok := other.(I128Type); return ok }

// Unsigned integer types
type U8Type struct{}

func (t U8Type) String() string         { return "u8" }
func (t U8Type) BitWidth() int          { return 8 }
func (t U8Type) IsSigned() bool         { return false }
func (t U8Type) IsFloat() bool          { return false }
func (t U8Type) Equals(other Type) bool { _, ok := other.(U8Type); return ok }

type U16Type struct{}

func (t U16Type) String() string         { return "u16" }
func (t U16Type) BitWidth() int          { return 16 }
func (t U16Type) IsSigned() bool         { return false }
func (t U16Type) IsFloat() bool          { return false }
func (t U16Type) Equals(other Type) bool { _, ok := other.(U16Type); return ok }

type U32Type struct{}

func (t U32Type) String() string         { return "u32" }
func (t U32Type) BitWidth() int          { return 32 }
func (t U32Type) IsSigned() bool         { return false }
func (t U32Type) IsFloat() bool          { return false }
func (t U32Type) Equals(other Type) bool { _, ok := other.(U32Type); return ok }

type U64Type struct{}

func (t U64Type) String() string         { return "u64" }
func (t U64Type) BitWidth() int          { return 64 }
func (t U64Type) IsSigned() bool         { return false }
func (t U64Type) IsFloat() bool          { return false }
func (t U64Type) Equals(other Type) bool { _, ok := other.(U64Type); return ok }

type U128Type struct{}

func (t U128Type) String() string         { return "u128" }
func (t U128Type) BitWidth() int          { return 128 }
func (t U128Type) IsSigned() bool         { return false }
func (t U128Type) IsFloat() bool          { return false }
func (t U128Type) Equals(other Type) bool { _, ok := other.(U128Type); return ok }

// Floating point types
type F32Type struct{}

func (t F32Type) String() string         { return "f32" }
func (t F32Type) BitWidth() int          { return 32 }
func (t F32Type) IsSigned() bool         { return true }
func (t F32Type) IsFloat() bool          { return true }
func (t F32Type) Equals(other Type) bool { _, ok := other.(F32Type); return ok }

type F64Type struct{}

func (t F64Type) String() string         { return "f64" }
func (t F64Type) BitWidth() int          { return 64 }
func (t F64Type) IsSigned() bool         { return true }
func (t F64Type) IsFloat() bool          { return true }
func (t F64Type) Equals(other Type) bool { _, ok := other.(F64Type); return ok }

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

// StructFieldInfo holds information about a struct field
type StructFieldInfo struct {
	Name    string // field name
	Type    Type   // field type
	Mutable bool   // true for var, false for val
	Index   int    // field index (position in struct)
}

// StructType represents a struct type with named fields
type StructType struct {
	Name   string            // struct name
	Fields []StructFieldInfo // list of fields
}

func (t StructType) String() string {
	return t.Name
}

func (t StructType) Equals(other Type) bool {
	o, ok := other.(StructType)
	if !ok {
		return false
	}
	// Nominal type equality - structs are equal if they have the same name
	return t.Name == o.Name
}

// GetField returns field info by name
func (t StructType) GetField(name string) (StructFieldInfo, bool) {
	for _, f := range t.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return StructFieldInfo{}, false
}

// FieldOffset returns the byte offset of a field from the struct start
// Each field is 8 bytes (aligned)
func (t StructType) FieldOffset(name string) int {
	offset := 0
	for _, f := range t.Fields {
		if f.Name == name {
			return offset
		}
		offset += 8 // all fields are 8-byte aligned
	}
	return -1 // field not found
}

// Size returns the total size of the struct in bytes
func (t StructType) Size() int {
	total := 0
	for _, field := range t.Fields {
		total += TypeByteSize(field.Type)
	}
	return total
}

// ArraySizeUnknown indicates that an array's size is not yet known.
// This is used when parsing type annotations like Array<i64> where the size
// will be inferred from the literal. A value of -1 distinguishes "unknown"
// from "zero elements" (which is an error for array literals).
const ArraySizeUnknown = -1

// ArrayType represents a fixed-size array type
type ArrayType struct {
	ElementType Type // element type (e.g., I64Type)
	Size        int  // fixed size (known at compile time), or ArraySizeUnknown
}

func (t ArrayType) String() string {
	return "Array<" + t.ElementType.String() + ">"
}

func (t ArrayType) Equals(other Type) bool {
	o, ok := other.(ArrayType)
	if !ok {
		return false
	}
	return t.Size == o.Size && t.ElementType.Equals(o.ElementType)
}

// ElementSize returns the byte size of each element based on the element type
func (t ArrayType) ElementSize() int {
	return TypeByteSize(t.ElementType)
}

// TotalSize returns the total byte size of the array
func (t ArrayType) TotalSize() int {
	return t.Size * t.ElementSize()
}

// TypeByteSize returns the byte size of a type for memory allocation purposes.
// For numeric types, this is derived from the bit width.
// For pointers (strings), this is 8 bytes on 64-bit systems.
// For composite types, this is computed from their structure.
func TypeByteSize(t Type) int {
	// Check if it implements NumericType for bit-width based sizing
	if numType, ok := t.(NumericType); ok {
		return numType.BitWidth() / 8
	}

	// Handle other types
	switch t.(type) {
	case StringType:
		return 8 // pointer size on 64-bit
	case BooleanType:
		return 1 // logically 1 byte, though may be padded in practice
	case VoidType:
		return 0
	case ErrorType:
		return 0
	case StructType:
		return t.(StructType).Size()
	case ArrayType:
		return t.(ArrayType).TotalSize()
	default:
		return 8 // default to 8 bytes for unknown types
	}
}

// Common type instances
var (
	// Default types
	TypeInteger = I64Type{} // default integer (i64)
	TypeString  = StringType{}
	TypeBoolean = BooleanType{}
	TypeVoid    = VoidType{}
	TypeError   = ErrorType{}

	// Signed integer types
	TypeI8   = I8Type{}
	TypeI16  = I16Type{}
	TypeI32  = I32Type{}
	TypeI64  = I64Type{}
	TypeI128 = I128Type{}

	// Unsigned integer types
	TypeU8   = U8Type{}
	TypeU16  = U16Type{}
	TypeU32  = U32Type{}
	TypeU64  = U64Type{}
	TypeU128 = U128Type{}

	// Floating point types
	TypeFloat32 = F32Type{}
	TypeFloat64 = F64Type{}
)

// IsIntegerType checks if a type is any integer type
func IsIntegerType(t Type) bool {
	switch t.(type) {
	case I8Type, I16Type, I32Type, I64Type, I128Type,
		U8Type, U16Type, U32Type, U64Type, U128Type:
		return true
	}
	return false
}

// IsFloatType checks if a type is any float type
func IsFloatType(t Type) bool {
	switch t.(type) {
	case F32Type, F64Type:
		return true
	}
	return false
}

// IsArrayType checks if a type is an array type
func IsArrayType(t Type) bool {
	_, ok := t.(ArrayType)
	return ok
}

// TypeFromName converts a type name string to a Type
func TypeFromName(name string) Type {
	switch name {
	// Legacy/alias names
	case "int":
		return TypeInteger
	case "string":
		return TypeString
	case "bool":
		return TypeBoolean
	case "void":
		return TypeVoid

	// Signed integers
	case "i8":
		return TypeI8
	case "i16":
		return TypeI16
	case "i32":
		return TypeI32
	case "i64":
		return TypeI64
	case "i128":
		return TypeI128

	// Unsigned integers
	case "u8":
		return TypeU8
	case "u16":
		return TypeU16
	case "u32":
		return TypeU32
	case "u64":
		return TypeU64
	case "u128":
		return TypeU128

	// Floating point
	case "f32":
		return TypeFloat32
	case "f64":
		return TypeFloat64

	default:
		return TypeError
	}
}

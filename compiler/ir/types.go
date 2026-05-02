package ir

import "fmt"

// Type represents a type in the IR type system.
type Type interface {
	// String returns a human-readable representation of the type.
	String() string

	// Size returns the size of the type in bytes.
	Size() int

	// Align returns the alignment requirement in bytes.
	Align() int

	// Equal returns true if this type equals another type.
	Equal(other Type) bool

	irType() // marker method
}

// IntType represents signed or unsigned integer types.
type IntType struct {
	Bits   int  // 8, 16, 32, 64, or 128
	Signed bool // true for signed, false for unsigned
}

func (t *IntType) String() string {
	prefix := "u"
	if t.Signed {
		prefix = "s"
	}
	return fmt.Sprintf("%s%d", prefix, t.Bits)
}

func (t *IntType) Size() int {
	return t.Bits / 8
}

func (t *IntType) Align() int {
	// Alignment is same as size, up to 8 bytes
	if t.Bits <= 64 {
		return t.Bits / 8
	}
	return 8 // 128-bit types align to 8 bytes
}

func (t *IntType) Equal(other Type) bool {
	if o, ok := other.(*IntType); ok {
		return t.Bits == o.Bits && t.Signed == o.Signed
	}
	return false
}

func (t *IntType) irType() {}

// Common integer types
var (
	TypeS8   = &IntType{Bits: 8, Signed: true}
	TypeS16  = &IntType{Bits: 16, Signed: true}
	TypeS32  = &IntType{Bits: 32, Signed: true}
	TypeS64  = &IntType{Bits: 64, Signed: true}
	TypeS128 = &IntType{Bits: 128, Signed: true}
	TypeU8   = &IntType{Bits: 8, Signed: false}
	TypeU16  = &IntType{Bits: 16, Signed: false}
	TypeU32  = &IntType{Bits: 32, Signed: false}
	TypeU64  = &IntType{Bits: 64, Signed: false}
	TypeU128 = &IntType{Bits: 128, Signed: false}
)

// BoolType represents boolean values.
type BoolType struct{}

func (t *BoolType) String() string { return "bool" }
func (t *BoolType) Size() int      { return 1 }
func (t *BoolType) Align() int     { return 1 }

func (t *BoolType) Equal(other Type) bool {
	_, ok := other.(*BoolType)
	return ok
}

func (t *BoolType) irType() {}

// TypeBool is the singleton boolean type.
var TypeBool = &BoolType{}

// VoidType represents the absence of a value (for functions with no return).
type VoidType struct{}

func (t *VoidType) String() string { return "void" }
func (t *VoidType) Size() int      { return 0 }
func (t *VoidType) Align() int     { return 1 }

func (t *VoidType) Equal(other Type) bool {
	_, ok := other.(*VoidType)
	return ok
}

func (t *VoidType) irType() {}

// TypeVoid is the singleton void type.
var TypeVoid = &VoidType{}

// PtrType represents a pointer to another type.
type PtrType struct {
	Elem Type // element type
}

func (t *PtrType) String() string {
	return "*" + t.Elem.String()
}

func (t *PtrType) Size() int  { return 8 } // 64-bit pointers
func (t *PtrType) Align() int { return 8 }

func (t *PtrType) Equal(other Type) bool {
	if o, ok := other.(*PtrType); ok {
		return t.Elem.Equal(o.Elem)
	}
	return false
}

func (t *PtrType) irType() {}

// ArrayType represents a fixed-size array.
type ArrayType struct {
	Elem Type // element type
	Len  int  // number of elements
}

func (t *ArrayType) String() string {
	return fmt.Sprintf("[%d]%s", t.Len, t.Elem.String())
}

func (t *ArrayType) Size() int {
	return t.Elem.Size() * t.Len
}

func (t *ArrayType) Align() int {
	return t.Elem.Align()
}

func (t *ArrayType) Equal(other Type) bool {
	if o, ok := other.(*ArrayType); ok {
		return t.Len == o.Len && t.Elem.Equal(o.Elem)
	}
	return false
}

func (t *ArrayType) irType() {}

// StructField represents a field in a struct type.
type StructField struct {
	Name   string
	Type   Type
	Offset int // byte offset from struct start (computed during layout)
}

// StructType represents a user-defined struct.
type StructType struct {
	Name   string
	Fields []StructField
	size   int // cached size (computed during layout)
	align  int // cached alignment (computed during layout)
}

func (t *StructType) String() string {
	return t.Name
}

func (t *StructType) Size() int {
	return t.size
}

func (t *StructType) Align() int {
	if t.align == 0 {
		return 8 // default alignment
	}
	return t.align
}

func (t *StructType) Equal(other Type) bool {
	if o, ok := other.(*StructType); ok {
		return t.Name == o.Name
	}
	return false
}

func (t *StructType) irType() {}

// ComputeLayout computes field offsets and struct size with proper alignment.
func (t *StructType) ComputeLayout() {
	offset := 0
	maxAlign := 1

	for i := range t.Fields {
		field := &t.Fields[i]
		align := field.Type.Align()
		if align > maxAlign {
			maxAlign = align
		}

		// Pad to alignment
		if offset%align != 0 {
			offset += align - (offset % align)
		}

		field.Offset = offset
		offset += field.Type.Size()
	}

	// Pad struct size to alignment (for arrays of structs)
	if offset%maxAlign != 0 {
		offset += maxAlign - (offset % maxAlign)
	}

	t.size = offset
	t.align = maxAlign
}

// FieldByName returns the field with the given name, or nil if not found.
func (t *StructType) FieldByName(name string) *StructField {
	for i := range t.Fields {
		if t.Fields[i].Name == name {
			return &t.Fields[i]
		}
	}
	return nil
}

// NullableType represents a nullable wrapper type (T?).
type NullableType struct {
	Elem Type // the wrapped type
}

func (t *NullableType) String() string {
	return t.Elem.String() + "?"
}

func (t *NullableType) Size() int {
	return 8
}

func (t *NullableType) Align() int {
	return 8
}

func (t *NullableType) Equal(other Type) bool {
	if o, ok := other.(*NullableType); ok {
		return t.Elem.Equal(o.Elem)
	}
	return false
}

func (t *NullableType) irType() {}

// IsReferenceNullable reports whether the wrapped element is itself a pointer
// or struct reference. All nullables share an 8-byte pointer representation,
// but reference-element nullables carry the pointer through directly while
// value-element nullables (s64?, bool?, ...) require a heap slot at wrap time
// and a load at unwrap time.
func (t *NullableType) IsReferenceNullable() bool {
	switch t.Elem.(type) {
	case *PtrType, *StructType:
		return true
	default:
		return false
	}
}

// FuncType represents a function signature.
type FuncType struct {
	Params []Type
	Return Type // nil for void
}

func (t *FuncType) String() string {
	result := "("
	for i, p := range t.Params {
		if i > 0 {
			result += ", "
		}
		result += p.String()
	}
	result += ")"
	if t.Return != nil {
		result += " -> " + t.Return.String()
	}
	return result
}

func (t *FuncType) Size() int  { return 8 } // function pointer
func (t *FuncType) Align() int { return 8 }

func (t *FuncType) Equal(other Type) bool {
	o, ok := other.(*FuncType)
	if !ok {
		return false
	}
	if len(t.Params) != len(o.Params) {
		return false
	}
	for i := range t.Params {
		if !t.Params[i].Equal(o.Params[i]) {
			return false
		}
	}
	if t.Return == nil && o.Return == nil {
		return true
	}
	if t.Return == nil || o.Return == nil {
		return false
	}
	return t.Return.Equal(o.Return)
}

func (t *FuncType) irType() {}

// StringType represents string values.
// Strings are represented as a pointer to bytes plus a length.
type StringType struct{}

func (t *StringType) String() string { return "string" }
func (t *StringType) Size() int      { return 16 } // ptr (8) + len (8)
func (t *StringType) Align() int     { return 8 }

func (t *StringType) Equal(other Type) bool {
	_, ok := other.(*StringType)
	return ok
}

func (t *StringType) irType() {}

// TypeString is the singleton string type.
var TypeString = &StringType{}

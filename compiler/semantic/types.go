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
type S8Type struct{}

func (t S8Type) String() string         { return "s8" }
func (t S8Type) BitWidth() int          { return 8 }
func (t S8Type) IsSigned() bool         { return true }
func (t S8Type) IsFloat() bool          { return false }
func (t S8Type) Equals(other Type) bool { _, ok := other.(S8Type); return ok }

type S16Type struct{}

func (t S16Type) String() string         { return "s16" }
func (t S16Type) BitWidth() int          { return 16 }
func (t S16Type) IsSigned() bool         { return true }
func (t S16Type) IsFloat() bool          { return false }
func (t S16Type) Equals(other Type) bool { _, ok := other.(S16Type); return ok }

type S32Type struct{}

func (t S32Type) String() string         { return "s32" }
func (t S32Type) BitWidth() int          { return 32 }
func (t S32Type) IsSigned() bool         { return true }
func (t S32Type) IsFloat() bool          { return false }
func (t S32Type) Equals(other Type) bool { _, ok := other.(S32Type); return ok }

type S64Type struct{}

func (t S64Type) String() string { return "s64" }
func (t S64Type) BitWidth() int  { return 64 }
func (t S64Type) IsSigned() bool { return true }
func (t S64Type) IsFloat() bool  { return false }
func (t S64Type) Equals(other Type) bool {
	_, ok := other.(S64Type)
	return ok
}

type S128Type struct{}

func (t S128Type) String() string         { return "s128" }
func (t S128Type) BitWidth() int          { return 128 }
func (t S128Type) IsSigned() bool         { return true }
func (t S128Type) IsFloat() bool          { return false }
func (t S128Type) Equals(other Type) bool { _, ok := other.(S128Type); return ok }

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

// ============================================================================
// Shared field/method helpers for StructType, ClassType, and ObjectType
// ============================================================================

// getFieldFromList finds a field by name in a slice of StructFieldInfo.
// Returns the field info and true if found, or empty info and false if not found.
func getFieldFromList(fields []StructFieldInfo, name string) (StructFieldInfo, bool) {
	for _, f := range fields {
		if f.Name == name {
			return f, true
		}
	}
	return StructFieldInfo{}, false
}

// fieldOffsetFromList returns the byte offset of a field from the start of the struct/class.
// Each field is 8 bytes (aligned). Returns -1 if field not found.
func fieldOffsetFromList(fields []StructFieldInfo, name string) int {
	offset := 0
	for _, f := range fields {
		if f.Name == name {
			return offset
		}
		offset += 8 // all fields are 8-byte aligned
	}
	return -1 // field not found
}

// sizeFromFields returns the total size in bytes for a list of fields.
func sizeFromFields(fields []StructFieldInfo) int {
	total := 0
	for _, field := range fields {
		total += TypeByteSize(field.Type)
	}
	return total
}

// getMethodFromMap finds method overloads by name in a method map.
// Returns the method list and true if found, or nil and false if not found.
func getMethodFromMap(methods map[string][]*MethodInfo, name string) ([]*MethodInfo, bool) {
	m, ok := methods[name]
	return m, ok
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
	return getFieldFromList(t.Fields, name)
}

// FieldOffset returns the byte offset of a field from the struct start
// Each field is 8 bytes (aligned)
func (t StructType) FieldOffset(name string) int {
	return fieldOffsetFromList(t.Fields, name)
}

// Size returns the total size of the struct in bytes
func (t StructType) Size() int {
	return sizeFromFields(t.Fields)
}

// MethodInfo holds information about a method in a class
type MethodInfo struct {
	Name       string // method name
	ParamTypes []Type // parameter types (includes self type for instance methods)
	ParamNames []string // parameter names (includes "self" for instance methods)
	ReturnType Type   // return type (VoidType for void methods)
	IsStatic   bool   // true if first param is not 'self'
}

// ClassType represents a class type with fields and methods
type ClassType struct {
	Name    string                     // class name
	Fields  []StructFieldInfo          // list of fields (reuses struct field info)
	Methods map[string][]*MethodInfo   // methods by name (slice for overloading support)
}

func (t ClassType) String() string {
	return t.Name
}

func (t ClassType) Equals(other Type) bool {
	o, ok := other.(ClassType)
	if !ok {
		return false
	}
	// Nominal type equality - classes are equal if they have the same name
	return t.Name == o.Name
}

// GetField returns field info by name
func (t ClassType) GetField(name string) (StructFieldInfo, bool) {
	return getFieldFromList(t.Fields, name)
}

// GetMethod returns all overloads for a method by name
func (t ClassType) GetMethod(name string) ([]*MethodInfo, bool) {
	return getMethodFromMap(t.Methods, name)
}

// FieldOffset returns the byte offset of a field from the class start
// Each field is 8 bytes (aligned)
func (t ClassType) FieldOffset(name string) int {
	return fieldOffsetFromList(t.Fields, name)
}

// Size returns the total size of the class instance in bytes
func (t ClassType) Size() int {
	return sizeFromFields(t.Fields)
}

// ObjectType represents a singleton object type (static methods only, no fields)
type ObjectType struct {
	Name    string                   // object name
	Methods map[string][]*MethodInfo // methods by name (all must be static)
}

func (t ObjectType) String() string {
	return t.Name
}

func (t ObjectType) Equals(other Type) bool {
	o, ok := other.(ObjectType)
	if !ok {
		return false
	}
	// Nominal type equality - objects are equal if they have the same name
	return t.Name == o.Name
}

// GetMethod returns all overloads for a method by name
func (t ObjectType) GetMethod(name string) ([]*MethodInfo, bool) {
	return getMethodFromMap(t.Methods, name)
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

// NullableType wraps a type to indicate it may be null (T?)
type NullableType struct {
	InnerType Type // the non-nullable inner type
}

func (t NullableType) String() string {
	return t.InnerType.String() + "?"
}

func (t NullableType) Equals(other Type) bool {
	o, ok := other.(NullableType)
	if !ok {
		return false
	}
	return t.InnerType.Equals(o.InnerType)
}

// NothingType is the type of 'null', assignable to any T?
// This is the bottom type for nullable types.
type NothingType struct{}

func (t NothingType) String() string {
	return "Nothing"
}

func (t NothingType) Equals(other Type) bool {
	_, ok := other.(NothingType)
	return ok
}

// OwnedPointerType represents an owned pointer type Own<T>
// Owned pointers provide unique ownership of heap-allocated values.
// When an owned pointer goes out of scope, its memory is automatically freed.
type OwnedPointerType struct {
	ElementType Type // the type being pointed to (e.g., Point for *Point)
}

func (t OwnedPointerType) String() string {
	return "*" + t.ElementType.String()
}

func (t OwnedPointerType) Equals(other Type) bool {
	o, ok := other.(OwnedPointerType)
	if !ok {
		return false
	}
	return t.ElementType.Equals(o.ElementType)
}

// IsCopyable returns false for owned pointers (they are move-only)
func (t OwnedPointerType) IsCopyable() bool {
	return false
}

// RefPointerType represents an immutable borrowed reference type Ref<T>
// References provide temporary read-only access to a value without taking ownership.
// References can only appear in function parameter position.
type RefPointerType struct {
	ElementType Type // the type being pointed to (e.g., Point for &Point)
}

func (t RefPointerType) String() string {
	return "&" + t.ElementType.String()
}

func (t RefPointerType) Equals(other Type) bool {
	o, ok := other.(RefPointerType)
	if !ok {
		return false
	}
	return t.ElementType.Equals(o.ElementType)
}

// IsCopyable returns true for references (they are copyable within their scope)
func (t RefPointerType) IsCopyable() bool {
	return true
}

// MutRefPointerType represents a mutable borrowed reference type MutRef<T>
// Mutable references allow mutation of var fields in the referenced value.
// Mutable references can only appear in function parameter position.
type MutRefPointerType struct {
	ElementType Type // the type being pointed to (e.g., Point for &&Point)
}

func (t MutRefPointerType) String() string {
	return "&&" + t.ElementType.String()
}

func (t MutRefPointerType) Equals(other Type) bool {
	o, ok := other.(MutRefPointerType)
	if !ok {
		return false
	}
	return t.ElementType.Equals(o.ElementType)
}

// IsCopyable returns true for mutable references (they are copyable within their scope)
func (t MutRefPointerType) IsCopyable() bool {
	return true
}

// IsNullable checks if a type is nullable (T?)
func IsNullable(t Type) bool {
	_, ok := t.(NullableType)
	return ok
}

// MakeNullable wraps a type in NullableType, avoiding double-wrapping
func MakeNullable(t Type) Type {
	if IsNullable(t) {
		return t // don't double-wrap
	}
	return NullableType{InnerType: t}
}

// UnwrapNullable extracts the inner type from a NullableType.
// Returns (innerType, true) if t is nullable, (t, false) otherwise.
func UnwrapNullable(t Type) (Type, bool) {
	if n, ok := t.(NullableType); ok {
		return n.InnerType, true
	}
	return t, false
}

// IsReferenceType checks if a type is a reference type (struct, class, string, pointer).
// Reference types use 8-byte nullable pointers; primitives use 16-byte tagged unions.
func IsReferenceType(t Type) bool {
	switch t.(type) {
	case StructType, ClassType, StringType, OwnedPointerType, RefPointerType, MutRefPointerType:
		return true
	default:
		return false
	}
}

// IsOwnedPointer checks if a type is an owned pointer type Own<T>
func IsOwnedPointer(t Type) bool {
	_, ok := t.(OwnedPointerType)
	return ok
}

// IsNullableOwnedPointer checks if a type is a nullable owned pointer type Own<T>?
func IsNullableOwnedPointer(t Type) bool {
	if nullable, ok := t.(NullableType); ok {
		_, isOwned := nullable.InnerType.(OwnedPointerType)
		return isOwned
	}
	return false
}

// UnwrapOwnedPointer extracts the inner type from an OwnedPointerType.
// Returns (innerType, true) if t is an owned pointer, (t, false) otherwise.
func UnwrapOwnedPointer(t Type) (Type, bool) {
	if o, ok := t.(OwnedPointerType); ok {
		return o.ElementType, true
	}
	return t, false
}

// IsRefPointer checks if a type is an immutable reference pointer type Ref<T>
func IsRefPointer(t Type) bool {
	_, ok := t.(RefPointerType)
	return ok
}

// UnwrapRefPointer extracts the inner type from a RefPointerType.
// Returns (innerType, true) if t is a ref pointer, (t, false) otherwise.
func UnwrapRefPointer(t Type) (Type, bool) {
	if r, ok := t.(RefPointerType); ok {
		return r.ElementType, true
	}
	return t, false
}

// IsMutRefPointer checks if a type is a mutable reference pointer type MutRef<T>
func IsMutRefPointer(t Type) bool {
	_, ok := t.(MutRefPointerType)
	return ok
}

// UnwrapMutRefPointer extracts the inner type from a MutRefPointerType.
// Returns (innerType, true) if t is a mutable ref pointer, (t, false) otherwise.
func UnwrapMutRefPointer(t Type) (Type, bool) {
	if r, ok := t.(MutRefPointerType); ok {
		return r.ElementType, true
	}
	return t, false
}

// IsAnyRefPointer checks if a type is any reference pointer type (Ref<T> or MutRef<T>)
func IsAnyRefPointer(t Type) bool {
	return IsRefPointer(t) || IsMutRefPointer(t)
}

// UnwrapAnyRefPointer extracts the inner type from any reference pointer type.
// Returns (innerType, isMutable, true) if t is a ref pointer, (t, false, false) otherwise.
func UnwrapAnyRefPointer(t Type) (Type, bool, bool) {
	if r, ok := t.(RefPointerType); ok {
		return r.ElementType, false, true
	}
	if r, ok := t.(MutRefPointerType); ok {
		return r.ElementType, true, true
	}
	return t, false, false
}

// IsClassType checks if a type is a class type
func IsClassType(t Type) bool {
	_, ok := t.(ClassType)
	return ok
}

// IsObjectType checks if a type is an object type (singleton)
func IsObjectType(t Type) bool {
	_, ok := t.(ObjectType)
	return ok
}

// UnwrapClassFromPointer extracts the ClassType from any pointer type.
// Returns (classType, true) if the pointer's element is a class, (ClassType{}, false) otherwise.
func UnwrapClassFromPointer(t Type) (ClassType, bool) {
	var elementType Type

	switch pt := t.(type) {
	case OwnedPointerType:
		elementType = pt.ElementType
	case RefPointerType:
		elementType = pt.ElementType
	case MutRefPointerType:
		elementType = pt.ElementType
	default:
		return ClassType{}, false
	}

	if ct, ok := elementType.(ClassType); ok {
		return ct, true
	}
	return ClassType{}, false
}

// GetUnderlyingType extracts the underlying type from pointer wrappers.
// For *T, &T, &&T: returns T
// For other types: returns the type itself
func GetUnderlyingType(t Type) Type {
	switch pt := t.(type) {
	case OwnedPointerType:
		return pt.ElementType
	case RefPointerType:
		return pt.ElementType
	case MutRefPointerType:
		return pt.ElementType
	default:
		return t
	}
}

// IsAssignableTo checks if the source type can be assigned to the target type.
// This includes exact equality plus:
// - T -> T? coercion (non-nullable to nullable)
// - NothingType (null) -> T? coercion
func IsAssignableTo(source, target Type) bool {
	// Exact equality
	if source.Equals(target) {
		return true
	}

	// T -> T? coercion: source T can be assigned to target T?
	if targetNullable, ok := target.(NullableType); ok {
		// null -> T?
		if _, isNothing := source.(NothingType); isNothing {
			return true
		}
		// T -> T?
		if source.Equals(targetNullable.InnerType) {
			return true
		}
	}

	return false
}

// NullableSize returns the byte size of a nullable value.
// Reference types: 8 bytes (nullable pointer)
// Primitives: 16 bytes (tagged union: 8-byte tag + 8-byte value)
func NullableSize(inner Type) int {
	if IsReferenceType(inner) {
		return 8 // nullable pointer
	}
	return 16 // tagged union
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
	switch tt := t.(type) {
	case StringType:
		return 8 // pointer size on 64-bit
	case BooleanType:
		return 1 // logically 1 byte, though may be padded in practice
	case VoidType:
		return 0
	case ErrorType:
		return 0
	case NothingType:
		return 0 // null has no size on its own
	case NullableType:
		return NullableSize(tt.InnerType)
	case StructType:
		return tt.Size()
	case ClassType:
		return tt.Size()
	case ArrayType:
		return tt.TotalSize()
	case OwnedPointerType:
		return 8 // pointers are 8 bytes on 64-bit systems
	case RefPointerType, MutRefPointerType:
		return 8 // references are pointers, 8 bytes on 64-bit systems
	default:
		return 8 // default to 8 bytes for unknown types
	}
}

// Common type instances
var (
	// Default types
	TypeInteger = S64Type{} // default integer (s64)
	TypeString  = StringType{}
	TypeBoolean = BooleanType{}
	TypeVoid    = VoidType{}
	TypeError   = ErrorType{}
	TypeNothing = NothingType{} // type of null literal

	// Signed integer types
	TypeS8   = S8Type{}
	TypeS16  = S16Type{}
	TypeS32  = S32Type{}
	TypeS64  = S64Type{}
	TypeS128 = S128Type{}

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

// primitiveTypes is the registry of primitive type names to types
var primitiveTypes = map[string]Type{}

// RegisterPrimitiveType adds a type name mapping to the registry.
// This allows extensions to register custom type aliases.
func RegisterPrimitiveType(name string, t Type) {
	primitiveTypes[name] = t
}

func init() {
	// Register standard types
	RegisterPrimitiveType("int", TypeInteger)
	RegisterPrimitiveType("s8", TypeS8)
	RegisterPrimitiveType("s16", TypeS16)
	RegisterPrimitiveType("s32", TypeS32)
	RegisterPrimitiveType("s64", TypeS64)
	RegisterPrimitiveType("s128", TypeS128)
	RegisterPrimitiveType("u8", TypeU8)
	RegisterPrimitiveType("u16", TypeU16)
	RegisterPrimitiveType("u32", TypeU32)
	RegisterPrimitiveType("u64", TypeU64)
	RegisterPrimitiveType("u128", TypeU128)
	RegisterPrimitiveType("f32", TypeFloat32)
	RegisterPrimitiveType("f64", TypeFloat64)
	RegisterPrimitiveType("string", TypeString)
	RegisterPrimitiveType("bool", TypeBoolean)
	RegisterPrimitiveType("void", TypeVoid)
}

// IsIntegerType checks if a type is any integer type
func IsIntegerType(t Type) bool {
	switch t.(type) {
	case S8Type, S16Type, S32Type, S64Type, S128Type,
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

// TypeFromName converts a type name string to a Type.
// It first looks up the type in the primitive registry.
func TypeFromName(name string) Type {
	if t, ok := primitiveTypes[name]; ok {
		return t
	}
	return TypeError
}

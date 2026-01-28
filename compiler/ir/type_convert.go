package ir

import "github.com/seanrogers2657/slang/compiler/semantic"

// TypeConverter handles conversion from semantic types to IR types.
type TypeConverter struct {
	g *Generator
}

// types returns a TypeConverter for this generator.
func (g *Generator) types() *TypeConverter {
	return &TypeConverter{g: g}
}

// Convert converts a semantic type to an IR type.
func (tc *TypeConverter) Convert(t semantic.Type) Type {
	if t == nil {
		return TypeVoid
	}

	// Check primitive types first (type switch is fast and handles all cases)
	if irType := tc.convertPrimitive(t); irType != nil {
		return irType
	}

	// Handle composite types that need recursion
	return tc.convertComposite(t)
}

// convertPrimitive converts primitive semantic types to IR types.
// Returns nil if t is not a primitive type.
func (tc *TypeConverter) convertPrimitive(t semantic.Type) Type {
	switch t.(type) {
	// Signed integers
	case semantic.S8Type:
		return TypeS8
	case semantic.S16Type:
		return TypeS16
	case semantic.S32Type:
		return TypeS32
	case semantic.S64Type:
		return TypeS64
	case semantic.S128Type:
		return TypeS128

	// Unsigned integers
	case semantic.U8Type:
		return TypeU8
	case semantic.U16Type:
		return TypeU16
	case semantic.U32Type:
		return TypeU32
	case semantic.U64Type:
		return TypeU64
	case semantic.U128Type:
		return TypeU128

	// Other primitives
	case semantic.BooleanType:
		return TypeBool
	case semantic.StringType:
		return TypeString
	case semantic.VoidType:
		return TypeVoid

	default:
		return nil
	}
}

// convertComposite handles types that require recursive conversion.
func (tc *TypeConverter) convertComposite(t semantic.Type) Type {
	// Handle pointer types uniformly (all 3 pointer kinds -> PtrType)
	if elem := pointerElementType(t); elem != nil {
		return &PtrType{Elem: tc.Convert(elem)}
	}

	// Handle struct types (both value and pointer forms)
	if st := asSemanticStructType(t); st != nil {
		return tc.convertStruct(st)
	}

	// Handle class types (both value and pointer forms)
	if ct := asSemanticClassType(t); ct != nil {
		return tc.convertClass(ct)
	}

	// Handle nullable types (both value and pointer forms)
	if nt := asSemanticNullableType(t); nt != nil {
		return &NullableType{Elem: tc.Convert(nt.InnerType)}
	}

	// Handle array (only pointer form exists in practice)
	if at, ok := t.(*semantic.ArrayType); ok {
		return &ArrayType{Elem: tc.Convert(at.ElementType), Len: at.Size}
	}

	// Handle function type
	if ft, ok := t.(*semantic.FunctionType); ok {
		return tc.convertFunc(ft)
	}

	// Unknown type
	return TypeVoid
}

// convertStruct converts a semantic struct type to IR StructType.
func (tc *TypeConverter) convertStruct(st *semantic.StructType) *StructType {
	fields := make([]StructField, len(st.Fields))
	for i, f := range st.Fields {
		fields[i] = StructField{
			Name: f.Name,
			Type: tc.Convert(f.Type),
		}
	}
	irStruct := &StructType{Name: st.Name, Fields: fields}
	irStruct.ComputeLayout()
	return irStruct
}

// convertClass converts a semantic class type to IR StructType.
func (tc *TypeConverter) convertClass(ct *semantic.ClassType) *StructType {
	fields := make([]StructField, len(ct.Fields))
	for i, f := range ct.Fields {
		fields[i] = StructField{
			Name: f.Name,
			Type: tc.Convert(f.Type),
		}
	}
	irStruct := &StructType{Name: ct.Name, Fields: fields}
	irStruct.ComputeLayout()
	return irStruct
}

// convertFunc converts a semantic function type to IR FuncType.
func (tc *TypeConverter) convertFunc(ft *semantic.FunctionType) *FuncType {
	params := make([]Type, len(ft.ParamTypes))
	for i, p := range ft.ParamTypes {
		params[i] = tc.Convert(p)
	}
	return &FuncType{
		Params: params,
		Return: tc.Convert(ft.ReturnType),
	}
}

// =============================================================================
// Type Assertion Helpers for Semantic Types
// =============================================================================

// pointerElementType extracts the element type from any pointer-like semantic type.
// Returns nil if not a pointer type.
func pointerElementType(t semantic.Type) semantic.Type {
	switch ty := t.(type) {
	case *semantic.OwnedPointerType:
		return ty.ElementType
	case semantic.OwnedPointerType:
		return ty.ElementType
	case *semantic.RefPointerType:
		return ty.ElementType
	case semantic.RefPointerType:
		return ty.ElementType
	case *semantic.MutRefPointerType:
		return ty.ElementType
	case semantic.MutRefPointerType:
		return ty.ElementType
	default:
		return nil
	}
}

// asSemanticStructType converts to *semantic.StructType, handling both value and pointer forms.
func asSemanticStructType(t semantic.Type) *semantic.StructType {
	switch ty := t.(type) {
	case *semantic.StructType:
		return ty
	case semantic.StructType:
		return &ty
	default:
		return nil
	}
}

// asSemanticClassType converts to *semantic.ClassType, handling both value and pointer forms.
func asSemanticClassType(t semantic.Type) *semantic.ClassType {
	switch ty := t.(type) {
	case *semantic.ClassType:
		return ty
	case semantic.ClassType:
		return &ty
	default:
		return nil
	}
}

// asSemanticNullableType converts to *semantic.NullableType, handling both value and pointer forms.
func asSemanticNullableType(t semantic.Type) *semantic.NullableType {
	switch ty := t.(type) {
	case *semantic.NullableType:
		return ty
	case semantic.NullableType:
		return &ty
	default:
		return nil
	}
}

// =============================================================================
// Nullable Type Helpers
// =============================================================================

// UnwrapNullableType extracts the inner type from a semantic nullable type.
// Returns the type itself if not nullable.
func UnwrapNullableType(t semantic.Type) semantic.Type {
	if nt := asSemanticNullableType(t); nt != nil {
		return nt.InnerType
	}
	return t
}

// UnwrapIRNullableType extracts the element type from an IR nullable type.
// Returns the type itself if not nullable.
func UnwrapIRNullableType(t Type) Type {
	if nt, ok := t.(*NullableType); ok {
		return nt.Elem
	}
	return t
}

// =============================================================================
// Field Offset and Type Name Helpers
// =============================================================================

// FieldOffset returns the byte offset of a field in a type.
func (tc *TypeConverter) FieldOffset(t semantic.Type, fieldName string) int {
	// Unwrap pointer types first
	if elem := pointerElementType(t); elem != nil {
		return tc.FieldOffset(elem, fieldName)
	}

	// Handle struct types
	if st := asSemanticStructType(t); st != nil {
		irStruct := tc.convertStruct(st)
		if f := irStruct.FieldByName(fieldName); f != nil {
			return f.Offset
		}
	}

	// Handle class types
	if ct := asSemanticClassType(t); ct != nil {
		irStruct := tc.convertClass(ct)
		if f := irStruct.FieldByName(fieldName); f != nil {
			return f.Offset
		}
	}

	return 0
}

// TypeName returns the name of a type for method mangling.
func (tc *TypeConverter) TypeName(t semantic.Type) string {
	// Unwrap pointer types
	if elem := pointerElementType(t); elem != nil {
		return tc.TypeName(elem)
	}

	// Unwrap nullable types
	if nt := asSemanticNullableType(t); nt != nil {
		return tc.TypeName(nt.InnerType)
	}

	// Get name from struct/class/object types
	switch ty := t.(type) {
	case *semantic.StructType:
		return ty.Name
	case semantic.StructType:
		return ty.Name
	case *semantic.ClassType:
		return ty.Name
	case semantic.ClassType:
		return ty.Name
	case *semantic.ObjectType:
		return ty.Name
	case semantic.ObjectType:
		return ty.Name
	default:
		return "unknown"
	}
}

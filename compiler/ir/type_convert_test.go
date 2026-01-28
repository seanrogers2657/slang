package ir

import (
	"testing"

	"github.com/seanrogers2657/slang/compiler/semantic"
)

func TestTypeConverter_ConvertPrimitives(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	tests := []struct {
		name     string
		input    semantic.Type
		expected Type
	}{
		// Signed integers
		{"s8", semantic.S8Type{}, TypeS8},
		{"s16", semantic.S16Type{}, TypeS16},
		{"s32", semantic.S32Type{}, TypeS32},
		{"s64", semantic.S64Type{}, TypeS64},
		{"s128", semantic.S128Type{}, TypeS128},

		// Unsigned integers
		{"u8", semantic.U8Type{}, TypeU8},
		{"u16", semantic.U16Type{}, TypeU16},
		{"u32", semantic.U32Type{}, TypeU32},
		{"u64", semantic.U64Type{}, TypeU64},
		{"u128", semantic.U128Type{}, TypeU128},

		// Other primitives
		{"bool", semantic.BooleanType{}, TypeBool},
		{"string", semantic.StringType{}, TypeString},
		{"void", semantic.VoidType{}, TypeVoid},

		// Nil returns void
		{"nil", nil, TypeVoid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tc.Convert(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("Convert(%T) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTypeConverter_ConvertPointerTypes(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("owned_pointer", func(t *testing.T) {
		input := &semantic.OwnedPointerType{ElementType: semantic.S64Type{}}
		result := tc.Convert(input)
		ptrType, ok := result.(*PtrType)
		if !ok {
			t.Fatalf("expected *PtrType, got %T", result)
		}
		if !ptrType.Elem.Equal(TypeS64) {
			t.Errorf("expected elem type s64, got %v", ptrType.Elem)
		}
	})

	t.Run("owned_pointer_value", func(t *testing.T) {
		input := semantic.OwnedPointerType{ElementType: semantic.S64Type{}}
		result := tc.Convert(input)
		ptrType, ok := result.(*PtrType)
		if !ok {
			t.Fatalf("expected *PtrType, got %T", result)
		}
		if !ptrType.Elem.Equal(TypeS64) {
			t.Errorf("expected elem type s64, got %v", ptrType.Elem)
		}
	})

	t.Run("ref_pointer", func(t *testing.T) {
		input := &semantic.RefPointerType{ElementType: semantic.S32Type{}}
		result := tc.Convert(input)
		ptrType, ok := result.(*PtrType)
		if !ok {
			t.Fatalf("expected *PtrType, got %T", result)
		}
		if !ptrType.Elem.Equal(TypeS32) {
			t.Errorf("expected elem type s32, got %v", ptrType.Elem)
		}
	})

	t.Run("ref_pointer_value", func(t *testing.T) {
		input := semantic.RefPointerType{ElementType: semantic.S32Type{}}
		result := tc.Convert(input)
		ptrType, ok := result.(*PtrType)
		if !ok {
			t.Fatalf("expected *PtrType, got %T", result)
		}
		if !ptrType.Elem.Equal(TypeS32) {
			t.Errorf("expected elem type s32, got %v", ptrType.Elem)
		}
	})

	t.Run("mut_ref_pointer", func(t *testing.T) {
		input := &semantic.MutRefPointerType{ElementType: semantic.BooleanType{}}
		result := tc.Convert(input)
		ptrType, ok := result.(*PtrType)
		if !ok {
			t.Fatalf("expected *PtrType, got %T", result)
		}
		if !ptrType.Elem.Equal(TypeBool) {
			t.Errorf("expected elem type bool, got %v", ptrType.Elem)
		}
	})

	t.Run("mut_ref_pointer_value", func(t *testing.T) {
		input := semantic.MutRefPointerType{ElementType: semantic.BooleanType{}}
		result := tc.Convert(input)
		ptrType, ok := result.(*PtrType)
		if !ok {
			t.Fatalf("expected *PtrType, got %T", result)
		}
		if !ptrType.Elem.Equal(TypeBool) {
			t.Errorf("expected elem type bool, got %v", ptrType.Elem)
		}
	})
}

func TestTypeConverter_ConvertStructType(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("struct_pointer", func(t *testing.T) {
		input := &semantic.StructType{
			Name: "Point",
			Fields: []semantic.StructFieldInfo{
				{Name: "x", Type: semantic.S64Type{}},
				{Name: "y", Type: semantic.S64Type{}},
			},
		}
		result := tc.Convert(input)
		structType, ok := result.(*StructType)
		if !ok {
			t.Fatalf("expected *StructType, got %T", result)
		}
		if structType.Name != "Point" {
			t.Errorf("expected name Point, got %s", structType.Name)
		}
		if len(structType.Fields) != 2 {
			t.Fatalf("expected 2 fields, got %d", len(structType.Fields))
		}
		if structType.Fields[0].Name != "x" {
			t.Errorf("expected field name x, got %s", structType.Fields[0].Name)
		}
	})

	t.Run("struct_value", func(t *testing.T) {
		input := semantic.StructType{
			Name: "Size",
			Fields: []semantic.StructFieldInfo{
				{Name: "width", Type: semantic.S32Type{}},
				{Name: "height", Type: semantic.S32Type{}},
			},
		}
		result := tc.Convert(input)
		structType, ok := result.(*StructType)
		if !ok {
			t.Fatalf("expected *StructType, got %T", result)
		}
		if structType.Name != "Size" {
			t.Errorf("expected name Size, got %s", structType.Name)
		}
	})

	t.Run("struct_layout_computed", func(t *testing.T) {
		input := &semantic.StructType{
			Name: "Mixed",
			Fields: []semantic.StructFieldInfo{
				{Name: "a", Type: semantic.S8Type{}},
				{Name: "b", Type: semantic.S64Type{}},
			},
		}
		result := tc.Convert(input)
		structType, ok := result.(*StructType)
		if !ok {
			t.Fatalf("expected *StructType, got %T", result)
		}
		// Layout should be computed with alignment
		if structType.Size() == 0 {
			t.Error("expected non-zero struct size after layout")
		}
	})
}

func TestTypeConverter_ConvertClassType(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("class_pointer", func(t *testing.T) {
		input := &semantic.ClassType{
			Name: "Counter",
			Fields: []semantic.StructFieldInfo{
				{Name: "count", Type: semantic.S64Type{}},
			},
		}
		result := tc.Convert(input)
		structType, ok := result.(*StructType)
		if !ok {
			t.Fatalf("expected *StructType, got %T", result)
		}
		if structType.Name != "Counter" {
			t.Errorf("expected name Counter, got %s", structType.Name)
		}
	})

	t.Run("class_value", func(t *testing.T) {
		input := semantic.ClassType{
			Name: "Timer",
			Fields: []semantic.StructFieldInfo{
				{Name: "elapsed", Type: semantic.S64Type{}},
			},
		}
		result := tc.Convert(input)
		structType, ok := result.(*StructType)
		if !ok {
			t.Fatalf("expected *StructType, got %T", result)
		}
		if structType.Name != "Timer" {
			t.Errorf("expected name Timer, got %s", structType.Name)
		}
	})
}

func TestTypeConverter_ConvertNullableType(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("nullable_pointer", func(t *testing.T) {
		input := &semantic.NullableType{InnerType: semantic.S64Type{}}
		result := tc.Convert(input)
		nullableType, ok := result.(*NullableType)
		if !ok {
			t.Fatalf("expected *NullableType, got %T", result)
		}
		if !nullableType.Elem.Equal(TypeS64) {
			t.Errorf("expected elem type s64, got %v", nullableType.Elem)
		}
	})

	t.Run("nullable_value", func(t *testing.T) {
		input := semantic.NullableType{InnerType: semantic.BooleanType{}}
		result := tc.Convert(input)
		nullableType, ok := result.(*NullableType)
		if !ok {
			t.Fatalf("expected *NullableType, got %T", result)
		}
		if !nullableType.Elem.Equal(TypeBool) {
			t.Errorf("expected elem type bool, got %v", nullableType.Elem)
		}
	})
}

func TestTypeConverter_ConvertArrayType(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("array_of_ints", func(t *testing.T) {
		input := &semantic.ArrayType{ElementType: semantic.S64Type{}, Size: 10}
		result := tc.Convert(input)
		arrayType, ok := result.(*ArrayType)
		if !ok {
			t.Fatalf("expected *ArrayType, got %T", result)
		}
		if !arrayType.Elem.Equal(TypeS64) {
			t.Errorf("expected elem type s64, got %v", arrayType.Elem)
		}
		if arrayType.Len != 10 {
			t.Errorf("expected len 10, got %d", arrayType.Len)
		}
	})
}

func TestTypeConverter_ConvertFuncType(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("function_with_params_and_return", func(t *testing.T) {
		input := &semantic.FunctionType{
			ParamTypes: []semantic.Type{semantic.S64Type{}, semantic.S64Type{}},
			ReturnType: semantic.S64Type{},
		}
		result := tc.Convert(input)
		funcType, ok := result.(*FuncType)
		if !ok {
			t.Fatalf("expected *FuncType, got %T", result)
		}
		if len(funcType.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(funcType.Params))
		}
		if !funcType.Params[0].Equal(TypeS64) {
			t.Errorf("expected param 0 s64, got %v", funcType.Params[0])
		}
		if !funcType.Return.Equal(TypeS64) {
			t.Errorf("expected return s64, got %v", funcType.Return)
		}
	})

	t.Run("function_void_return", func(t *testing.T) {
		input := &semantic.FunctionType{
			ParamTypes: []semantic.Type{semantic.StringType{}},
			ReturnType: semantic.VoidType{},
		}
		result := tc.Convert(input)
		funcType, ok := result.(*FuncType)
		if !ok {
			t.Fatalf("expected *FuncType, got %T", result)
		}
		if !funcType.Return.Equal(TypeVoid) {
			t.Errorf("expected return void, got %v", funcType.Return)
		}
	})
}

func TestPointerElementType(t *testing.T) {
	t.Run("owned_pointer", func(t *testing.T) {
		pt := &semantic.OwnedPointerType{ElementType: semantic.S64Type{}}
		elem := pointerElementType(pt)
		if elem == nil {
			t.Fatal("expected non-nil element type")
		}
		if _, ok := elem.(semantic.S64Type); !ok {
			t.Errorf("expected S64Type, got %T", elem)
		}
	})

	t.Run("owned_pointer_value", func(t *testing.T) {
		pt := semantic.OwnedPointerType{ElementType: semantic.S32Type{}}
		elem := pointerElementType(pt)
		if elem == nil {
			t.Fatal("expected non-nil element type")
		}
	})

	t.Run("ref_pointer", func(t *testing.T) {
		pt := &semantic.RefPointerType{ElementType: semantic.BooleanType{}}
		elem := pointerElementType(pt)
		if elem == nil {
			t.Fatal("expected non-nil element type")
		}
	})

	t.Run("ref_pointer_value", func(t *testing.T) {
		pt := semantic.RefPointerType{ElementType: semantic.BooleanType{}}
		elem := pointerElementType(pt)
		if elem == nil {
			t.Fatal("expected non-nil element type")
		}
	})

	t.Run("mut_ref_pointer", func(t *testing.T) {
		pt := &semantic.MutRefPointerType{ElementType: semantic.StringType{}}
		elem := pointerElementType(pt)
		if elem == nil {
			t.Fatal("expected non-nil element type")
		}
	})

	t.Run("mut_ref_pointer_value", func(t *testing.T) {
		pt := semantic.MutRefPointerType{ElementType: semantic.StringType{}}
		elem := pointerElementType(pt)
		if elem == nil {
			t.Fatal("expected non-nil element type")
		}
	})

	t.Run("non_pointer_returns_nil", func(t *testing.T) {
		elem := pointerElementType(semantic.S64Type{})
		if elem != nil {
			t.Errorf("expected nil for non-pointer, got %T", elem)
		}
	})
}

func TestAsSemanticStructType(t *testing.T) {
	t.Run("pointer_struct", func(t *testing.T) {
		st := &semantic.StructType{Name: "Point"}
		result := asSemanticStructType(st)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Name != "Point" {
			t.Errorf("expected name Point, got %s", result.Name)
		}
	})

	t.Run("value_struct", func(t *testing.T) {
		st := semantic.StructType{Name: "Size"}
		result := asSemanticStructType(st)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Name != "Size" {
			t.Errorf("expected name Size, got %s", result.Name)
		}
	})

	t.Run("non_struct_returns_nil", func(t *testing.T) {
		result := asSemanticStructType(semantic.S64Type{})
		if result != nil {
			t.Errorf("expected nil for non-struct, got %v", result)
		}
	})
}

func TestAsSemanticClassType(t *testing.T) {
	t.Run("pointer_class", func(t *testing.T) {
		ct := &semantic.ClassType{Name: "Counter"}
		result := asSemanticClassType(ct)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Name != "Counter" {
			t.Errorf("expected name Counter, got %s", result.Name)
		}
	})

	t.Run("value_class", func(t *testing.T) {
		ct := semantic.ClassType{Name: "Timer"}
		result := asSemanticClassType(ct)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Name != "Timer" {
			t.Errorf("expected name Timer, got %s", result.Name)
		}
	})

	t.Run("non_class_returns_nil", func(t *testing.T) {
		result := asSemanticClassType(semantic.S64Type{})
		if result != nil {
			t.Errorf("expected nil for non-class, got %v", result)
		}
	})
}

func TestAsSemanticNullableType(t *testing.T) {
	t.Run("pointer_nullable", func(t *testing.T) {
		nt := &semantic.NullableType{InnerType: semantic.S64Type{}}
		result := asSemanticNullableType(nt)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("value_nullable", func(t *testing.T) {
		nt := semantic.NullableType{InnerType: semantic.BooleanType{}}
		result := asSemanticNullableType(nt)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("non_nullable_returns_nil", func(t *testing.T) {
		result := asSemanticNullableType(semantic.S64Type{})
		if result != nil {
			t.Errorf("expected nil for non-nullable, got %v", result)
		}
	})
}

func TestTypeConverter_FieldOffset(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("struct_field_offset", func(t *testing.T) {
		st := &semantic.StructType{
			Name: "Point",
			Fields: []semantic.StructFieldInfo{
				{Name: "x", Type: semantic.S64Type{}},
				{Name: "y", Type: semantic.S64Type{}},
			},
		}
		xOffset := tc.FieldOffset(st, "x")
		yOffset := tc.FieldOffset(st, "y")
		if xOffset != 0 {
			t.Errorf("expected x offset 0, got %d", xOffset)
		}
		if yOffset != 8 {
			t.Errorf("expected y offset 8, got %d", yOffset)
		}
	})

	t.Run("class_field_offset", func(t *testing.T) {
		ct := &semantic.ClassType{
			Name: "Counter",
			Fields: []semantic.StructFieldInfo{
				{Name: "value", Type: semantic.S64Type{}},
				{Name: "step", Type: semantic.S32Type{}},
			},
		}
		valueOffset := tc.FieldOffset(ct, "value")
		stepOffset := tc.FieldOffset(ct, "step")
		if valueOffset != 0 {
			t.Errorf("expected value offset 0, got %d", valueOffset)
		}
		if stepOffset != 8 {
			t.Errorf("expected step offset 8, got %d", stepOffset)
		}
	})

	t.Run("pointer_unwrapping", func(t *testing.T) {
		st := &semantic.StructType{
			Name: "Point",
			Fields: []semantic.StructFieldInfo{
				{Name: "x", Type: semantic.S64Type{}},
			},
		}
		pt := &semantic.OwnedPointerType{ElementType: st}
		offset := tc.FieldOffset(pt, "x")
		if offset != 0 {
			t.Errorf("expected x offset 0, got %d", offset)
		}
	})

	t.Run("unknown_field_returns_zero", func(t *testing.T) {
		st := &semantic.StructType{
			Name:   "Point",
			Fields: []semantic.StructFieldInfo{{Name: "x", Type: semantic.S64Type{}}},
		}
		offset := tc.FieldOffset(st, "nonexistent")
		if offset != 0 {
			t.Errorf("expected 0 for unknown field, got %d", offset)
		}
	})

	t.Run("non_struct_returns_zero", func(t *testing.T) {
		offset := tc.FieldOffset(semantic.S64Type{}, "x")
		if offset != 0 {
			t.Errorf("expected 0 for non-struct, got %d", offset)
		}
	})
}

// ============================================================================
// Unwrap Helpers Tests
// ============================================================================

func TestUnwrapNullableType(t *testing.T) {
	t.Run("unwraps nullable pointer", func(t *testing.T) {
		inner := semantic.S64Type{}
		nt := &semantic.NullableType{InnerType: inner}
		result := UnwrapNullableType(nt)
		if _, ok := result.(semantic.S64Type); !ok {
			t.Errorf("expected S64Type, got %T", result)
		}
	})

	t.Run("unwraps nullable value", func(t *testing.T) {
		inner := semantic.BooleanType{}
		nt := semantic.NullableType{InnerType: inner}
		result := UnwrapNullableType(nt)
		if _, ok := result.(semantic.BooleanType); !ok {
			t.Errorf("expected BooleanType, got %T", result)
		}
	})

	t.Run("returns non-nullable unchanged", func(t *testing.T) {
		input := semantic.S64Type{}
		result := UnwrapNullableType(input)
		if result != input {
			t.Errorf("expected same type, got %T", result)
		}
	})

	t.Run("returns struct type unchanged", func(t *testing.T) {
		input := &semantic.StructType{Name: "Point"}
		result := UnwrapNullableType(input)
		if result != input {
			t.Errorf("expected same struct type, got %T", result)
		}
	})
}

func TestUnwrapIRNullableType(t *testing.T) {
	t.Run("unwraps nullable type", func(t *testing.T) {
		inner := TypeS64
		nt := &NullableType{Elem: inner}
		result := UnwrapIRNullableType(nt)
		if result != inner {
			t.Errorf("expected TypeS64, got %v", result)
		}
	})

	t.Run("returns non-nullable unchanged", func(t *testing.T) {
		result := UnwrapIRNullableType(TypeS64)
		if result != TypeS64 {
			t.Errorf("expected TypeS64, got %v", result)
		}
	})

	t.Run("returns struct type unchanged", func(t *testing.T) {
		st := &StructType{Name: "Point"}
		result := UnwrapIRNullableType(st)
		if result != st {
			t.Errorf("expected same struct type, got %v", result)
		}
	})

	t.Run("returns ptr type unchanged", func(t *testing.T) {
		pt := &PtrType{Elem: TypeS64}
		result := UnwrapIRNullableType(pt)
		if result != pt {
			t.Errorf("expected same ptr type, got %v", result)
		}
	})

	t.Run("unwraps nested types correctly", func(t *testing.T) {
		// Nullable containing struct
		st := &StructType{Name: "Point"}
		nt := &NullableType{Elem: st}
		result := UnwrapIRNullableType(nt)
		if result != st {
			t.Errorf("expected struct type, got %v", result)
		}
	})
}

// ============================================================================
// TypeConverter TypeName Tests
// ============================================================================

func TestTypeConverter_TypeName(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	tc := g.types()

	t.Run("struct_name", func(t *testing.T) {
		st := &semantic.StructType{Name: "Point"}
		name := tc.TypeName(st)
		if name != "Point" {
			t.Errorf("expected Point, got %s", name)
		}
	})

	t.Run("struct_value_name", func(t *testing.T) {
		st := semantic.StructType{Name: "Size"}
		name := tc.TypeName(st)
		if name != "Size" {
			t.Errorf("expected Size, got %s", name)
		}
	})

	t.Run("class_name", func(t *testing.T) {
		ct := &semantic.ClassType{Name: "Counter"}
		name := tc.TypeName(ct)
		if name != "Counter" {
			t.Errorf("expected Counter, got %s", name)
		}
	})

	t.Run("class_value_name", func(t *testing.T) {
		ct := semantic.ClassType{Name: "Timer"}
		name := tc.TypeName(ct)
		if name != "Timer" {
			t.Errorf("expected Timer, got %s", name)
		}
	})

	t.Run("object_name", func(t *testing.T) {
		ot := &semantic.ObjectType{Name: "MyObject"}
		name := tc.TypeName(ot)
		if name != "MyObject" {
			t.Errorf("expected MyObject, got %s", name)
		}
	})

	t.Run("object_value_name", func(t *testing.T) {
		ot := semantic.ObjectType{Name: "YourObject"}
		name := tc.TypeName(ot)
		if name != "YourObject" {
			t.Errorf("expected YourObject, got %s", name)
		}
	})

	t.Run("pointer_unwrapping", func(t *testing.T) {
		st := &semantic.StructType{Name: "Point"}
		pt := &semantic.OwnedPointerType{ElementType: st}
		name := tc.TypeName(pt)
		if name != "Point" {
			t.Errorf("expected Point, got %s", name)
		}
	})

	t.Run("nullable_unwrapping", func(t *testing.T) {
		st := &semantic.StructType{Name: "Point"}
		nt := &semantic.NullableType{InnerType: st}
		name := tc.TypeName(nt)
		if name != "Point" {
			t.Errorf("expected Point, got %s", name)
		}
	})

	t.Run("unknown_type", func(t *testing.T) {
		name := tc.TypeName(semantic.S64Type{})
		if name != "unknown" {
			t.Errorf("expected unknown, got %s", name)
		}
	})
}

package semantic

import (
	"testing"
)

func TestTypeFromName(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantType Type
	}{
		// Integer types
		{"int", "int", TypeInteger},
		{"s8", "s8", TypeS8},
		{"s16", "s16", TypeS16},
		{"s32", "s32", TypeS32},
		{"s64", "s64", TypeS64},
		{"s128", "s128", TypeS128},
		{"u8", "u8", TypeU8},
		{"u16", "u16", TypeU16},
		{"u32", "u32", TypeU32},
		{"u64", "u64", TypeU64},
		{"u128", "u128", TypeU128},

		// Float types
		{"f32", "f32", TypeFloat32},
		{"f64", "f64", TypeFloat64},

		// Other primitives
		{"string", "string", TypeString},
		{"bool", "bool", TypeBoolean},
		{"void", "void", TypeVoid},

		// Unknown type should return TypeError
		{"unknown", "unknownType", TypeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TypeFromName(tt.typeName)
			if got != tt.wantType {
				t.Errorf("TypeFromName(%q) = %v, want %v", tt.typeName, got, tt.wantType)
			}
		})
	}
}

func TestRegisterPrimitiveType(t *testing.T) {
	// Use an existing type that's not already registered with a different name
	// Register TypeS64 under a custom name
	RegisterPrimitiveType("myCustomInt", TypeS64)

	// Verify it was registered
	got := TypeFromName("myCustomInt")
	if got != TypeS64 {
		t.Errorf("expected TypeS64, got %v", got)
	}

	// Cleanup - remove the custom registration
	delete(primitiveTypes, "myCustomInt")

	// Verify cleanup worked
	got = TypeFromName("myCustomInt")
	if got != TypeError {
		t.Errorf("expected TypeError after cleanup, got %v", got)
	}
}

func TestPrimitiveTypesInit(t *testing.T) {
	// Verify all expected types are registered during init
	expectedTypes := map[string]Type{
		"int":    TypeInteger,
		"s8":     TypeS8,
		"s16":    TypeS16,
		"s32":    TypeS32,
		"s64":    TypeS64,
		"s128":   TypeS128,
		"u8":     TypeU8,
		"u16":    TypeU16,
		"u32":    TypeU32,
		"u64":    TypeU64,
		"u128":   TypeU128,
		"f32":    TypeFloat32,
		"f64":    TypeFloat64,
		"string": TypeString,
		"bool":   TypeBoolean,
		"void":   TypeVoid,
	}

	for name, expectedType := range expectedTypes {
		if got, ok := primitiveTypes[name]; !ok {
			t.Errorf("expected %q to be registered in primitiveTypes", name)
		} else if got != expectedType {
			t.Errorf("primitiveTypes[%q] = %v, want %v", name, got, expectedType)
		}
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		typ  Type
		want string
	}{
		{TypeInteger, "s64"},
		{TypeS8, "s8"},
		{TypeS16, "s16"},
		{TypeS32, "s32"},
		{TypeS64, "s64"},
		{TypeS128, "s128"},
		{TypeU8, "u8"},
		{TypeU16, "u16"},
		{TypeU32, "u32"},
		{TypeU64, "u64"},
		{TypeU128, "u128"},
		{TypeFloat32, "f32"},
		{TypeFloat64, "f64"},
		{TypeString, "string"},
		{TypeBoolean, "boolean"},
		{TypeVoid, "void"},
		{TypeError, "<error>"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.typ.String()
			if got != tt.want {
				t.Errorf("Type.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestArrayType(t *testing.T) {
	t.Run("string representation", func(t *testing.T) {
		arrType := ArrayType{ElementType: TypeS64, Size: 5}
		want := "s64[]"
		if got := arrType.String(); got != want {
			t.Errorf("ArrayType.String() = %q, want %q", got, want)
		}
	})

	t.Run("unknown size representation", func(t *testing.T) {
		arrType := ArrayType{ElementType: TypeS32, Size: ArraySizeUnknown}
		want := "s32[]"
		if got := arrType.String(); got != want {
			t.Errorf("ArrayType.String() = %q, want %q", got, want)
		}
	})
}

func TestNullableType(t *testing.T) {
	t.Run("string representation", func(t *testing.T) {
		nullableType := NullableType{InnerType: TypeS64}
		want := "s64?"
		if got := nullableType.String(); got != want {
			t.Errorf("NullableType.String() = %q, want %q", got, want)
		}
	})

	t.Run("nested types", func(t *testing.T) {
		nullableType := NullableType{InnerType: TypeString}
		want := "string?"
		if got := nullableType.String(); got != want {
			t.Errorf("NullableType.String() = %q, want %q", got, want)
		}
	})
}

func TestPointerTypes(t *testing.T) {
	t.Run("owned pointer", func(t *testing.T) {
		ptrType := OwnedPointerType{ElementType: TypeS64}
		want := "*s64"
		if got := ptrType.String(); got != want {
			t.Errorf("OwnedPointerType.String() = %q, want %q", got, want)
		}
	})

	t.Run("ref pointer (immutable borrow)", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeString}
		want := "&string"
		if got := refType.String(); got != want {
			t.Errorf("RefPointerType.String() = %q, want %q", got, want)
		}
	})

	t.Run("mut ref pointer (mutable borrow)", func(t *testing.T) {
		mutRefType := MutRefPointerType{ElementType: TypeBoolean}
		want := "&&boolean"
		if got := mutRefType.String(); got != want {
			t.Errorf("MutRefPointerType.String() = %q, want %q", got, want)
		}
	})
}

func TestIntegerWidensTo(t *testing.T) {
	tests := []struct {
		name string
		from Type
		to   Type
		want bool
	}{
		// Same signedness: narrower (or equal) widens.
		{"s8 to s64", TypeS8, TypeS64, true},
		{"s64 to s64", TypeS64, TypeS64, true},
		{"u8 to u64", TypeU8, TypeU64, true},
		{"s64 to s8", TypeS64, TypeS8, false},
		{"u64 to u8", TypeU64, TypeU8, false},

		// Unsigned to signed: allowed only when strictly narrower, so the
		// whole unsigned range fits below the signed limit.
		{"u8 to s64", TypeU8, TypeS64, true},
		{"u16 to s64", TypeU16, TypeS64, true},
		{"u32 to s64", TypeU32, TypeS64, true},
		{"u8 to s16", TypeU8, TypeS16, true},
		{"u64 to s64", TypeU64, TypeS64, false}, // high half would read negative
		{"u8 to s8", TypeU8, TypeS8, false},     // 255 does not fit in s8
		{"u32 to s32", TypeU32, TypeS32, false},

		// Signed to unsigned: never, since negatives have no representation.
		{"s8 to u64", TypeS8, TypeU64, false},
		{"s64 to u64", TypeS64, TypeU64, false},

		// Non-integers never widen.
		{"string to s64", TypeString, TypeS64, false},
		{"s64 to string", TypeS64, TypeString, false},
		{"bool to s64", TypeBoolean, TypeS64, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntegerWidensTo(tt.from, tt.to); got != tt.want {
				t.Errorf("IntegerWidensTo(%s, %s) = %v, want %v",
					tt.from.String(), tt.to.String(), got, tt.want)
			}
		})
	}
}

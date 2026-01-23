package semantic

import (
	"reflect"
	"testing"
)

func TestCheckIntegerBoundsCore(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		targetType Type
		wantErr    bool
		errContain string
	}{
		// S8 bounds: -128 to 127
		{"s8 min valid", "-128", TypeS8, false, ""},
		{"s8 max valid", "127", TypeS8, false, ""},
		{"s8 zero", "0", TypeS8, false, ""},
		{"s8 overflow positive", "128", TypeS8, true, "out of range for s8"},
		{"s8 overflow negative", "-129", TypeS8, true, "out of range for s8"},

		// S16 bounds: -32768 to 32767
		{"s16 min valid", "-32768", TypeS16, false, ""},
		{"s16 max valid", "32767", TypeS16, false, ""},
		{"s16 overflow positive", "32768", TypeS16, true, "out of range for s16"},
		{"s16 overflow negative", "-32769", TypeS16, true, "out of range for s16"},

		// S32 bounds: -2147483648 to 2147483647
		{"s32 min valid", "-2147483648", TypeS32, false, ""},
		{"s32 max valid", "2147483647", TypeS32, false, ""},
		{"s32 overflow positive", "2147483648", TypeS32, true, "out of range for s32"},

		// S64 bounds
		{"s64 min valid", "-9223372036854775808", TypeS64, false, ""},
		{"s64 max valid", "9223372036854775807", TypeS64, false, ""},
		{"s64 overflow positive", "9223372036854775808", TypeS64, true, "out of range for s64"},

		// U8 bounds: 0 to 255
		{"u8 min valid", "0", TypeU8, false, ""},
		{"u8 max valid", "255", TypeU8, false, ""},
		{"u8 overflow positive", "256", TypeU8, true, "out of range for u8"},
		{"u8 negative", "-1", TypeU8, true, "out of range for u8"},

		// U16 bounds: 0 to 65535
		{"u16 min valid", "0", TypeU16, false, ""},
		{"u16 max valid", "65535", TypeU16, false, ""},
		{"u16 overflow positive", "65536", TypeU16, true, "out of range for u16"},

		// U32 bounds: 0 to 4294967295
		{"u32 min valid", "0", TypeU32, false, ""},
		{"u32 max valid", "4294967295", TypeU32, false, ""},
		{"u32 overflow positive", "4294967296", TypeU32, true, "out of range for u32"},

		// U64 bounds
		{"u64 min valid", "0", TypeU64, false, ""},
		{"u64 max valid", "18446744073709551615", TypeU64, false, ""},
		{"u64 overflow positive", "18446744073709551616", TypeU64, true, "out of range for u64"},

		// Invalid literal
		{"invalid literal", "abc", TypeS8, true, "invalid integer literal"},

		// Unknown type (should return empty string, no error)
		{"unknown type", "42", TypeString, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := checkIntegerBoundsCore(tt.value, tt.targetType)
			if tt.wantErr {
				if errMsg == "" {
					t.Errorf("expected error containing %q, got no error", tt.errContain)
				} else if tt.errContain != "" && !contains(errMsg, tt.errContain) {
					t.Errorf("expected error containing %q, got %q", tt.errContain, errMsg)
				}
			} else {
				if errMsg != "" {
					t.Errorf("expected no error, got %q", errMsg)
				}
			}
		})
	}
}

func TestGetIntegerBounds(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		wantOK   bool
		wantName string
	}{
		{"s8", TypeS8, true, "s8"},
		{"s16", TypeS16, true, "s16"},
		{"s32", TypeS32, true, "s32"},
		{"s64", TypeS64, true, "s64"},
		{"s128", TypeS128, true, "s128"},
		{"u8", TypeU8, true, "u8"},
		{"u16", TypeU16, true, "u16"},
		{"u32", TypeU32, true, "u32"},
		{"u64", TypeU64, true, "u64"},
		{"u128", TypeU128, true, "u128"},
		{"string (not integer)", TypeString, false, ""},
		{"bool (not integer)", TypeBoolean, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounds, ok := GetIntegerBounds(tt.typ)
			if ok != tt.wantOK {
				t.Errorf("GetIntegerBounds() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && bounds.TypeName != tt.wantName {
				t.Errorf("GetIntegerBounds() TypeName = %v, want %v", bounds.TypeName, tt.wantName)
			}
		})
	}
}

func TestRegisterIntegerBounds(t *testing.T) {
	// Create a custom type for testing
	customBounds := IntBounds{
		Min:      minS8,
		Max:      maxS8,
		TypeName: "custom_int",
	}

	// Register bounds for a type that doesn't normally have bounds
	RegisterIntegerBounds(TypeVoid, customBounds)

	// Verify it was registered
	bounds, ok := GetIntegerBounds(TypeVoid)
	if !ok {
		t.Fatal("expected bounds to be registered for TypeVoid")
	}
	if bounds.TypeName != "custom_int" {
		t.Errorf("expected TypeName 'custom_int', got %q", bounds.TypeName)
	}

	// Clean up - remove the custom registration
	// We need to access the internal map key type directly
	delete(integerBounds, reflect.TypeOf(VoidType{}))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

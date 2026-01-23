package semantic

import (
	"fmt"
	"math/big"
	"reflect"
)

// IntBounds defines the min/max bounds for an integer type
type IntBounds struct {
	Min      *big.Int
	Max      *big.Int
	TypeName string // for error messages (e.g., "s8", "u16")
}

// integerBounds maps type to its bounds
var integerBounds map[reflect.Type]IntBounds

// Pre-computed bounds constants
var (
	minS8, _   = new(big.Int).SetString("-128", 10)
	maxS8, _   = new(big.Int).SetString("127", 10)
	minS16, _  = new(big.Int).SetString("-32768", 10)
	maxS16, _  = new(big.Int).SetString("32767", 10)
	minS32, _  = new(big.Int).SetString("-2147483648", 10)
	maxS32, _  = new(big.Int).SetString("2147483647", 10)
	minS64, _  = new(big.Int).SetString("-9223372036854775808", 10)
	maxS64, _  = new(big.Int).SetString("9223372036854775807", 10)
	minS128, _ = new(big.Int).SetString("-170141183460469231731687303715884105728", 10)
	maxS128, _ = new(big.Int).SetString("170141183460469231731687303715884105727", 10)

	maxU8, _   = new(big.Int).SetString("255", 10)
	maxU16, _  = new(big.Int).SetString("65535", 10)
	maxU32, _  = new(big.Int).SetString("4294967295", 10)
	maxU64, _  = new(big.Int).SetString("18446744073709551615", 10)
	maxU128, _ = new(big.Int).SetString("340282366920938463463374607431768211455", 10)

	zero = big.NewInt(0)
)

func init() {
	integerBounds = map[reflect.Type]IntBounds{
		reflect.TypeOf(S8Type{}):   {minS8, maxS8, "s8"},
		reflect.TypeOf(S16Type{}):  {minS16, maxS16, "s16"},
		reflect.TypeOf(S32Type{}):  {minS32, maxS32, "s32"},
		reflect.TypeOf(S64Type{}):  {minS64, maxS64, "s64"},
		reflect.TypeOf(S128Type{}): {minS128, maxS128, "s128"},
		reflect.TypeOf(U8Type{}):   {zero, maxU8, "u8"},
		reflect.TypeOf(U16Type{}):  {zero, maxU16, "u16"},
		reflect.TypeOf(U32Type{}):  {zero, maxU32, "u32"},
		reflect.TypeOf(U64Type{}):  {zero, maxU64, "u64"},
		reflect.TypeOf(U128Type{}): {zero, maxU128, "u128"},
	}
}

// RegisterIntegerBounds registers bounds for a custom integer type.
// This allows extensions to add new integer types with bounds checking.
func RegisterIntegerBounds(t Type, bounds IntBounds) {
	integerBounds[reflect.TypeOf(t)] = bounds
}

// GetIntegerBounds returns the bounds for an integer type, if registered.
func GetIntegerBounds(t Type) (IntBounds, bool) {
	bounds, ok := integerBounds[reflect.TypeOf(t)]
	return bounds, ok
}

// checkIntegerBoundsCore checks if an integer literal fits in the declared type.
// Returns an error message if out of bounds, or empty string if valid.
func checkIntegerBoundsCore(value string, targetType Type) string {
	bounds, ok := integerBounds[reflect.TypeOf(targetType)]
	if !ok {
		return "" // unknown type, skip check
	}

	val, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return fmt.Sprintf("invalid integer literal: %s", value)
	}

	if val.Cmp(bounds.Min) < 0 || val.Cmp(bounds.Max) > 0 {
		return fmt.Sprintf("integer literal %s out of range for %s (%s to %s)",
			value, bounds.TypeName, bounds.Min, bounds.Max)
	}
	return ""
}


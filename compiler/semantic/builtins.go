package semantic

// BuiltinFunc defines a built-in function's signature
type BuiltinFunc struct {
	ParamTypes []Type
	ReturnType Type
	NoReturn   bool // true for functions like exit that never return
	// AcceptedTypes allows a parameter to accept multiple types (e.g., print accepts i64 or string)
	// Key is parameter index, value is slice of accepted types
	AcceptedTypes map[int][]Type
	// IsArrayLen indicates this is the special len() function for arrays
	IsArrayLen bool
}

// Builtins is the registry of all built-in functions
var Builtins = map[string]BuiltinFunc{
	"exit": {
		ParamTypes: []Type{TypeI64},
		ReturnType: TypeVoid,
		NoReturn:   true,
	},
	"print": {
		ParamTypes: []Type{TypeI64}, // default type for error messages
		ReturnType: TypeVoid,
		NoReturn:   false,
		AcceptedTypes: map[int][]Type{
			0: {TypeI64, TypeString, TypeBoolean}, // print accepts i64, string, or bool
		},
	},
	"len": {
		ParamTypes: []Type{TypeError}, // special: accepts any array type
		ReturnType: TypeI64,
		NoReturn:   false,
		IsArrayLen: true,
	},
}

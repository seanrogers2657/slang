package semantic

// BuiltinFunc defines a built-in function's signature
type BuiltinFunc struct {
	ParamTypes []Type
	ReturnType Type
	NoReturn   bool // true for functions like exit that never return
	// AcceptedTypes allows a parameter to accept multiple types (e.g., print accepts s64 or string)
	// Key is parameter index, value is slice of accepted types
	AcceptedTypes map[int][]Type
	// IsArrayLen indicates this is the special len() function for arrays
	IsArrayLen bool
}

// Builtins is the registry of all built-in functions
var Builtins = map[string]BuiltinFunc{
	"exit": {
		ParamTypes: []Type{TypeS64},
		ReturnType: TypeVoid,
		NoReturn:   true,
	},
	"print": {
		ParamTypes: []Type{TypeS64}, // default type for error messages
		ReturnType: TypeVoid,
		NoReturn:   false,
		AcceptedTypes: map[int][]Type{
			0: {TypeS64, TypeString, TypeBoolean}, // print accepts s64, string, or bool
		},
	},
	"len": {
		ParamTypes: []Type{TypeError}, // special: accepts any array type
		ReturnType: TypeS64,
		NoReturn:   false,
		IsArrayLen: true,
	},
	"sleep": {
		ParamTypes: []Type{TypeS64}, // nanoseconds to sleep
		ReturnType: TypeVoid,
		NoReturn:   false,
	},
}

package semantic

// BuiltinFunc defines a built-in function's signature
type BuiltinFunc struct {
	ParamTypes []Type
	ReturnType Type
	NoReturn   bool // true for functions like exit that never return
	// AcceptedTypes allows a parameter to accept multiple types (e.g., print accepts i64 or string)
	// Key is parameter index, value is slice of accepted types
	AcceptedTypes map[int][]Type
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
			0: {TypeI64, TypeString}, // print accepts i64 or string
		},
	},
}

package semantic

// BuiltinFunc defines a built-in function's signature
type BuiltinFunc struct {
	ParamTypes []Type
	ReturnType Type
	NoReturn   bool // true for functions like exit that never return
}

// Builtins is the registry of all built-in functions
var Builtins = map[string]BuiltinFunc{
	"exit": {
		ParamTypes: []Type{TypeI64},
		ReturnType: TypeVoid,
		NoReturn:   true,
	},
	"print": {
		ParamTypes: []Type{TypeI64},
		ReturnType: TypeVoid,
		NoReturn:   false,
	},
}

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

// BuiltinRegistry manages built-in functions with support for runtime registration.
type BuiltinRegistry struct {
	funcs map[string]BuiltinFunc
}

// NewBuiltinRegistry creates a new registry with default builtins registered.
func NewBuiltinRegistry() *BuiltinRegistry {
	r := &BuiltinRegistry{
		funcs: make(map[string]BuiltinFunc),
	}
	r.registerDefaults()
	return r
}

// Register adds a builtin function to the registry.
func (r *BuiltinRegistry) Register(name string, fn BuiltinFunc) {
	r.funcs[name] = fn
}

// Lookup finds a builtin by name.
// Returns the builtin and true if found, or empty BuiltinFunc and false if not found.
func (r *BuiltinRegistry) Lookup(name string) (BuiltinFunc, bool) {
	fn, ok := r.funcs[name]
	return fn, ok
}

// All returns all registered builtins as a map.
func (r *BuiltinRegistry) All() map[string]BuiltinFunc {
	return r.funcs
}

// registerDefaults registers the standard built-in functions.
func (r *BuiltinRegistry) registerDefaults() {
	r.Register("exit", BuiltinFunc{
		ParamTypes: []Type{TypeS64},
		ReturnType: TypeVoid,
		NoReturn:   true,
	})
	r.Register("print", BuiltinFunc{
		ParamTypes: []Type{TypeS64}, // default type for error messages
		ReturnType: TypeVoid,
		NoReturn:   false,
		AcceptedTypes: map[int][]Type{
			0: {TypeS64, TypeString, TypeBoolean}, // print accepts s64, string, or bool
		},
	})
	r.Register("len", BuiltinFunc{
		ParamTypes: []Type{TypeError}, // special: accepts any array type
		ReturnType: TypeS64,
		NoReturn:   false,
		IsArrayLen: true,
	})
	r.Register("sleep", BuiltinFunc{
		ParamTypes: []Type{TypeS64}, // nanoseconds to sleep
		ReturnType: TypeVoid,
		NoReturn:   false,
	})
}

// defaultBuiltinRegistry is the shared default registry
var defaultBuiltinRegistry = NewBuiltinRegistry()

// Builtins is the registry of all built-in functions.
// Maintained for backward compatibility - use BuiltinRegistry for new code.
var Builtins = defaultBuiltinRegistry.funcs

// RegisterBuiltin adds a builtin function to the default registry.
func RegisterBuiltin(name string, fn BuiltinFunc) {
	defaultBuiltinRegistry.Register(name, fn)
}

// LookupBuiltin finds a builtin by name in the default registry.
func LookupBuiltin(name string) (BuiltinFunc, bool) {
	return defaultBuiltinRegistry.Lookup(name)
}

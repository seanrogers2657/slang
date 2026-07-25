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
	r.Register("assert", BuiltinFunc{
		ParamTypes: []Type{TypeBoolean, TypeString}, // condition, message
		ReturnType: TypeVoid,
		NoReturn:   false,
	})
	// String built-ins. substr slices the half-open byte range [start, end)
	// into a fresh string (range-checked at runtime); chr builds a one-byte
	// string from a byte, the inverse of the s[i] index. Together with len(s)
	// and s[i] these are enough to write split/trim/parse in Slang itself.
	r.Register("substr", BuiltinFunc{
		ParamTypes: []Type{TypeString, TypeS64, TypeS64}, // string, start, end
		ReturnType: TypeString,
	})
	r.Register("chr", BuiltinFunc{
		ParamTypes: []Type{TypeU8}, // byte value
		ReturnType: TypeString,
		// s[i] yields u8, but byte arithmetic naturally widens to s64, so
		// accept any integer the caller already has in hand.
		AcceptedTypes: map[int][]Type{
			0: {TypeU8, TypeU16, TypeU32, TypeU64, TypeS8, TypeS16, TypeS32, TypeS64},
		},
	})
	// Growable vector (vec) built-ins. vec() makes an empty vec; push/get/set
	// operate on it (get/set are bounds-checked at runtime). len() also accepts a
	// vec (handled via IsArrayLen).
	r.Register("vec", BuiltinFunc{
		ParamTypes: []Type{},
		ReturnType: TypeVec,
	})
	r.Register("push", BuiltinFunc{
		ParamTypes: []Type{TypeVec, TypeS64}, // vec, value
		ReturnType: TypeVoid,
	})
	r.Register("get", BuiltinFunc{
		ParamTypes: []Type{TypeVec, TypeS64}, // vec, index
		ReturnType: TypeS64,
	})
	r.Register("set", BuiltinFunc{
		ParamTypes: []Type{TypeVec, TypeS64, TypeS64}, // vec, index, value
		ReturnType: TypeVoid,
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

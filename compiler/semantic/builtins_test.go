package semantic

import (
	"testing"
)

func TestNewBuiltinRegistry(t *testing.T) {
	registry := NewBuiltinRegistry()

	if registry == nil {
		t.Fatal("NewBuiltinRegistry() returned nil")
	}
	if registry.funcs == nil {
		t.Error("funcs map is nil")
	}

	// Verify default builtins are registered
	defaults := []string{"exit", "print", "len", "sleep"}
	for _, name := range defaults {
		if _, ok := registry.Lookup(name); !ok {
			t.Errorf("expected default builtin %q to be registered", name)
		}
	}
}

func TestBuiltinRegistry_Register(t *testing.T) {
	registry := NewBuiltinRegistry()

	// Register a new builtin
	customFn := BuiltinFunc{
		ParamTypes: []Type{TypeString, TypeS64},
		ReturnType: TypeBoolean,
	}
	registry.Register("customFunc", customFn)

	// Verify it was registered
	got, ok := registry.Lookup("customFunc")
	if !ok {
		t.Fatal("customFunc not found after registration")
	}
	if len(got.ParamTypes) != 2 {
		t.Errorf("expected 2 param types, got %d", len(got.ParamTypes))
	}
	if got.ReturnType != TypeBoolean {
		t.Errorf("expected return type Boolean, got %v", got.ReturnType)
	}
}

func TestBuiltinRegistry_Lookup(t *testing.T) {
	registry := NewBuiltinRegistry()

	t.Run("lookup existing", func(t *testing.T) {
		fn, ok := registry.Lookup("exit")
		if !ok {
			t.Fatal("expected to find exit builtin")
		}
		if !fn.NoReturn {
			t.Error("expected exit to have NoReturn=true")
		}
	})

	t.Run("lookup nonexistent", func(t *testing.T) {
		_, ok := registry.Lookup("notABuiltin")
		if ok {
			t.Error("expected not to find notABuiltin")
		}
	})
}

func TestBuiltinRegistry_All(t *testing.T) {
	registry := NewBuiltinRegistry()

	all := registry.All()
	if len(all) != 4 {
		t.Errorf("expected 4 default builtins, got %d", len(all))
	}

	expectedBuiltins := []string{"exit", "print", "len", "sleep"}
	for _, name := range expectedBuiltins {
		if _, ok := all[name]; !ok {
			t.Errorf("expected %q in All() result", name)
		}
	}
}

func TestBuiltinRegistry_DefaultBuiltins(t *testing.T) {
	registry := NewBuiltinRegistry()

	t.Run("exit builtin", func(t *testing.T) {
		fn, ok := registry.Lookup("exit")
		if !ok {
			t.Fatal("exit not found")
		}
		if len(fn.ParamTypes) != 1 {
			t.Errorf("expected 1 param, got %d", len(fn.ParamTypes))
		}
		if fn.ParamTypes[0] != TypeS64 {
			t.Errorf("expected s64 param, got %v", fn.ParamTypes[0])
		}
		if fn.ReturnType != TypeVoid {
			t.Errorf("expected void return, got %v", fn.ReturnType)
		}
		if !fn.NoReturn {
			t.Error("expected NoReturn=true")
		}
	})

	t.Run("print builtin", func(t *testing.T) {
		fn, ok := registry.Lookup("print")
		if !ok {
			t.Fatal("print not found")
		}
		if fn.AcceptedTypes == nil {
			t.Fatal("expected AcceptedTypes to be set")
		}
		acceptedForParam0 := fn.AcceptedTypes[0]
		if len(acceptedForParam0) != 3 {
			t.Errorf("expected 3 accepted types for print, got %d", len(acceptedForParam0))
		}
	})

	t.Run("len builtin", func(t *testing.T) {
		fn, ok := registry.Lookup("len")
		if !ok {
			t.Fatal("len not found")
		}
		if !fn.IsArrayLen {
			t.Error("expected IsArrayLen=true")
		}
		if fn.ReturnType != TypeS64 {
			t.Errorf("expected s64 return, got %v", fn.ReturnType)
		}
	})

	t.Run("sleep builtin", func(t *testing.T) {
		fn, ok := registry.Lookup("sleep")
		if !ok {
			t.Fatal("sleep not found")
		}
		if len(fn.ParamTypes) != 1 {
			t.Errorf("expected 1 param, got %d", len(fn.ParamTypes))
		}
		if fn.ParamTypes[0] != TypeS64 {
			t.Errorf("expected s64 param, got %v", fn.ParamTypes[0])
		}
	})
}

func TestRegisterBuiltin_GlobalFunction(t *testing.T) {
	// Test the global RegisterBuiltin function
	customFn := BuiltinFunc{
		ParamTypes: []Type{TypeString},
		ReturnType: TypeS64,
	}

	RegisterBuiltin("testGlobalFunc", customFn)

	// Verify via LookupBuiltin
	got, ok := LookupBuiltin("testGlobalFunc")
	if !ok {
		t.Fatal("testGlobalFunc not found after global registration")
	}
	if got.ReturnType != TypeS64 {
		t.Errorf("expected s64 return, got %v", got.ReturnType)
	}

	// Also verify it appears in Builtins map
	if _, ok := Builtins["testGlobalFunc"]; !ok {
		t.Error("expected testGlobalFunc in Builtins map")
	}

	// Cleanup
	delete(Builtins, "testGlobalFunc")
}

func TestLookupBuiltin_GlobalFunction(t *testing.T) {
	// Test lookup of default builtins via global function
	fn, ok := LookupBuiltin("exit")
	if !ok {
		t.Fatal("expected to find exit via LookupBuiltin")
	}
	if !fn.NoReturn {
		t.Error("expected NoReturn=true for exit")
	}

	// Test lookup of nonexistent
	_, ok = LookupBuiltin("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent builtin")
	}
}

func TestBuiltins_BackwardCompatibility(t *testing.T) {
	// Verify Builtins map maintains backward compatibility
	if Builtins == nil {
		t.Fatal("Builtins map should not be nil")
	}

	// Verify default builtins are accessible via the map
	if _, ok := Builtins["exit"]; !ok {
		t.Error("expected exit in Builtins map")
	}
	if _, ok := Builtins["print"]; !ok {
		t.Error("expected print in Builtins map")
	}
}

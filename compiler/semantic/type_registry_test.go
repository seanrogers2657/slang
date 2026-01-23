package semantic

import (
	"testing"
)

func TestNewTypeRegistry(t *testing.T) {
	registry := NewTypeRegistry()

	if registry == nil {
		t.Fatal("NewTypeRegistry() returned nil")
	}
	if registry.structs == nil {
		t.Error("structs map is nil")
	}
	if registry.classes == nil {
		t.Error("classes map is nil")
	}
	if registry.objects == nil {
		t.Error("objects map is nil")
	}
}

func TestTypeRegistry_RegisterStruct(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		registry := NewTypeRegistry()
		st := StructType{Name: "Point", Fields: []StructFieldInfo{
			{Name: "x", Type: TypeS64, Index: 0},
			{Name: "y", Type: TypeS64, Index: 1},
		}}

		err := registry.RegisterStruct("Point", st)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify it was registered
		got, ok := registry.LookupStruct("Point")
		if !ok {
			t.Fatal("struct not found after registration")
		}
		if got.Name != "Point" {
			t.Errorf("expected name 'Point', got %q", got.Name)
		}
		if len(got.Fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(got.Fields))
		}
	})

	t.Run("duplicate struct registration", func(t *testing.T) {
		registry := NewTypeRegistry()
		st := StructType{Name: "Point"}

		// First registration should succeed
		err := registry.RegisterStruct("Point", st)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}

		// Second registration should fail
		err = registry.RegisterStruct("Point", st)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})

	t.Run("struct name conflicts with class", func(t *testing.T) {
		registry := NewTypeRegistry()
		registry.RegisterClass("Widget", ClassType{Name: "Widget"})

		err := registry.RegisterStruct("Widget", StructType{Name: "Widget"})
		if err == nil {
			t.Error("expected error when struct name conflicts with class")
		}
	})

	t.Run("struct name conflicts with object", func(t *testing.T) {
		registry := NewTypeRegistry()
		registry.RegisterObject("Utils", ObjectType{Name: "Utils"})

		err := registry.RegisterStruct("Utils", StructType{Name: "Utils"})
		if err == nil {
			t.Error("expected error when struct name conflicts with object")
		}
	})
}

func TestTypeRegistry_RegisterClass(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		registry := NewTypeRegistry()
		ct := ClassType{
			Name:   "Counter",
			Fields: []StructFieldInfo{{Name: "count", Type: TypeS64, Index: 0}},
			Methods: map[string][]*MethodInfo{
				"increment": {{Name: "increment", ParamTypes: []Type{}, ReturnType: TypeVoid}},
			},
		}

		err := registry.RegisterClass("Counter", ct)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		got, ok := registry.LookupClass("Counter")
		if !ok {
			t.Fatal("class not found after registration")
		}
		if got.Name != "Counter" {
			t.Errorf("expected name 'Counter', got %q", got.Name)
		}
	})

	t.Run("duplicate class registration", func(t *testing.T) {
		registry := NewTypeRegistry()
		ct := ClassType{Name: "Counter"}

		registry.RegisterClass("Counter", ct)
		err := registry.RegisterClass("Counter", ct)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})
}

func TestTypeRegistry_RegisterObject(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		registry := NewTypeRegistry()
		ot := ObjectType{
			Name: "Math",
			Methods: map[string][]*MethodInfo{
				"abs": {{Name: "abs", ParamTypes: []Type{TypeS64}, ReturnType: TypeS64, IsStatic: true}},
			},
		}

		err := registry.RegisterObject("Math", ot)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		got, ok := registry.LookupObject("Math")
		if !ok {
			t.Fatal("object not found after registration")
		}
		if got.Name != "Math" {
			t.Errorf("expected name 'Math', got %q", got.Name)
		}
	})

	t.Run("duplicate object registration", func(t *testing.T) {
		registry := NewTypeRegistry()
		ot := ObjectType{Name: "Math"}

		registry.RegisterObject("Math", ot)
		err := registry.RegisterObject("Math", ot)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})
}

func TestTypeRegistry_Lookup(t *testing.T) {
	registry := NewTypeRegistry()

	// Register different types
	registry.RegisterStruct("Point", StructType{Name: "Point"})
	registry.RegisterClass("Counter", ClassType{Name: "Counter"})
	registry.RegisterObject("Utils", ObjectType{Name: "Utils"})

	t.Run("lookup struct", func(t *testing.T) {
		typ, ok := registry.Lookup("Point")
		if !ok {
			t.Fatal("expected to find Point")
		}
		if _, isStruct := typ.(StructType); !isStruct {
			t.Error("expected StructType")
		}
	})

	t.Run("lookup class", func(t *testing.T) {
		typ, ok := registry.Lookup("Counter")
		if !ok {
			t.Fatal("expected to find Counter")
		}
		if _, isClass := typ.(ClassType); !isClass {
			t.Error("expected ClassType")
		}
	})

	t.Run("lookup object", func(t *testing.T) {
		typ, ok := registry.Lookup("Utils")
		if !ok {
			t.Fatal("expected to find Utils")
		}
		if _, isObject := typ.(ObjectType); !isObject {
			t.Error("expected ObjectType")
		}
	})

	t.Run("lookup nonexistent", func(t *testing.T) {
		_, ok := registry.Lookup("NotExists")
		if ok {
			t.Error("expected not to find NotExists")
		}
	})
}

func TestTypeRegistry_NameExists(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterStruct("Point", StructType{Name: "Point"})
	registry.RegisterClass("Counter", ClassType{Name: "Counter"})
	registry.RegisterObject("Utils", ObjectType{Name: "Utils"})

	tests := []struct {
		name     string
		lookup   string
		wantKind TypeKind
		wantOK   bool
	}{
		{"struct exists", "Point", TypeKindStruct, true},
		{"class exists", "Counter", TypeKindClass, true},
		{"object exists", "Utils", TypeKindObject, true},
		{"not exists", "NotExists", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, ok := registry.NameExists(tt.lookup)
			if ok != tt.wantOK {
				t.Errorf("NameExists() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && kind != tt.wantKind {
				t.Errorf("NameExists() kind = %v, want %v", kind, tt.wantKind)
			}
		})
	}
}

func TestTypeRegistry_Update(t *testing.T) {
	t.Run("update struct", func(t *testing.T) {
		registry := NewTypeRegistry()
		registry.RegisterStruct("Point", StructType{Name: "Point", Fields: nil})

		// Update with fields
		registry.UpdateStruct("Point", StructType{
			Name:   "Point",
			Fields: []StructFieldInfo{{Name: "x", Type: TypeS64, Index: 0}},
		})

		got, _ := registry.LookupStruct("Point")
		if len(got.Fields) != 1 {
			t.Errorf("expected 1 field after update, got %d", len(got.Fields))
		}
	})

	t.Run("update class", func(t *testing.T) {
		registry := NewTypeRegistry()
		registry.RegisterClass("Counter", ClassType{Name: "Counter"})

		registry.UpdateClass("Counter", ClassType{
			Name:   "Counter",
			Fields: []StructFieldInfo{{Name: "count", Type: TypeS64, Index: 0}},
		})

		got, _ := registry.LookupClass("Counter")
		if len(got.Fields) != 1 {
			t.Errorf("expected 1 field after update, got %d", len(got.Fields))
		}
	})

	t.Run("update object", func(t *testing.T) {
		registry := NewTypeRegistry()
		registry.RegisterObject("Math", ObjectType{Name: "Math", Methods: nil})

		registry.UpdateObject("Math", ObjectType{
			Name:    "Math",
			Methods: map[string][]*MethodInfo{"abs": {}},
		})

		got, _ := registry.LookupObject("Math")
		if len(got.Methods) != 1 {
			t.Errorf("expected 1 method after update, got %d", len(got.Methods))
		}
	})
}

func TestTypeRegistry_AllMethods(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterStruct("Point", StructType{Name: "Point"})
	registry.RegisterStruct("Rect", StructType{Name: "Rect"})
	registry.RegisterClass("Counter", ClassType{Name: "Counter"})
	registry.RegisterObject("Utils", ObjectType{Name: "Utils"})

	t.Run("AllStructs", func(t *testing.T) {
		all := registry.AllStructs()
		if len(all) != 2 {
			t.Errorf("expected 2 structs, got %d", len(all))
		}
		if _, ok := all["Point"]; !ok {
			t.Error("expected Point in AllStructs")
		}
		if _, ok := all["Rect"]; !ok {
			t.Error("expected Rect in AllStructs")
		}
	})

	t.Run("AllClasses", func(t *testing.T) {
		all := registry.AllClasses()
		if len(all) != 1 {
			t.Errorf("expected 1 class, got %d", len(all))
		}
		if _, ok := all["Counter"]; !ok {
			t.Error("expected Counter in AllClasses")
		}
	})

	t.Run("AllObjects", func(t *testing.T) {
		all := registry.AllObjects()
		if len(all) != 1 {
			t.Errorf("expected 1 object, got %d", len(all))
		}
		if _, ok := all["Utils"]; !ok {
			t.Error("expected Utils in AllObjects")
		}
	})
}

func TestTypeRegistry_Clear(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterStruct("Point", StructType{Name: "Point"})
	registry.RegisterClass("Counter", ClassType{Name: "Counter"})
	registry.RegisterObject("Utils", ObjectType{Name: "Utils"})

	registry.Clear()

	if len(registry.AllStructs()) != 0 {
		t.Error("expected empty structs after Clear")
	}
	if len(registry.AllClasses()) != 0 {
		t.Error("expected empty classes after Clear")
	}
	if len(registry.AllObjects()) != 0 {
		t.Error("expected empty objects after Clear")
	}
}

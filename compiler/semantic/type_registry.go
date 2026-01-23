package semantic

import "fmt"

// TypeKind represents the kind of a registered type
type TypeKind string

const (
	TypeKindStruct TypeKind = "struct"
	TypeKindClass  TypeKind = "class"
	TypeKindObject TypeKind = "object"
)

// TypeRegistry centralizes struct/class/object management.
// It provides unified lookup and registration with automatic duplicate checking.
type TypeRegistry struct {
	structs map[string]StructType
	classes map[string]ClassType
	objects map[string]ObjectType
}

// NewTypeRegistry creates a new empty type registry
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		structs: make(map[string]StructType),
		classes: make(map[string]ClassType),
		objects: make(map[string]ObjectType),
	}
}

// RegisterStruct adds a struct to the registry.
// Returns an error if the name is already taken by any type.
func (r *TypeRegistry) RegisterStruct(name string, st StructType) error {
	if kind, exists := r.NameExists(name); exists {
		return fmt.Errorf("type '%s' is already declared as %s", name, kind)
	}
	r.structs[name] = st
	return nil
}

// RegisterClass adds a class to the registry.
// Returns an error if the name is already taken by any type.
func (r *TypeRegistry) RegisterClass(name string, ct ClassType) error {
	if kind, exists := r.NameExists(name); exists {
		return fmt.Errorf("type '%s' is already declared as %s", name, kind)
	}
	r.classes[name] = ct
	return nil
}

// RegisterObject adds an object to the registry.
// Returns an error if the name is already taken by any type.
func (r *TypeRegistry) RegisterObject(name string, ot ObjectType) error {
	if kind, exists := r.NameExists(name); exists {
		return fmt.Errorf("type '%s' is already declared as %s", name, kind)
	}
	r.objects[name] = ot
	return nil
}

// Lookup finds a type by name in any registry (struct, class, or object).
// Returns the type and true if found, or nil and false if not found.
func (r *TypeRegistry) Lookup(name string) (Type, bool) {
	if st, ok := r.structs[name]; ok {
		return st, true
	}
	if ct, ok := r.classes[name]; ok {
		return ct, true
	}
	if ot, ok := r.objects[name]; ok {
		return ot, true
	}
	return nil, false
}

// LookupStruct finds a struct by name.
// Returns the struct type and true if found, or empty struct and false if not found.
func (r *TypeRegistry) LookupStruct(name string) (StructType, bool) {
	st, ok := r.structs[name]
	return st, ok
}

// LookupClass finds a class by name.
// Returns the class type and true if found, or empty class and false if not found.
func (r *TypeRegistry) LookupClass(name string) (ClassType, bool) {
	ct, ok := r.classes[name]
	return ct, ok
}

// LookupObject finds an object by name.
// Returns the object type and true if found, or empty object and false if not found.
func (r *TypeRegistry) LookupObject(name string) (ObjectType, bool) {
	ot, ok := r.objects[name]
	return ot, ok
}

// NameExists checks if a name exists in any registry.
// Returns the kind ("struct", "class", or "object") and true if found,
// or empty string and false if not found.
func (r *TypeRegistry) NameExists(name string) (TypeKind, bool) {
	if _, ok := r.structs[name]; ok {
		return TypeKindStruct, true
	}
	if _, ok := r.classes[name]; ok {
		return TypeKindClass, true
	}
	if _, ok := r.objects[name]; ok {
		return TypeKindObject, true
	}
	return "", false
}

// UpdateStruct updates an existing struct in the registry.
// This is used for second-pass field resolution.
func (r *TypeRegistry) UpdateStruct(name string, st StructType) {
	r.structs[name] = st
}

// UpdateClass updates an existing class in the registry.
// This is used for second-pass field/method resolution.
func (r *TypeRegistry) UpdateClass(name string, ct ClassType) {
	r.classes[name] = ct
}

// UpdateObject updates an existing object in the registry.
// This is used for second-pass method resolution.
func (r *TypeRegistry) UpdateObject(name string, ot ObjectType) {
	r.objects[name] = ot
}

// AllStructs returns all registered structs
func (r *TypeRegistry) AllStructs() map[string]StructType {
	return r.structs
}

// AllClasses returns all registered classes
func (r *TypeRegistry) AllClasses() map[string]ClassType {
	return r.classes
}

// AllObjects returns all registered objects
func (r *TypeRegistry) AllObjects() map[string]ObjectType {
	return r.objects
}

// Clear removes all registered types from the registry
func (r *TypeRegistry) Clear() {
	r.structs = make(map[string]StructType)
	r.classes = make(map[string]ClassType)
	r.objects = make(map[string]ObjectType)
}

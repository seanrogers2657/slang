package semantic

// ExportKind distinguishes the nature of an exported symbol.
type ExportKind int

const (
	ExportFunc ExportKind = iota // function declaration
	ExportType                   // struct, class, or object declaration
	ExportVal                    // immutable variable (val)
	ExportVar                    // mutable variable (var)
)

// Export represents a single public symbol from a package.
type Export struct {
	Type Type       // the symbol's type (FunctionType, StructType, etc.)
	Kind ExportKind // what kind of declaration this is
}

// PackageNamespace is bound to an import name in the analyzer's scope.
// When the analyzer processes `import "math"`, it creates a
// PackageNamespace and binds it to "math" in the current scope.
type PackageNamespace struct {
	Path    string            // canonical import path (e.g., "math")
	Exports map[string]Export // references Package.Exports directly
}

// PackageNamespaceType is a sentinel type used to store a PackageNamespace
// in the analyzer's scope. It is not a real runtime type.
type PackageNamespaceType struct {
	Namespace *PackageNamespace
}

func (t PackageNamespaceType) String() string  { return "<package " + t.Namespace.Path + ">" }
func (t PackageNamespaceType) Equals(other Type) bool { return false }

// ExtractExports builds an Export map from a TypedProgram's declarations.
func ExtractExports(typed *TypedProgram) map[string]Export {
	exports := make(map[string]Export)

	for _, decl := range typed.Declarations {
		switch d := decl.(type) {
		case *TypedFunctionDecl:
			paramTypes := make([]Type, len(d.Parameters))
			for i, p := range d.Parameters {
				paramTypes[i] = p.Type
			}
			exports[d.Name] = Export{
				Type: FunctionType{
					ParamTypes: paramTypes,
					ReturnType: d.ReturnType,
				},
				Kind: ExportFunc,
			}
		case *TypedStructDecl:
			exports[d.StructType.Name] = Export{
				Type: d.StructType,
				Kind: ExportType,
			}
		case *TypedClassDecl:
			exports[d.ClassType.Name] = Export{
				Type: d.ClassType,
				Kind: ExportType,
			}
		case *TypedObjectDecl:
			exports[d.ObjectType.Name] = Export{
				Type: d.ObjectType,
				Kind: ExportType,
			}
		}
	}

	// Export top-level variables (val/var)
	for _, stmt := range typed.Statements {
		if varDecl, ok := stmt.(*TypedVarDeclStmt); ok {
			kind := ExportVal
			if varDecl.Mutable {
				kind = ExportVar
			}
			exports[varDecl.Name] = Export{
				Type: varDecl.DeclaredType,
				Kind: kind,
			}
		}
	}

	return exports
}

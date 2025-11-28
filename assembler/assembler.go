package assembler

// Assembler is the interface for building executable programs from assembly
type Assembler interface {
	// Build performs the complete build process from assembly string to executable
	Build(assembly string, opts BuildOptions) error

	// Assemble converts an assembly file (.s) to an object file (.o)
	Assemble(inputPath, outputPath string) error

	// Link creates an executable from object files
	Link(objectFiles []string, outputPath string) error
}

// BuildOptions contains options for building an executable
type BuildOptions struct {
	// AssemblyPath is the path to write the .s file (optional, defaults to outputPath + ".s")
	AssemblyPath string
	// ObjectPath is the path to write the .o file (optional, defaults to outputPath + ".o")
	ObjectPath string
	// OutputPath is the path to write the executable
	OutputPath string
	// KeepIntermediates determines whether to keep .s and .o files after building
	KeepIntermediates bool
}

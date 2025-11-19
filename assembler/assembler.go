package assembler

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// Assembler is the interface for building executable programs from assembly
type Assembler interface {
	// Build performs the complete build process from assembly string to executable
	Build(assembly string, opts BuildOptions) error
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

// SystemAssembler is a concrete implementation of Assembler for the current system
type SystemAssembler struct {
	// Arch specifies the target architecture (default: arm64)
	Arch string
	// SDKPath is the path to the macOS SDK (optional, will attempt to detect if not set)
	SDKPath string
	// EntryPoint is the entry point symbol (default: _start)
	EntryPoint string
	// SystemLibs specifies whether to link system libraries (default: true)
	SystemLibs bool
}

// New creates a new SystemAssembler with default settings
func New() *SystemAssembler {
	return &SystemAssembler{
		Arch:       "arm64",
		EntryPoint: "_start",
		SystemLibs: true,
	}
}

// Assemble converts an assembly file (.s) to an object file (.o)
func (a *SystemAssembler) Assemble(inputPath, outputPath string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run the assembler
	cmd := exec.Command("as", "-arch", a.Arch, inputPath, "-o", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("assembly failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Link creates an executable from object files
func (a *SystemAssembler) Link(objectFiles []string, outputPath string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Detect SDK path if not set
	sdkPath := a.SDKPath
	if sdkPath == "" {
		var err error
		sdkPath, err = detectSDKPath()
		if err != nil {
			return fmt.Errorf("failed to detect SDK path: %w", err)
		}
	}

	// Build linker arguments
	args := []string{
		"-o", outputPath,
	}
	args = append(args, objectFiles...)

	if a.SystemLibs {
		args = append(args, "-lSystem")
	}

	args = append(args,
		"-syslibroot", sdkPath,
		"-e", a.EntryPoint,
		"-arch", a.Arch,
	)

	// Run the linker
	cmd := exec.Command("ld", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("linking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Build performs the complete build process: write assembly, assemble, and link
func (a *SystemAssembler) Build(assembly string, opts BuildOptions) error {
	// Set defaults
	assemblyPath := opts.AssemblyPath
	if assemblyPath == "" {
		assemblyPath = opts.OutputPath + ".s"
	}
	objectPath := opts.ObjectPath
	if objectPath == "" {
		objectPath = opts.OutputPath + ".o"
	}

	// Ensure directory exists for assembly file
	assemblyDir := filepath.Dir(assemblyPath)
	if assemblyDir != "." && assemblyDir != "" {
		if err := os.MkdirAll(assemblyDir, 0755); err != nil {
			return fmt.Errorf("failed to create assembly directory: %w", err)
		}
	}

	// Write assembly file
	if err := os.WriteFile(assemblyPath, []byte(assembly), fs.ModePerm); err != nil {
		return fmt.Errorf("failed to write assembly: %w", err)
	}

	// Clean up assembly file if requested
	if !opts.KeepIntermediates {
		defer os.Remove(assemblyPath)
	}

	// Assemble
	if err := a.Assemble(assemblyPath, objectPath); err != nil {
		return err
	}

	// Clean up object file if requested
	if !opts.KeepIntermediates {
		defer os.Remove(objectPath)
	}

	// Link
	if err := a.Link([]string{objectPath}, opts.OutputPath); err != nil {
		return err
	}

	return nil
}

// detectSDKPath attempts to find the macOS SDK path
func detectSDKPath() (string, error) {
	// Try xcrun to get the SDK path
	cmd := exec.Command("xcrun", "--show-sdk-path")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return string(output[:len(output)-1]), nil // trim newline
	}

	// Fallback to common locations
	commonPaths := []string{
		"/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk",
		"/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX15.5.sdk",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not detect macOS SDK path")
}

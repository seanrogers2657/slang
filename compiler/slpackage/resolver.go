package slpackage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/seanrogers2657/slang/compiler/stdlib"
)

// PackageResolver translates import paths to filesystem directories
// within the packages/ directory under the project root.
type PackageResolver struct {
	RootDir     string // project root (entry file's directory)
	PackagesDir string // RootDir + "/packages"
	StdFS       fs.FS  // embedded standard library ("std/..." import paths)
}

// resolvedPkg describes where a package's .sl files can be read from.
type resolvedPkg struct {
	fsys    fs.FS  // filesystem containing the package
	subdir  string // directory within fsys ("." for on-disk packages)
	display string // path used in error messages and FileAST.Path
}

// NewResolver creates a PackageResolver for the given project root.
func NewResolver(rootDir string) *PackageResolver {
	return &PackageResolver{
		RootDir:     rootDir,
		PackagesDir: filepath.Join(rootDir, "packages"),
		StdFS:       stdlib.FS,
	}
}

// Resolve validates an import path and returns the display path for the
// package (an absolute directory for on-disk packages, or the import path
// itself for embedded stdlib packages). Returns an error if the path is
// invalid or the package doesn't exist.
func (r *PackageResolver) Resolve(importPath string) (string, error) {
	rp, err := r.resolvePackage(importPath)
	if err != nil {
		return "", err
	}
	return rp.display, nil
}

// resolvePackage validates an import path and returns where its .sl files live.
func (r *PackageResolver) resolvePackage(importPath string) (*resolvedPkg, error) {
	// Reject relative/absolute path prefixes
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") || strings.HasPrefix(importPath, "/") {
		return nil, fmt.Errorf("import path %q must not start with './', '../', or '/'; use a package-relative path like \"math\" or \"utils/helpers\"", importPath)
	}

	// Reject empty path
	if importPath == "" {
		return nil, fmt.Errorf("import path must not be empty")
	}

	// Validate path segments
	segments := strings.Split(importPath, "/")
	for _, seg := range segments {
		if !isValidPathSegment(seg) {
			return nil, fmt.Errorf("invalid import path segment %q: must match [a-z][a-z0-9_]*", seg)
		}
	}

	// Reserved: "main" names the root package.
	if importPath == "main" {
		return nil, fmt.Errorf("import path \"main\" is reserved for the root package")
	}

	// Standard library: served from the embedded filesystem, not packages/.
	if importPath == "std" || strings.HasPrefix(importPath, "std/") {
		return r.resolveStdPackage(importPath)
	}

	// Check packages/ directory exists
	info, err := os.Stat(r.PackagesDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no 'packages' directory found; create a 'packages/' directory in the project root to use imports")
	}
	if err != nil {
		return nil, fmt.Errorf("error accessing packages directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("'packages' exists but is not a directory")
	}

	// Resolve to absolute path
	pkgDir := filepath.Join(r.PackagesDir, filepath.FromSlash(importPath))

	// Check package directory exists
	pkgInfo, err := os.Stat(pkgDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("package %q not found; expected a directory at %s", importPath, pkgDir)
	}
	if err != nil {
		return nil, fmt.Errorf("error accessing package %q: %w", importPath, err)
	}
	if !pkgInfo.IsDir() {
		return nil, fmt.Errorf("package path %q is not a directory", importPath)
	}

	// Check directory has .sl files
	files, err := DiscoverSlFiles(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("error reading package %q: %w", importPath, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("package %q has no .sl files", importPath)
	}

	return &resolvedPkg{fsys: os.DirFS(pkgDir), subdir: ".", display: pkgDir}, nil
}

// resolveStdPackage resolves a "std/..." import against the embedded stdlib.
func (r *PackageResolver) resolveStdPackage(importPath string) (*resolvedPkg, error) {
	if r.StdFS == nil {
		return nil, fmt.Errorf("standard library is unavailable in this build")
	}
	if importPath == "std" {
		return nil, fmt.Errorf("import path \"std\" is not a package; import a submodule like \"std/math\"")
	}

	info, err := fs.Stat(r.StdFS, importPath)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("standard library package %q not found", importPath)
	}

	names, err := slFileNames(r.StdFS, importPath)
	if err != nil {
		return nil, fmt.Errorf("error reading standard library package %q: %w", importPath, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("standard library package %q has no .sl files", importPath)
	}

	return &resolvedPkg{fsys: r.StdFS, subdir: importPath, display: importPath}, nil
}

// DiscoverSlFiles finds all .sl files in a directory, sorted alphabetically.
// It does not recurse into subdirectories.
func DiscoverSlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sl") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// slFileNames returns the base names of .sl files in a directory of an fs.FS,
// sorted alphabetically. It does not recurse into subdirectories.
func slFileNames(fsys fs.FS, subdir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, subdir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sl") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// isValidPathSegment checks that a path segment matches [a-z][a-z0-9_]*.
func isValidPathSegment(seg string) bool {
	if len(seg) == 0 {
		return false
	}
	first := rune(seg[0])
	if !unicode.IsLower(first) || !unicode.IsLetter(first) {
		return false
	}
	for _, ch := range seg[1:] {
		if !unicode.IsLower(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}

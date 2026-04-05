package slpackage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to create a project with files
func setupProject(t *testing.T, files map[string]string) string {
	t.Helper()
	rootDir := t.TempDir()
	for relPath, content := range files {
		absPath := filepath.Join(rootDir, relPath)
		os.MkdirAll(filepath.Dir(absPath), 0755)
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", relPath, err)
		}
	}
	return rootDir
}

func TestCompilerSingleFile(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl": "main = () { print(42) }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	pkgFiles, errs := c.DiscoverAndParse()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// Should have one package ("main") with one file
	if len(pkgFiles) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgFiles))
	}
	mainFiles, ok := pkgFiles["main"]
	if !ok {
		t.Fatal("expected 'main' package")
	}
	if len(mainFiles) != 1 {
		t.Fatalf("expected 1 file in main package, got %d", len(mainFiles))
	}

	// Analysis order should be just ["main"]
	if len(c.AnalysisOrder) != 1 || c.AnalysisOrder[0] != "main" {
		t.Errorf("expected analysis order [main], got %v", c.AnalysisOrder)
	}
}

func TestCompilerMultiFileRootPackage(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl":    "main = () { print(helper()) }",
		"helpers.sl": "helper = () -> s64 { return 42 }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	rootFiles := []string{
		filepath.Join(rootDir, "helpers.sl"),
		filepath.Join(rootDir, "main.sl"),
	}
	c := NewCompiler(rootDir, entryFile, rootFiles)
	pkgFiles, errs := c.DiscoverAndParse()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(pkgFiles["main"]) != 2 {
		t.Fatalf("expected 2 files in main package, got %d", len(pkgFiles["main"]))
	}
}

func TestCompilerSingleImport(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl": "import \"math\"\nmain = () { print(42) }",
		"packages/math/math.sl": "add = (a: s64, b: s64) -> s64 { return a + b }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	pkgFiles, errs := c.DiscoverAndParse()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// Two packages: main and math
	if len(pkgFiles) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgFiles))
	}
	if _, ok := pkgFiles["math"]; !ok {
		t.Fatal("expected 'math' package")
	}

	// Analysis order: math before main
	if len(c.AnalysisOrder) != 2 {
		t.Fatalf("expected 2 in analysis order, got %d", len(c.AnalysisOrder))
	}
	if c.AnalysisOrder[0] != "math" {
		t.Errorf("expected math first in analysis order, got %s", c.AnalysisOrder[0])
	}
	if c.AnalysisOrder[1] != "main" {
		t.Errorf("expected main second in analysis order, got %s", c.AnalysisOrder[1])
	}
}

func TestCompilerTransitiveImports(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl": "import \"validator\"\nmain = () { }",
		"packages/validator/validator.sl": "import \"logger\"\nvalidate = (x: s64) -> bool { return true }",
		"packages/logger/logger.sl":       "log = (msg: string) { print(msg) }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	pkgFiles, errs := c.DiscoverAndParse()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(pkgFiles) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgFiles))
	}

	// Analysis order: logger, validator, main
	if len(c.AnalysisOrder) != 3 {
		t.Fatalf("expected 3 in analysis order, got %d", len(c.AnalysisOrder))
	}
	if c.AnalysisOrder[0] != "logger" {
		t.Errorf("expected logger first, got %s", c.AnalysisOrder[0])
	}
	if c.AnalysisOrder[1] != "validator" {
		t.Errorf("expected validator second, got %s", c.AnalysisOrder[1])
	}
	if c.AnalysisOrder[2] != "main" {
		t.Errorf("expected main third, got %s", c.AnalysisOrder[2])
	}
}

func TestCompilerCircularDependency(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl":            "import \"a\"\nmain = () { }",
		"packages/a/a.sl":   "import \"b\"\nfoo = () -> s64 { return 1 }",
		"packages/b/b.sl":   "import \"a\"\nbar = () -> s64 { return 2 }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	_, errs := c.DiscoverAndParse()

	if len(errs) == 0 {
		t.Fatal("expected circular dependency error, got none")
	}

	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "circular dependency") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected circular dependency error, got: %v", errs)
	}
}

func TestCompilerMissingPackage(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl": "import \"nonexistent\"\nmain = () { }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	_, errs := c.DiscoverAndParse()

	if len(errs) == 0 {
		t.Fatal("expected error, got none")
	}

	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "no 'packages' directory found") || strings.Contains(e.Error(), "not found") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing package error, got: %v", errs)
	}
}

func TestCompilerMultiFileDependency(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl": "import \"utils\"\nmain = () { }",
		"packages/utils/format.sl":  "format_s64 = (n: s64) -> s64 { return n }",
		"packages/utils/convert.sl": "to_hex = (n: s64) -> s64 { return n }",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	pkgFiles, errs := c.DiscoverAndParse()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	utilsFiles := pkgFiles["utils"]
	if len(utilsFiles) != 2 {
		t.Fatalf("expected 2 files in utils package, got %d", len(utilsFiles))
	}

	// Files should be in alphabetical order
	if filepath.Base(utilsFiles[0].Path) != "convert.sl" {
		t.Errorf("expected convert.sl first, got %s", filepath.Base(utilsFiles[0].Path))
	}
	if filepath.Base(utilsFiles[1].Path) != "format.sl" {
		t.Errorf("expected format.sl second, got %s", filepath.Base(utilsFiles[1].Path))
	}
}

func TestCompilerLooseFilesWarning(t *testing.T) {
	rootDir := setupProject(t, map[string]string{
		"main.sl":                      "import \"math\"\nmain = () { print(1) }",
		"packages/math/math.sl":        "add = (a: s64, b: s64) -> s64 { return a + b }",
		"packages/loose.sl":            "// this should produce a warning",
	})

	entryFile := filepath.Join(rootDir, "main.sl")
	c := NewCompiler(rootDir, entryFile, []string{entryFile})
	_, errs := c.DiscoverAndParse()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(c.Warnings) == 0 {
		t.Fatal("expected warning about loose file, got none")
	}

	found := false
	for _, w := range c.Warnings {
		if strings.Contains(w.Message, "loose.sl") && strings.Contains(w.Message, "not part of any package") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about loose.sl, got: %v", c.Warnings)
	}
}

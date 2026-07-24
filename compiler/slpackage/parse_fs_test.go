package slpackage

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// TestParseFilesFS_EmbeddedDisplayPaths verifies that files read from a
// (simulated) embedded filesystem are discovered in sorted order and carry
// clean, slash-joined display paths — not the raw fs read path.
func TestParseFilesFS_EmbeddedDisplayPaths(t *testing.T) {
	fsys := fstest.MapFS{
		"std/foo/b.sl":     {Data: []byte("b = () -> s64 { 2 }")},
		"std/foo/a.sl":     {Data: []byte("a = () -> s64 { 1 }")},
		"std/foo/notes.md": {Data: []byte("# not slang")},
	}
	rp := &resolvedPkg{fsys: fsys, subdir: "std/foo", display: "std/foo"}

	fileASTs, errs := (&PackageCompiler{}).parseFilesFS(rp)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(fileASTs) != 2 {
		t.Fatalf("expected 2 .sl files (the .md ignored), got %d", len(fileASTs))
	}
	if fileASTs[0].Path != "std/foo/a.sl" {
		t.Errorf("expected first path std/foo/a.sl, got %q", fileASTs[0].Path)
	}
	if fileASTs[1].Path != "std/foo/b.sl" {
		t.Errorf("expected second path std/foo/b.sl, got %q", fileASTs[1].Path)
	}
}

// TestParseFilesFS_NestedSubdir verifies the read path is joined with forward
// slashes (fs.FS semantics) even for deeply nested std packages.
func TestParseFilesFS_NestedSubdir(t *testing.T) {
	fsys := fstest.MapFS{
		"std/a/b/c.sl": {Data: []byte("c = () -> s64 { 3 }")},
	}
	rp := &resolvedPkg{fsys: fsys, subdir: "std/a/b", display: "std/a/b"}

	fileASTs, errs := (&PackageCompiler{}).parseFilesFS(rp)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(fileASTs) != 1 || fileASTs[0].Path != "std/a/b/c.sl" {
		t.Fatalf("expected std/a/b/c.sl, got %v", fileASTs)
	}
}

// TestParseFilesFS_ParseErrorCarriesDisplayPath verifies that a syntax error in
// an embedded file is reported against its display path, not a bare filename.
func TestParseFilesFS_ParseErrorCarriesDisplayPath(t *testing.T) {
	fsys := fstest.MapFS{
		// Unterminated function — should fail lexing/parsing.
		"std/broken/bad.sl": {Data: []byte("bad = (")},
	}
	rp := &resolvedPkg{fsys: fsys, subdir: "std/broken", display: "std/broken"}

	fileASTs, errs := (&PackageCompiler{}).parseFilesFS(rp)
	if len(errs) == 0 {
		t.Fatalf("expected a parse error, got none (parsed %d files)", len(fileASTs))
	}
	if len(fileASTs) != 0 {
		t.Errorf("expected no successfully-parsed files, got %d", len(fileASTs))
	}
	found := false
	for _, e := range errs {
		if e.Filename == "std/broken/bad.sl" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error carrying display path std/broken/bad.sl, got: %v", errs)
	}
}

// TestParseFilesFS_OnDiskPackage verifies the same helper reads real on-disk
// packages via os.DirFS, with absolute display paths matching the old behavior.
func TestParseFilesFS_OnDiskPackage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.sl"), []byte("m = () -> s64 { 5 }"), 0644); err != nil {
		t.Fatal(err)
	}
	rp := &resolvedPkg{fsys: os.DirFS(dir), subdir: ".", display: dir}

	fileASTs, errs := (&PackageCompiler{}).parseFilesFS(rp)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	want := filepath.Join(dir, "m.sl")
	if len(fileASTs) != 1 || fileASTs[0].Path != want {
		t.Fatalf("expected %q, got %v", want, fileASTs)
	}
}

// TestResolveAndParseStdMath is an integration check that the real embedded
// std/math resolves and parses through the same path production uses.
func TestResolveAndParseStdMath(t *testing.T) {
	r := NewResolver(t.TempDir())
	rp, err := r.resolvePackage("std/math")
	if err != nil {
		t.Fatalf("resolvePackage(std/math): %v", err)
	}
	if rp.subdir != "std/math" || rp.display != "std/math" {
		t.Fatalf("unexpected resolved package: %+v", rp)
	}

	fileASTs, errs := (&PackageCompiler{}).parseFilesFS(rp)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors for std/math: %v", errs)
	}
	if len(fileASTs) == 0 {
		t.Fatal("expected std/math to parse at least one file")
	}
	if fileASTs[0].Path != "std/math/math.sl" {
		t.Errorf("expected std/math/math.sl, got %q", fileASTs[0].Path)
	}
}

// TestResolveStdPackage_Errors covers the stdlib-specific error paths using an
// injected filesystem so the assertions don't depend on shipped contents.
func TestResolveStdPackage_Errors(t *testing.T) {
	r := NewResolver(t.TempDir())
	r.StdFS = fstest.MapFS{
		"std/real/x.sl":    {Data: []byte("x = () -> s64 { 1 }")},
		"std/empty/doc.md": {Data: []byte("no sl here")},
	}

	// Present package resolves.
	if _, err := r.resolvePackage("std/real"); err != nil {
		t.Errorf("expected std/real to resolve, got: %v", err)
	}
	// Bare std is not a package.
	if _, err := r.resolvePackage("std"); err == nil {
		t.Error("expected bare 'std' to error")
	}
	// Missing package.
	if _, err := r.resolvePackage("std/missing"); err == nil {
		t.Error("expected std/missing to error")
	}
	// Directory with no .sl files.
	if _, err := r.resolvePackage("std/empty"); err == nil {
		t.Error("expected std/empty (no .sl files) to error")
	}

	// A build without an embedded stdlib should degrade cleanly.
	r.StdFS = nil
	if _, err := r.resolvePackage("std/real"); err == nil {
		t.Error("expected an error when StdFS is nil")
	}
}

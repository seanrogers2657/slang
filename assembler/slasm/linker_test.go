package slasm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkerSingleObjectFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "linker_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test assembly source
	srcPath := filepath.Join(tmpDir, "test.s")
	objPath := filepath.Join(tmpDir, "test.o")
	outPath := filepath.Join(tmpDir, "test")

	asmSource := `.global _start

.text
_start:
    mov x0, #42
    mov x16, #1
    svc #0
`
	if err := os.WriteFile(srcPath, []byte(asmSource), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Assemble to object file
	asm := New()
	err = asm.Assemble(srcPath, objPath)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Verify object file exists
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		t.Fatal("Object file was not created")
	}

	// Link the object file
	err = asm.Link([]string{objPath}, outPath)
	if err != nil {
		t.Fatalf("Link failed: %v", err)
	}

	// Verify output executable exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("Output executable was not created")
	}

	// Verify it's executable
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("Failed to stat output: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("Output file is not executable")
	}
}

func TestLinkerMultipleObjectFiles(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "linker_test_multi")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write first assembly file (main with _start)
	src1Path := filepath.Join(tmpDir, "main.s")
	obj1Path := filepath.Join(tmpDir, "main.o")

	asmSource1 := `.global _start

.text
_start:
    mov x0, #50
    mov x16, #1
    svc #0
`
	if err := os.WriteFile(src1Path, []byte(asmSource1), 0644); err != nil {
		t.Fatalf("Failed to write source file 1: %v", err)
	}

	// Write second assembly file (helper function)
	src2Path := filepath.Join(tmpDir, "helper.s")
	obj2Path := filepath.Join(tmpDir, "helper.o")

	asmSource2 := `.global _helper

.text
_helper:
    add x0, x0, x1
    ret
`
	if err := os.WriteFile(src2Path, []byte(asmSource2), 0644); err != nil {
		t.Fatalf("Failed to write source file 2: %v", err)
	}

	outPath := filepath.Join(tmpDir, "linked")

	// Assemble both files
	asm := New()
	err = asm.Assemble(src1Path, obj1Path)
	if err != nil {
		t.Fatalf("Assemble main failed: %v", err)
	}

	err = asm.Assemble(src2Path, obj2Path)
	if err != nil {
		t.Fatalf("Assemble helper failed: %v", err)
	}

	// Link both object files
	err = asm.Link([]string{obj1Path, obj2Path}, outPath)
	if err != nil {
		t.Fatalf("Link failed: %v", err)
	}

	// Verify output exists and is executable
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("Failed to stat output: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("Output file is not executable")
	}
}

func TestLinkerEmptyObjectList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "linker_test_empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "output")

	asm := New()
	err = asm.Link([]string{}, outPath)
	if err == nil {
		t.Error("Expected error when linking empty object list, got nil")
	}
}

func TestLinkerMissingObjectFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "linker_test_missing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "output")

	asm := New()
	err = asm.Link([]string{"/nonexistent/file.o"}, outPath)
	if err == nil {
		t.Error("Expected error when linking nonexistent file, got nil")
	}
}

func TestReadObjectFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "object_reader_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test assembly source
	srcPath := filepath.Join(tmpDir, "test.s")
	objPath := filepath.Join(tmpDir, "test.o")

	asmSource := `.global _start

.text
_start:
    mov x0, #42
    mov x16, #1
    svc #0
`
	if err := os.WriteFile(srcPath, []byte(asmSource), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Assemble to object file
	asm := New()
	err = asm.Assemble(srcPath, objPath)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Read the object file back
	obj, err := ReadObjectFile(objPath)
	if err != nil {
		t.Fatalf("ReadObjectFile failed: %v", err)
	}

	// Verify header
	if obj.Magic != MH_MAGIC_64 {
		t.Errorf("Expected magic 0x%x, got 0x%x", MH_MAGIC_64, obj.Magic)
	}
	if obj.FileType != MH_OBJECT {
		t.Errorf("Expected file type MH_OBJECT (%d), got %d", MH_OBJECT, obj.FileType)
	}

	// Verify text section exists
	if obj.TextSection == nil {
		t.Error("Expected text section, got nil")
	} else {
		if len(obj.TextSection.Data) == 0 {
			t.Error("Expected non-empty text section")
		}
	}

	// Verify symbols contain _start
	foundStart := false
	for _, sym := range obj.Symbols {
		if sym.Name == "_start" {
			foundStart = true
			if !sym.Extern {
				t.Error("Expected _start to be external (global)")
			}
			break
		}
	}
	if !foundStart {
		t.Error("Expected to find _start symbol")
	}
}

func TestReadObjectFileInvalid(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "object_reader_invalid")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with non-existent file
	_, err = ReadObjectFile("/nonexistent/file.o")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with empty file
	emptyPath := filepath.Join(tmpDir, "empty.o")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}
	_, err = ReadObjectFile(emptyPath)
	if err == nil {
		t.Error("Expected error for empty file")
	}

	// Test with invalid magic
	invalidPath := filepath.Join(tmpDir, "invalid.o")
	invalidData := make([]byte, 64)
	invalidData[0] = 0xFF // Invalid magic
	if err := os.WriteFile(invalidPath, invalidData, 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}
	_, err = ReadObjectFile(invalidPath)
	if err == nil {
		t.Error("Expected error for invalid magic")
	}
}

func TestLinkerSymbolTable(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "linker_symtab_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write assembly with multiple symbols
	srcPath := filepath.Join(tmpDir, "test.s")
	objPath := filepath.Join(tmpDir, "test.o")

	asmSource := `.global _start
.global _helper

.text
_start:
    mov x0, #42
    b _helper

_helper:
    mov x16, #1
    svc #0
`
	if err := os.WriteFile(srcPath, []byte(asmSource), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Assemble
	asm := New()
	err = asm.Assemble(srcPath, objPath)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Read and verify symbols
	obj, err := ReadObjectFile(objPath)
	if err != nil {
		t.Fatalf("ReadObjectFile failed: %v", err)
	}

	// Count global symbols
	globalSymbols := 0
	symbolNames := make(map[string]bool)
	for _, sym := range obj.Symbols {
		symbolNames[sym.Name] = true
		if sym.Extern {
			globalSymbols++
		}
	}

	if !symbolNames["_start"] {
		t.Error("Missing _start symbol")
	}
	if !symbolNames["_helper"] {
		t.Error("Missing _helper symbol")
	}
}

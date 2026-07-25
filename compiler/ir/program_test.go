package ir

import (
	"strings"
	"testing"
)

// stringConst builds a function containing a single string-constant value
// referencing the given pool index, mimicking what the generator emits.
func stringConst(p *Program, fnName, s string) *Value {
	idx := p.AddString(s)
	fn := p.NewFunction(fnName, TypeVoid)
	block := fn.NewBlock(BlockPlain)
	v := block.NewValue(OpConst, TypeString)
	v.AuxInt = int64(idx)
	v.AuxString = s
	return v
}

func TestProgramMerge(t *testing.T) {
	t.Run("remaps string indices onto the destination pool", func(t *testing.T) {
		dst := NewProgram()
		dstVal := stringConst(dst, "dst_fn", "DST_ZERO")

		// The source numbers its own pool from zero, so this value's index
		// collides with dst's unrelated entry at index 0.
		src := NewProgram()
		srcVal := stringConst(src, "src_fn", "SRC_ZERO")
		if srcVal.AuxInt != 0 {
			t.Fatalf("precondition: src index = %d, want 0", srcVal.AuxInt)
		}

		dst.Merge(src)

		if got := dst.GetString(int(dstVal.AuxInt)); got != "DST_ZERO" {
			t.Errorf("dst value resolves to %q, want %q", got, "DST_ZERO")
		}
		if got := dst.GetString(int(srcVal.AuxInt)); got != "SRC_ZERO" {
			t.Errorf("src value resolves to %q, want %q", got, "SRC_ZERO")
		}
		if dstVal.AuxInt == srcVal.AuxInt {
			t.Errorf("distinct literals share pool index %d", dstVal.AuxInt)
		}
	})

	t.Run("deduplicates a literal present in both pools", func(t *testing.T) {
		dst := NewProgram()
		dstVal := stringConst(dst, "dst_fn", "SHARED")

		src := NewProgram()
		srcVal := stringConst(src, "src_fn", "SHARED")

		dst.Merge(src)

		if dstVal.AuxInt != srcVal.AuxInt {
			t.Errorf("shared literal has indices %d and %d, want one entry",
				dstVal.AuxInt, srcVal.AuxInt)
		}
		if len(dst.Strings) != 1 {
			t.Errorf("pool has %d entries, want 1", len(dst.Strings))
		}
		if got := dst.GetString(int(srcVal.AuxInt)); got != "SHARED" {
			t.Errorf("resolves to %q, want %q", got, "SHARED")
		}
	})

	t.Run("carries functions and globals", func(t *testing.T) {
		dst := NewProgram()
		dst.NewFunction("dst_fn", TypeVoid)

		src := NewProgram()
		src.NewFunction("src_fn", TypeVoid)
		src.AddGlobal(&Global{Name: "g", Type: TypeS64})

		dst.Merge(src)

		if dst.FunctionByName("src_fn") == nil {
			t.Error("merged function missing")
		}
		if dst.FunctionByName("dst_fn") == nil {
			t.Error("original function lost")
		}
		if len(dst.Globals) != 1 {
			t.Errorf("globals = %d, want 1", len(dst.Globals))
		}
	})

	// Struct names are never package-mangled, so adopting the source's structs
	// would make two packages that each declare a `Point` collide and fail
	// validation with "duplicate struct name".
	t.Run("does not adopt struct types", func(t *testing.T) {
		dst := NewProgram()
		dst.AddStruct(&StructType{Name: "Point"})

		src := NewProgram()
		src.AddStruct(&StructType{Name: "Point"})

		dst.Merge(src)

		if len(dst.Structs) != 1 {
			t.Errorf("structs = %d, want 1", len(dst.Structs))
		}
		for _, err := range dst.Validate() {
			if strings.Contains(err.Error(), "duplicate struct name") {
				t.Errorf("validation error after merge: %v", err)
			}
		}
	})

	t.Run("leaves non-string constants alone", func(t *testing.T) {
		dst := NewProgram()
		dst.AddString("padding0")
		dst.AddString("padding1")

		src := NewProgram()
		fn := src.NewFunction("src_fn", TypeVoid)
		block := fn.NewBlock(BlockPlain)
		intVal := block.NewValue(OpConst, TypeS64)
		intVal.AuxInt = 1 // an integer value, not a pool index

		dst.Merge(src)

		if intVal.AuxInt != 1 {
			t.Errorf("integer constant rewritten to %d, want 1", intVal.AuxInt)
		}
	})

	t.Run("merging nil is a no-op", func(t *testing.T) {
		dst := NewProgram()
		dst.NewFunction("dst_fn", TypeVoid)
		dst.Merge(nil)
		if len(dst.Functions) != 1 {
			t.Errorf("functions = %d, want 1", len(dst.Functions))
		}
	})
}

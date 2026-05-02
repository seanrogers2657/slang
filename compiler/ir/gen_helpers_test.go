package ir

import (
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

func TestWrapIfNeeded(t *testing.T) {
	// Setup
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}

	t.Run("nil_value_returns_nil", func(t *testing.T) {
		result := g.wrapIfNeeded(nil, TypeS64)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("non_nullable_target_returns_unchanged", func(t *testing.T) {
		val := block.NewValue(OpConst, TypeS64)
		val.AuxInt = 42
		result := g.wrapIfNeeded(val, TypeS64)
		if result != val {
			t.Errorf("expected same value, got different")
		}
	})

	t.Run("nullable_value_to_nullable_target_returns_unchanged", func(t *testing.T) {
		nullableType := &NullableType{Elem: TypeS64}
		val := block.NewValue(OpWrap, nullableType)
		result := g.wrapIfNeeded(val, nullableType)
		if result != val {
			t.Errorf("expected same value, got different")
		}
	})

	t.Run("non_nullable_value_to_nullable_target_wraps", func(t *testing.T) {
		val := block.NewValue(OpConst, TypeS64)
		val.AuxInt = 42
		nullableType := &NullableType{Elem: TypeS64}

		result := g.wrapIfNeeded(val, nullableType)

		if result == val {
			t.Error("expected wrapped value, got original")
		}
		if result.Op != OpWrap {
			t.Errorf("expected OpWrap, got %v", result.Op)
		}
		if len(result.Args) != 1 || result.Args[0] != val {
			t.Error("expected original value as argument")
		}
	})
}

func TestIsNullLiteral(t *testing.T) {
	t.Run("null_literal_returns_true", func(t *testing.T) {
		expr := &semantic.TypedLiteralExpr{
			LitType: ast.LiteralTypeNull,
		}
		if !isNullLiteral(expr) {
			t.Error("expected true for null literal")
		}
	})

	t.Run("integer_literal_returns_false", func(t *testing.T) {
		expr := &semantic.TypedLiteralExpr{
			LitType: ast.LiteralTypeInteger,
			Value:   "42",
		}
		if isNullLiteral(expr) {
			t.Error("expected false for integer literal")
		}
	})

	t.Run("boolean_literal_returns_false", func(t *testing.T) {
		expr := &semantic.TypedLiteralExpr{
			LitType: ast.LiteralTypeBoolean,
			Value:   "true",
		}
		if isNullLiteral(expr) {
			t.Error("expected false for boolean literal")
		}
	})

	t.Run("string_literal_returns_false", func(t *testing.T) {
		expr := &semantic.TypedLiteralExpr{
			LitType: ast.LiteralTypeString,
			Value:   "hello",
		}
		if isNullLiteral(expr) {
			t.Error("expected false for string literal")
		}
	})

	t.Run("identifier_returns_false", func(t *testing.T) {
		expr := &semantic.TypedIdentifierExpr{
			Name: "x",
		}
		if isNullLiteral(expr) {
			t.Error("expected false for identifier")
		}
	})
}

func TestAsNullableType(t *testing.T) {
	t.Run("nullable_type_returns_type", func(t *testing.T) {
		nt := &NullableType{Elem: TypeS64}
		result := asNullableType(nt)
		if result != nt {
			t.Errorf("expected same nullable type, got %v", result)
		}
	})

	t.Run("non_nullable_type_returns_nil", func(t *testing.T) {
		result := asNullableType(TypeS64)
		if result != nil {
			t.Errorf("expected nil for non-nullable type, got %v", result)
		}
	})

	t.Run("ptr_type_returns_nil", func(t *testing.T) {
		pt := &PtrType{Elem: TypeS64}
		result := asNullableType(pt)
		if result != nil {
			t.Errorf("expected nil for ptr type, got %v", result)
		}
	})
}

func TestIsReferenceNullable(t *testing.T) {
	t.Run("nullable_ptr_is_reference", func(t *testing.T) {
		nt := &NullableType{Elem: &PtrType{Elem: TypeS64}}
		if !nt.IsReferenceNullable() {
			t.Error("expected true for nullable ptr")
		}
	})

	t.Run("nullable_struct_is_reference", func(t *testing.T) {
		st := &StructType{Name: "Point"}
		nt := &NullableType{Elem: st}
		if !nt.IsReferenceNullable() {
			t.Error("expected true for nullable struct")
		}
	})

	t.Run("nullable_int_is_not_reference", func(t *testing.T) {
		nt := &NullableType{Elem: TypeS64}
		if nt.IsReferenceNullable() {
			t.Error("expected false for nullable int")
		}
	})

	t.Run("nullable_bool_is_not_reference", func(t *testing.T) {
		nt := &NullableType{Elem: TypeBool}
		if nt.IsReferenceNullable() {
			t.Error("expected false for nullable bool")
		}
	})
}

// TestNullableTypeSize pins the post-refactor invariant that every nullable
// is an 8-byte pointer regardless of its element type. The whole memory-
// model rests on this.
func TestNullableTypeSize(t *testing.T) {
	cases := []struct {
		name string
		nt   *NullableType
	}{
		{"s64?", &NullableType{Elem: TypeS64}},
		{"bool?", &NullableType{Elem: TypeBool}},
		{"s128?", &NullableType{Elem: TypeS128}},
		{"*Point?", &NullableType{Elem: &PtrType{Elem: &StructType{Name: "Point"}}}},
		{"Point?", &NullableType{Elem: &StructType{Name: "Point"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.nt.Size(); got != 8 {
				t.Errorf("Size() = %d, want 8", got)
			}
		})
	}
}

// TestNullableValueInner separates nullables of value-type elements (s64?,
// bool?) from nullables wrapping pointers/structs. The value-element case
// requires heap allocation at wrap time; the pointer-element case carries
// the pointer through directly.
func TestNullableValueInner(t *testing.T) {
	t.Run("value_nullable_returns_inner", func(t *testing.T) {
		got := nullableValueInner(semantic.NullableType{InnerType: semantic.S64Type{}})
		if got == nil {
			t.Fatal("expected inner type, got nil")
		}
		if _, ok := got.(semantic.S64Type); !ok {
			t.Errorf("expected S64Type, got %T", got)
		}
	})
	t.Run("owned_pointer_nullable_returns_nil", func(t *testing.T) {
		nt := semantic.NullableType{InnerType: &semantic.OwnedPointerType{ElementType: semantic.S64Type{}}}
		if got := nullableValueInner(nt); got != nil {
			t.Errorf("expected nil for owned-pointer nullable, got %v", got)
		}
	})
	t.Run("struct_nullable_returns_nil", func(t *testing.T) {
		nt := semantic.NullableType{InnerType: semantic.StructType{Name: "Point"}}
		if got := nullableValueInner(nt); got != nil {
			t.Errorf("expected nil for struct nullable, got %v", got)
		}
	})
	t.Run("non_nullable_returns_nil", func(t *testing.T) {
		if got := nullableValueInner(semantic.S64Type{}); got != nil {
			t.Errorf("expected nil for non-nullable, got %v", got)
		}
	})
}

// TestVarOwnsHeapVsFieldOwnsHeap pins the distinction between variable-
// binding ownership and struct-field embedding. Struct/class/array types
// own heap when bound to a variable (their literal allocates) but are
// embedded inline as struct fields, so reassigning a field doesn't need
// a free of the old value.
func TestVarOwnsHeapVsFieldOwnsHeap(t *testing.T) {
	cases := []struct {
		name        string
		semType     semantic.Type
		fieldOwns   bool
		varOwns     bool
		isHeapValue bool
	}{
		{
			name:      "owned_pointer",
			semType:   &semantic.OwnedPointerType{ElementType: semantic.S64Type{}},
			fieldOwns: true,
			varOwns:   true,
		},
		{
			name:      "value_nullable",
			semType:   semantic.NullableType{InnerType: semantic.S64Type{}},
			fieldOwns: true,
			varOwns:   true,
		},
		{
			name:      "owned_pointer_nullable",
			semType:   semantic.NullableType{InnerType: &semantic.OwnedPointerType{ElementType: semantic.S64Type{}}},
			fieldOwns: true,
			varOwns:   true,
		},
		{
			name:        "struct_value",
			semType:     semantic.StructType{Name: "Point"},
			fieldOwns:   false,
			varOwns:     true,
			isHeapValue: true,
		},
		{
			name:        "class_value",
			semType:     semantic.ClassType{Name: "Counter"},
			fieldOwns:   false,
			varOwns:     true,
			isHeapValue: true,
		},
		{
			name:        "array_value",
			semType:     &semantic.ArrayType{ElementType: semantic.S64Type{}, Size: 3},
			fieldOwns:   false,
			varOwns:     true,
			isHeapValue: true,
		},
		{
			name:    "primitive_int",
			semType: semantic.S64Type{},
		},
		{
			name:    "primitive_bool",
			semType: semantic.BooleanType{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := fieldOwnsHeap(c.semType); got != c.fieldOwns {
				t.Errorf("fieldOwnsHeap = %v, want %v", got, c.fieldOwns)
			}
			if got := varOwnsHeap(c.semType); got != c.varOwns {
				t.Errorf("varOwnsHeap = %v, want %v", got, c.varOwns)
			}
			if got := isHeapValueType(c.semType); got != c.isHeapValue {
				t.Errorf("isHeapValueType = %v, want %v", got, c.isHeapValue)
			}
		})
	}
}

// TestArgTransfersOwnership pins the move-vs-borrow rule at call sites.
// Owned-pointer args going to owned-pointer params transfer; everything
// else borrows. Value-type nullables always borrow — they have copy-style
// semantics across function boundaries.
func TestArgTransfersOwnership(t *testing.T) {
	ownedS64 := &semantic.OwnedPointerType{ElementType: semantic.S64Type{}}
	refS64 := &semantic.RefPointerType{ElementType: semantic.S64Type{}}
	mutRefS64 := &semantic.MutRefPointerType{ElementType: semantic.S64Type{}}
	nullableValueS64 := semantic.NullableType{InnerType: semantic.S64Type{}}
	nullableOwnedS64 := semantic.NullableType{InnerType: ownedS64}

	cases := []struct {
		name      string
		argType   semantic.Type
		paramType semantic.Type
		want      bool
	}{
		{"owned_to_owned_moves", ownedS64, ownedS64, true},
		{"owned_to_immutable_borrow_does_not_move", ownedS64, refS64, false},
		{"owned_to_mutable_borrow_does_not_move", ownedS64, mutRefS64, false},
		{"owned_to_nullable_owned_moves", ownedS64, nullableOwnedS64, true},
		{"value_nullable_to_value_nullable_borrows", nullableValueS64, nullableValueS64, false},
		{"primitive_does_not_move", semantic.S64Type{}, semantic.S64Type{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := argTransfersOwnership(c.argType, c.paramType); got != c.want {
				t.Errorf("argTransfersOwnership = %v, want %v", got, c.want)
			}
		})
	}
}

// TestShouldCopyOnReturn pins the copy-on-extract rule. Container reads
// (arr[i], obj.field) of value-type nullables must be copied; everything
// else either transfers via markMoved or doesn't own heap to begin with.
func TestShouldCopyOnReturn(t *testing.T) {
	pos := ast.Position{Line: 1, Column: 1}
	nullableS64 := semantic.NullableType{InnerType: semantic.S64Type{}}

	cases := []struct {
		name string
		expr semantic.TypedExpression
		want bool
	}{
		{
			name: "index_of_nullable_copies",
			expr: &semantic.TypedIndexExpr{Type: nullableS64},
			want: true,
		},
		{
			name: "field_of_nullable_copies",
			expr: &semantic.TypedFieldAccessExpr{Type: nullableS64},
			want: true,
		},
		{
			name: "identifier_of_nullable_does_not_copy",
			expr: &semantic.TypedIdentifierExpr{Type: nullableS64, StartPos: pos, EndPos: pos},
			want: false,
		},
		{
			name: "index_of_primitive_does_not_copy",
			expr: &semantic.TypedIndexExpr{Type: semantic.S64Type{}},
			want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldCopyOnReturn(c.expr); got != c.want {
				t.Errorf("shouldCopyOnReturn = %v, want %v", got, c.want)
			}
		})
	}
}

func TestGenerateTypedValue(t *testing.T) {
	// This test uses the full compiler pipeline
	t.Run("generates_null_for_null_literal", func(t *testing.T) {
		prog := compileToIR(t, `
			main = () {
				val x: s64? = null
			}
		`)
		fn := prog.Main()
		if fn == nil {
			t.Fatal("expected main function")
		}
		// Check that WrapNull was generated
		found := false
		for _, block := range fn.Blocks {
			for _, v := range block.Values {
				if v.Op == OpWrapNull {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("expected WrapNull operation for null literal")
		}
	})

	t.Run("wraps_non_nullable_value_for_nullable_target", func(t *testing.T) {
		prog := compileToIR(t, `
			main = () {
				val x: s64? = 42
			}
		`)
		fn := prog.Main()
		if fn == nil {
			t.Fatal("expected main function")
		}
		// Check that Wrap was generated
		found := false
		for _, block := range fn.Blocks {
			for _, v := range block.Values {
				if v.Op == OpWrap {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("expected Wrap operation for non-nullable value in nullable target")
		}
	})
}

// ============================================================================
// Null Check Blocks Tests
// ============================================================================

func TestCreateNullCheckBlocks(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	fn.NewBlock(BlockPlain) // entry block
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: fn.Blocks[0],
	}

	blocks := g.createNullCheckBlocks()

	t.Run("creates_three_blocks", func(t *testing.T) {
		if blocks.nullBlock == nil {
			t.Error("nullBlock is nil")
		}
		if blocks.notNullBlock == nil {
			t.Error("notNullBlock is nil")
		}
		if blocks.mergeBlock == nil {
			t.Error("mergeBlock is nil")
		}
	})

	t.Run("blocks_are_distinct", func(t *testing.T) {
		if blocks.nullBlock == blocks.notNullBlock {
			t.Error("nullBlock and notNullBlock are the same")
		}
		if blocks.nullBlock == blocks.mergeBlock {
			t.Error("nullBlock and mergeBlock are the same")
		}
		if blocks.notNullBlock == blocks.mergeBlock {
			t.Error("notNullBlock and mergeBlock are the same")
		}
	})

	t.Run("blocks_have_correct_kind", func(t *testing.T) {
		if blocks.nullBlock.Kind != BlockPlain {
			t.Errorf("nullBlock kind = %v, want BlockPlain", blocks.nullBlock.Kind)
		}
		if blocks.notNullBlock.Kind != BlockPlain {
			t.Errorf("notNullBlock kind = %v, want BlockPlain", blocks.notNullBlock.Kind)
		}
		if blocks.mergeBlock.Kind != BlockPlain {
			t.Errorf("mergeBlock kind = %v, want BlockPlain", blocks.mergeBlock.Kind)
		}
	})
}

func TestEmitNullCheck(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	entryBlock := fn.NewBlock(BlockPlain)
	ssa := NewSSABuilder()
	ssa.SetFunction(fn)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: entryBlock,
		ssa:   ssa,
	}

	// Create a test value
	testVal := entryBlock.NewValue(OpConst, &NullableType{Elem: TypeS64})
	testVal.AuxInt = 42

	blocks := g.createNullCheckBlocks()
	g.emitNullCheck(testVal, blocks)

	t.Run("creates_isNull_check", func(t *testing.T) {
		found := false
		for _, v := range entryBlock.Values {
			if v.Op == OpIsNull {
				found = true
				if len(v.Args) != 1 || v.Args[0] != testVal {
					t.Error("IsNull should have testVal as argument")
				}
			}
		}
		if !found {
			t.Error("expected IsNull operation in entry block")
		}
	})

	t.Run("sets_block_to_if", func(t *testing.T) {
		if entryBlock.Kind != BlockIf {
			t.Errorf("entry block kind = %v, want BlockIf", entryBlock.Kind)
		}
	})

	t.Run("sets_control_value", func(t *testing.T) {
		if entryBlock.Control == nil {
			t.Error("entry block control is nil")
		}
		if entryBlock.Control.Op != OpIsNull {
			t.Errorf("control op = %v, want OpIsNull", entryBlock.Control.Op)
		}
	})

	t.Run("adds_successors", func(t *testing.T) {
		if len(entryBlock.Succs) != 2 {
			t.Fatalf("expected 2 successors, got %d", len(entryBlock.Succs))
		}
		if entryBlock.Succs[0] != blocks.nullBlock {
			t.Error("first successor should be nullBlock")
		}
		if entryBlock.Succs[1] != blocks.notNullBlock {
			t.Error("second successor should be notNullBlock")
		}
	})

	t.Run("seals_path_blocks", func(t *testing.T) {
		if !blocks.nullBlock.Sealed {
			t.Error("nullBlock should be sealed")
		}
		if !blocks.notNullBlock.Sealed {
			t.Error("notNullBlock should be sealed")
		}
	})
}

func TestMergeNullCheckResults(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	fn.NewBlock(BlockPlain) // entry block
	ssa := NewSSABuilder()
	ssa.SetFunction(fn)
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: fn.Blocks[0],
		ssa:   ssa,
	}

	// Setup blocks
	blocks := g.createNullCheckBlocks()
	resultType := &NullableType{Elem: TypeS64}

	// Create results
	g.block = blocks.nullBlock
	nullResult := blocks.nullBlock.NewValue(OpWrapNull, resultType)

	g.block = blocks.notNullBlock
	notNullResult := blocks.notNullBlock.NewValue(OpWrap, resultType)

	// Call merge
	phi, err := g.mergeNullCheckResults(blocks, resultType, nullResult, notNullResult)

	t.Run("returns_no_error", func(t *testing.T) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns_phi_node", func(t *testing.T) {
		if phi == nil {
			t.Fatal("expected phi node, got nil")
		}
		if phi.Op != OpPhi {
			t.Errorf("expected OpPhi, got %v", phi.Op)
		}
	})

	t.Run("phi_has_correct_type", func(t *testing.T) {
		if phi.Type != resultType {
			t.Errorf("phi type = %v, want %v", phi.Type, resultType)
		}
	})

	t.Run("phi_has_two_args", func(t *testing.T) {
		if len(phi.PhiArgs) != 2 {
			t.Fatalf("expected 2 phi args, got %d", len(phi.PhiArgs))
		}
	})

	t.Run("phi_args_have_correct_sources", func(t *testing.T) {
		if phi.PhiArgs[0].From != blocks.nullBlock {
			t.Error("first phi arg should be from nullBlock")
		}
		if phi.PhiArgs[0].Value != nullResult {
			t.Error("first phi arg should be nullResult")
		}
		if phi.PhiArgs[1].From != blocks.notNullBlock {
			t.Error("second phi arg should be from notNullBlock")
		}
		if phi.PhiArgs[1].Value != notNullResult {
			t.Error("second phi arg should be notNullResult")
		}
	})

	t.Run("sets_current_block_to_merge", func(t *testing.T) {
		if g.block != blocks.mergeBlock {
			t.Error("current block should be mergeBlock")
		}
	})

	t.Run("seals_merge_block", func(t *testing.T) {
		if !blocks.mergeBlock.Sealed {
			t.Error("mergeBlock should be sealed")
		}
	})
}

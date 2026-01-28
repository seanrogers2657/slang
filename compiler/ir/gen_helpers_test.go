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
		if !isReferenceNullable(nt) {
			t.Error("expected true for nullable ptr")
		}
	})

	t.Run("nullable_struct_is_reference", func(t *testing.T) {
		st := &StructType{Name: "Point"}
		nt := &NullableType{Elem: st}
		if !isReferenceNullable(nt) {
			t.Error("expected true for nullable struct")
		}
	})

	t.Run("nullable_int_is_not_reference", func(t *testing.T) {
		nt := &NullableType{Elem: TypeS64}
		if isReferenceNullable(nt) {
			t.Error("expected false for nullable int")
		}
	})

	t.Run("nullable_bool_is_not_reference", func(t *testing.T) {
		nt := &NullableType{Elem: TypeBool}
		if isReferenceNullable(nt) {
			t.Error("expected false for nullable bool")
		}
	})

	t.Run("non_nullable_returns_false", func(t *testing.T) {
		if isReferenceNullable(TypeS64) {
			t.Error("expected false for non-nullable type")
		}
	})
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

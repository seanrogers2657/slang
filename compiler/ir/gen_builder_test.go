package ir

import (
	"testing"
)

func TestValueBuilder(t *testing.T) {
	// Create a minimal function context for testing
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)

	// Create a generator to get a builder
	g := &Generator{
		prog:  prog,
		fn:    fn,
		block: block,
	}
	b := g.builder()

	t.Run("ConstInt", func(t *testing.T) {
		v := b.ConstInt(TypeS64, 42)
		if v.Op != OpConst {
			t.Errorf("expected OpConst, got %v", v.Op)
		}
		if v.AuxInt != 42 {
			t.Errorf("expected AuxInt=42, got %d", v.AuxInt)
		}
		if !v.Type.Equal(TypeS64) {
			t.Errorf("expected s64 type, got %v", v.Type)
		}
	})

	t.Run("ConstBool_true", func(t *testing.T) {
		v := b.ConstBool(true)
		if v.Op != OpConst {
			t.Errorf("expected OpConst, got %v", v.Op)
		}
		if v.AuxInt != 1 {
			t.Errorf("expected AuxInt=1 for true, got %d", v.AuxInt)
		}
		if !v.Type.Equal(TypeBool) {
			t.Errorf("expected bool type, got %v", v.Type)
		}
	})

	t.Run("ConstBool_false", func(t *testing.T) {
		v := b.ConstBool(false)
		if v.AuxInt != 0 {
			t.Errorf("expected AuxInt=0 for false, got %d", v.AuxInt)
		}
	})

	t.Run("ConstString", func(t *testing.T) {
		v := b.ConstString(prog, "hello")
		if v.Op != OpConst {
			t.Errorf("expected OpConst, got %v", v.Op)
		}
		if v.AuxString != "hello" {
			t.Errorf("expected AuxString='hello', got %q", v.AuxString)
		}
		if !v.Type.Equal(TypeString) {
			t.Errorf("expected string type, got %v", v.Type)
		}
		// Check string was added to program
		if len(prog.Strings) == 0 || prog.Strings[0] != "hello" {
			t.Errorf("string not added to program")
		}
	})

	t.Run("Alloc", func(t *testing.T) {
		elemType := TypeS64
		v := b.Alloc(elemType, 8)
		if v.Op != OpAlloc {
			t.Errorf("expected OpAlloc, got %v", v.Op)
		}
		if v.AuxInt != 8 {
			t.Errorf("expected size=8, got %d", v.AuxInt)
		}
		ptrType, ok := v.Type.(*PtrType)
		if !ok {
			t.Fatalf("expected PtrType, got %T", v.Type)
		}
		if !ptrType.Elem.Equal(elemType) {
			t.Errorf("expected elem type s64, got %v", ptrType.Elem)
		}
	})

	t.Run("Load", func(t *testing.T) {
		ptr := b.Alloc(TypeS64, 8)
		v := b.Load(ptr, TypeS64)
		if v.Op != OpLoad {
			t.Errorf("expected OpLoad, got %v", v.Op)
		}
		if len(v.Args) != 1 || v.Args[0] != ptr {
			t.Errorf("expected ptr as argument")
		}
	})

	t.Run("Store", func(t *testing.T) {
		ptr := b.Alloc(TypeS64, 8)
		val := b.ConstInt(TypeS64, 100)
		v := b.Store(ptr, val)
		if v.Op != OpStore {
			t.Errorf("expected OpStore, got %v", v.Op)
		}
		if len(v.Args) != 2 {
			t.Fatalf("expected 2 args, got %d", len(v.Args))
		}
		if v.Args[0] != ptr {
			t.Errorf("expected ptr as first arg")
		}
		if v.Args[1] != val {
			t.Errorf("expected val as second arg")
		}
	})

	t.Run("FieldPtr", func(t *testing.T) {
		structType := &StructType{Name: "Point", Fields: []StructField{
			{Name: "x", Type: TypeS64, Offset: 0},
			{Name: "y", Type: TypeS64, Offset: 8},
		}}
		ptr := b.Alloc(structType, 16)
		fieldPtr := b.FieldPtr(ptr, TypeS64, 8)
		if fieldPtr.Op != OpFieldPtr {
			t.Errorf("expected OpFieldPtr, got %v", fieldPtr.Op)
		}
		if fieldPtr.AuxInt != 8 {
			t.Errorf("expected offset=8, got %d", fieldPtr.AuxInt)
		}
	})

	t.Run("IndexPtr", func(t *testing.T) {
		arrayType := &ArrayType{Elem: TypeS64, Len: 10}
		arr := b.Alloc(arrayType, 80)
		idx := b.ConstInt(TypeS64, 5)
		elemPtr := b.IndexPtr(arr, idx, TypeS64)
		if elemPtr.Op != OpIndexPtr {
			t.Errorf("expected OpIndexPtr, got %v", elemPtr.Op)
		}
		if len(elemPtr.Args) != 2 {
			t.Fatalf("expected 2 args, got %d", len(elemPtr.Args))
		}
	})

	t.Run("MemCopy", func(t *testing.T) {
		dest := b.Alloc(TypeS64, 8)
		src := b.Alloc(TypeS64, 8)
		v := b.MemCopy(dest, src, 8)
		if v.Op != OpMemCopy {
			t.Errorf("expected OpMemCopy, got %v", v.Op)
		}
		if v.AuxInt != 8 {
			t.Errorf("expected size=8, got %d", v.AuxInt)
		}
	})

	t.Run("Binary_Add", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 10)
		right := b.ConstInt(TypeS64, 20)
		v := b.Add(TypeS64, left, right)
		if v.Op != OpAdd {
			t.Errorf("expected OpAdd, got %v", v.Op)
		}
	})

	t.Run("Binary_Sub", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 30)
		right := b.ConstInt(TypeS64, 10)
		v := b.Sub(TypeS64, left, right)
		if v.Op != OpSub {
			t.Errorf("expected OpSub, got %v", v.Op)
		}
	})

	t.Run("Binary_Mul", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 5)
		right := b.ConstInt(TypeS64, 6)
		v := b.Mul(TypeS64, left, right)
		if v.Op != OpMul {
			t.Errorf("expected OpMul, got %v", v.Op)
		}
	})

	t.Run("Binary_Div", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 100)
		right := b.ConstInt(TypeS64, 10)
		v := b.Div(TypeS64, left, right)
		if v.Op != OpDiv {
			t.Errorf("expected OpDiv, got %v", v.Op)
		}
	})

	t.Run("Binary_Mod", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 17)
		right := b.ConstInt(TypeS64, 5)
		v := b.Mod(TypeS64, left, right)
		if v.Op != OpMod {
			t.Errorf("expected OpMod, got %v", v.Op)
		}
	})

	t.Run("Comparison_Eq", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 10)
		right := b.ConstInt(TypeS64, 10)
		v := b.Eq(left, right)
		if v.Op != OpEq {
			t.Errorf("expected OpEq, got %v", v.Op)
		}
		if !v.Type.Equal(TypeBool) {
			t.Errorf("expected bool result type")
		}
	})

	t.Run("Comparison_Ne", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 10)
		right := b.ConstInt(TypeS64, 20)
		v := b.Ne(left, right)
		if v.Op != OpNe {
			t.Errorf("expected OpNe, got %v", v.Op)
		}
	})

	t.Run("Comparison_Lt", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 5)
		right := b.ConstInt(TypeS64, 10)
		v := b.Lt(left, right)
		if v.Op != OpLt {
			t.Errorf("expected OpLt, got %v", v.Op)
		}
	})

	t.Run("Comparison_Le", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 10)
		right := b.ConstInt(TypeS64, 10)
		v := b.Le(left, right)
		if v.Op != OpLe {
			t.Errorf("expected OpLe, got %v", v.Op)
		}
	})

	t.Run("Comparison_Gt", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 20)
		right := b.ConstInt(TypeS64, 10)
		v := b.Gt(left, right)
		if v.Op != OpGt {
			t.Errorf("expected OpGt, got %v", v.Op)
		}
	})

	t.Run("Comparison_Ge", func(t *testing.T) {
		left := b.ConstInt(TypeS64, 10)
		right := b.ConstInt(TypeS64, 5)
		v := b.Ge(left, right)
		if v.Op != OpGe {
			t.Errorf("expected OpGe, got %v", v.Op)
		}
	})

	t.Run("Not", func(t *testing.T) {
		operand := b.ConstBool(true)
		v := b.Not(operand)
		if v.Op != OpNot {
			t.Errorf("expected OpNot, got %v", v.Op)
		}
		if !v.Type.Equal(TypeBool) {
			t.Errorf("expected bool result type")
		}
	})

	t.Run("Neg", func(t *testing.T) {
		operand := b.ConstInt(TypeS64, 42)
		v := b.Neg(TypeS64, operand)
		if v.Op != OpNeg {
			t.Errorf("expected OpNeg, got %v", v.Op)
		}
	})

	t.Run("IsNull", func(t *testing.T) {
		nullableType := &NullableType{Elem: TypeS64}
		val := block.NewValue(OpWrapNull, nullableType)
		v := b.IsNull(val)
		if v.Op != OpIsNull {
			t.Errorf("expected OpIsNull, got %v", v.Op)
		}
		if !v.Type.Equal(TypeBool) {
			t.Errorf("expected bool result type")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		nullableType := &NullableType{Elem: TypeS64}
		val := block.NewValue(OpWrap, nullableType)
		v := b.Unwrap(val, TypeS64)
		if v.Op != OpUnwrap {
			t.Errorf("expected OpUnwrap, got %v", v.Op)
		}
	})

	t.Run("Wrap", func(t *testing.T) {
		val := b.ConstInt(TypeS64, 42)
		nullableType := &NullableType{Elem: TypeS64}
		v := b.Wrap(val, nullableType)
		if v.Op != OpWrap {
			t.Errorf("expected OpWrap, got %v", v.Op)
		}
	})

	t.Run("WrapNull", func(t *testing.T) {
		nullableType := &NullableType{Elem: TypeS64}
		v := b.WrapNull(nullableType)
		if v.Op != OpWrapNull {
			t.Errorf("expected OpWrapNull, got %v", v.Op)
		}
	})

	t.Run("Call", func(t *testing.T) {
		arg1 := b.ConstInt(TypeS64, 10)
		arg2 := b.ConstInt(TypeS64, 20)
		v := b.Call("add", TypeS64, arg1, arg2)
		if v.Op != OpCall {
			t.Errorf("expected OpCall, got %v", v.Op)
		}
		if v.AuxString != "add" {
			t.Errorf("expected function name 'add', got %q", v.AuxString)
		}
		if len(v.Args) != 2 {
			t.Errorf("expected 2 args, got %d", len(v.Args))
		}
	})

	t.Run("Return", func(t *testing.T) {
		val := b.ConstInt(TypeS64, 42)
		v := b.Return(val)
		if v.Op != OpReturn {
			t.Errorf("expected OpReturn, got %v", v.Op)
		}
		if len(v.Args) != 1 || v.Args[0] != val {
			t.Errorf("expected val as argument")
		}
	})

	t.Run("Return_void", func(t *testing.T) {
		v := b.Return(nil)
		if v.Op != OpReturn {
			t.Errorf("expected OpReturn, got %v", v.Op)
		}
		if len(v.Args) != 0 {
			t.Errorf("expected no args for void return, got %d", len(v.Args))
		}
	})

	t.Run("Phi", func(t *testing.T) {
		v := b.Phi(TypeS64)
		if v.Op != OpPhi {
			t.Errorf("expected OpPhi, got %v", v.Op)
		}
	})

	t.Run("Copy", func(t *testing.T) {
		ptrType := &PtrType{Elem: TypeS64}
		ptr := b.Alloc(TypeS64, 8)
		ptr.Type = ptrType
		v := b.Copy(ptr)
		if v.Op != OpCopy {
			t.Errorf("expected OpCopy, got %v", v.Op)
		}
	})
}

package ir

import "testing"

func TestSSABuilderBasicReadWrite(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	block.Sealed = true
	ssa.SetFunction(fn)

	// Write a value
	val := block.NewValue(OpConst, TypeS64)
	val.AuxInt = 42
	ssa.WriteVariable("x", block, val)

	// Read it back
	got := ssa.ReadVariable("x", block)
	if got != val {
		t.Errorf("ReadVariable returned %v, want %v", got, val)
	}
}

func TestSSABuilderSinglePredecessor(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	ssa.SetFunction(fn)

	block1 := fn.NewBlock(BlockPlain)
	block2 := fn.NewBlock(BlockPlain)
	block1.AddSucc(block2)

	block1.Sealed = true
	block2.Sealed = true

	// Write in block1
	val := block1.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("x", block1, val)

	// Read in block2 - should find definition from block1, no phi needed
	got := ssa.ReadVariable("x", block2)
	if got != val {
		t.Errorf("ReadVariable returned %v, want %v", got, val)
	}

	// Verify no phi was created
	for _, v := range block2.Values {
		if v.Op == OpPhi {
			t.Error("Unexpected phi node in single-predecessor block")
		}
	}
}

func TestSSABuilderMultiplePredecessors(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	ssa.SetFunction(fn)

	entry := fn.NewBlock(BlockPlain)
	left := fn.NewBlock(BlockPlain)
	right := fn.NewBlock(BlockPlain)
	merge := fn.NewBlock(BlockPlain)

	entry.AddSucc(left)
	entry.AddSucc(right)
	left.AddSucc(merge)
	right.AddSucc(merge)

	entry.Sealed = true
	left.Sealed = true
	right.Sealed = true
	merge.Sealed = true

	// Write different values in left and right
	valLeft := left.NewValue(OpConst, TypeS64)
	valLeft.AuxInt = 1
	ssa.WriteVariable("x", left, valLeft)

	valRight := right.NewValue(OpConst, TypeS64)
	valRight.AuxInt = 2
	ssa.WriteVariable("x", right, valRight)

	// Read in merge - should create phi
	got := ssa.ReadVariable("x", merge)
	if got.Op != OpPhi {
		t.Errorf("Expected phi node, got %v", got.Op)
	}
	if len(got.PhiArgs) != 2 {
		t.Errorf("Phi should have 2 args, got %d", len(got.PhiArgs))
	}
}

func TestSSABuilderTrivialPhiRemoval(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	ssa.SetFunction(fn)

	entry := fn.NewBlock(BlockPlain)
	left := fn.NewBlock(BlockPlain)
	right := fn.NewBlock(BlockPlain)
	merge := fn.NewBlock(BlockPlain)

	entry.AddSucc(left)
	entry.AddSucc(right)
	left.AddSucc(merge)
	right.AddSucc(merge)

	entry.Sealed = true
	left.Sealed = true
	right.Sealed = true
	merge.Sealed = true

	// Write SAME value in both branches
	val := entry.NewValue(OpConst, TypeS64)
	val.AuxInt = 42
	ssa.WriteVariable("x", left, val)
	ssa.WriteVariable("x", right, val)

	// Read in merge - phi should be removed as trivial
	got := ssa.ReadVariable("x", merge)
	if got != val {
		t.Errorf("Trivial phi should be removed, got %v (expected %v)", got, val)
	}
}

func TestSSABuilderIncompletePhi(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	ssa.SetFunction(fn)

	// Create a loop structure
	entry := fn.NewBlock(BlockPlain)
	loop := fn.NewBlock(BlockPlain)

	entry.AddSucc(loop)
	loop.AddSucc(loop) // Back edge

	entry.Sealed = true
	// loop is NOT sealed yet

	// Write in entry
	initVal := entry.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("i", entry, initVal)

	// Read in loop before sealing - should create incomplete phi
	got := ssa.ReadVariable("i", loop)
	if got.Op != OpPhi {
		t.Errorf("Expected incomplete phi, got %v", got.Op)
	}

	// Write updated value in loop
	newVal := loop.NewValue(OpAdd, TypeS64)
	ssa.WriteVariable("i", loop, newVal)

	// Now seal - phi should get its operands
	ssa.SealBlock(loop)

	// Verify loop is sealed
	if !loop.Sealed {
		t.Error("Loop block should be sealed")
	}
}

func TestSSABuilderReset(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	block.Sealed = true
	ssa.SetFunction(fn)

	val := block.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("x", block, val)

	// Verify variable exists
	if !ssa.HasDefinition("x") {
		t.Error("Variable should be defined before reset")
	}

	// Reset
	ssa.Reset()

	// Should not find the variable
	if ssa.HasDefinition("x") {
		t.Error("After reset, variable should not be defined")
	}
}

func TestSSABuilderHasDefinition(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	block.Sealed = true
	ssa.SetFunction(fn)

	// Initially no definitions
	if ssa.HasDefinition("x") {
		t.Error("Variable should not be defined initially")
	}

	// Write a value
	val := block.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("x", block, val)

	// Now it should be defined
	if !ssa.HasDefinition("x") {
		t.Error("Variable should be defined after write")
	}

	// Other variables still undefined
	if ssa.HasDefinition("y") {
		t.Error("Unwritten variable should not be defined")
	}
}

func TestSSABuilderDefinedVariables(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	block.Sealed = true
	ssa.SetFunction(fn)

	// Initially empty
	if len(ssa.DefinedVariables()) != 0 {
		t.Error("Should have no defined variables initially")
	}

	// Write some variables
	val := block.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("x", block, val)
	ssa.WriteVariable("y", block, val)
	ssa.WriteVariable("z", block, val)

	// Check defined variables
	defined := ssa.DefinedVariables()
	if len(defined) != 3 {
		t.Errorf("Expected 3 defined variables, got %d", len(defined))
	}

	// Check all names are present
	names := make(map[string]bool)
	for _, name := range defined {
		names[name] = true
	}
	for _, expected := range []string{"x", "y", "z"} {
		if !names[expected] {
			t.Errorf("Expected variable %q in defined list", expected)
		}
	}
}

func TestSSABuilderEntryBlockNoDefinition(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)
	block.Sealed = true
	ssa.SetFunction(fn)

	// Read undefined variable in entry block (no predecessors)
	got := ssa.ReadVariable("undefined", block)
	if got != nil {
		t.Errorf("Expected nil for undefined variable in entry block, got %v", got)
	}
}

func TestSSABuilderPhiTypeInference(t *testing.T) {
	ssa := NewSSABuilder()
	fn := NewProgram().NewFunction("test", TypeVoid)
	ssa.SetFunction(fn)

	entry := fn.NewBlock(BlockPlain)
	left := fn.NewBlock(BlockPlain)
	right := fn.NewBlock(BlockPlain)
	merge := fn.NewBlock(BlockPlain)

	entry.AddSucc(left)
	entry.AddSucc(right)
	left.AddSucc(merge)
	right.AddSucc(merge)

	entry.Sealed = true
	left.Sealed = true
	right.Sealed = true
	merge.Sealed = true

	// Write s64 values in both branches
	valLeft := left.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("x", left, valLeft)

	valRight := right.NewValue(OpConst, TypeS64)
	ssa.WriteVariable("x", right, valRight)

	// Read in merge - phi should have correct type
	phi := ssa.ReadVariable("x", merge)
	if phi.Op != OpPhi {
		t.Fatalf("Expected phi node, got %v", phi.Op)
	}
	if phi.Type != TypeS64 {
		t.Errorf("Phi type should be s64, got %v", phi.Type)
	}
}

func TestFindUniquePhi(t *testing.T) {
	prog := NewProgram()
	fn := prog.NewFunction("test", TypeVoid)
	block := fn.NewBlock(BlockPlain)

	t.Run("single unique value", func(t *testing.T) {
		phi := block.NewValue(OpPhi, TypeS64)
		val := block.NewValue(OpConst, TypeS64)

		phi.PhiArgs = []*PhiArg{
			{Value: val},
			{Value: val},
		}

		result := findUniquePhi(phi)
		if result != val {
			t.Errorf("Expected unique value, got %v", result)
		}
	})

	t.Run("multiple unique values", func(t *testing.T) {
		phi := block.NewValue(OpPhi, TypeS64)
		val1 := block.NewValue(OpConst, TypeS64)
		val2 := block.NewValue(OpConst, TypeS64)

		phi.PhiArgs = []*PhiArg{
			{Value: val1},
			{Value: val2},
		}

		result := findUniquePhi(phi)
		if result != nil {
			t.Errorf("Expected nil for multiple values, got %v", result)
		}
	})

	t.Run("self reference ignored", func(t *testing.T) {
		phi := block.NewValue(OpPhi, TypeS64)
		val := block.NewValue(OpConst, TypeS64)

		phi.PhiArgs = []*PhiArg{
			{Value: val},
			{Value: phi}, // Self-reference
		}

		result := findUniquePhi(phi)
		if result != val {
			t.Errorf("Expected unique value ignoring self, got %v", result)
		}
	})

	t.Run("empty phi", func(t *testing.T) {
		phi := block.NewValue(OpPhi, TypeS64)
		phi.PhiArgs = []*PhiArg{}

		result := findUniquePhi(phi)
		if result != nil {
			t.Errorf("Expected nil for empty phi, got %v", result)
		}
	})
}

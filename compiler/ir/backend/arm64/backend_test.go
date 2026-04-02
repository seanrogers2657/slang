package arm64

import (
	"fmt"
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/compiler/ir"
	"github.com/seanrogers2657/slang/compiler/ir/backend"
	"github.com/seanrogers2657/slang/compiler/lexer"
	"github.com/seanrogers2657/slang/compiler/parser"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// compileToIR compiles source code to IR.
func compileToIR(t *testing.T, src string) *ir.Program {
	t.Helper()

	// Lexer
	l := lexer.NewLexer([]byte(src))
	l.Parse()
	if len(l.Errors) > 0 {
		t.Fatalf("lexer errors: %v", l.Errors)
	}

	// Parser
	p := parser.NewParser(l.Tokens)
	prog := p.Parse()
	if len(p.Errors) > 0 {
		t.Fatalf("parser errors: %v", p.Errors)
	}

	// Semantic analysis
	analyzer := semantic.NewAnalyzer("<test>")
	compErrs, typed := analyzer.Analyze(prog)
	if len(compErrs) > 0 {
		t.Fatalf("semantic errors: %v", compErrs)
	}

	// IR generation
	irProg, err := ir.Generate(typed)
	if err != nil {
		t.Fatalf("IR generation error: %v", err)
	}

	return irProg
}

func TestBackendSimple(t *testing.T) {
	src := `main = () {
		val x = 42
		exit(x)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Basic checks
	if !strings.Contains(asm, ".global _main") {
		t.Error("Expected .global _main")
	}
	if !strings.Contains(asm, "_main:") {
		t.Error("Expected _main: label")
	}
	if !strings.Contains(asm, "stp x29, x30") {
		t.Error("Expected function prologue")
	}
}

func TestBackendArithmetic(t *testing.T) {
	src := `main = () {
		val a = 10
		val b = 3
		val sum = a + b
		val diff = a - b
		val prod = a * b
		val quot = a / b
		val mod = a % b
		exit(sum)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for arithmetic instructions
	if !strings.Contains(asm, "add") {
		t.Error("Expected add instruction")
	}
	if !strings.Contains(asm, "sub") {
		t.Error("Expected sub instruction")
	}
	if !strings.Contains(asm, "mul") {
		t.Error("Expected mul instruction")
	}
	if !strings.Contains(asm, "sdiv") {
		t.Error("Expected sdiv instruction")
	}
	if !strings.Contains(asm, "msub") {
		t.Error("Expected msub instruction for modulo")
	}
}

func TestBackendComparison(t *testing.T) {
	src := `main = () {
		val a = 10
		val b = 5
		val eq = a == b
		val lt = a < b
		val gt = a > b
		if eq {
			exit(1)
		}
		exit(0)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for comparison instructions
	if !strings.Contains(asm, "cmp") {
		t.Error("Expected cmp instruction")
	}
	if !strings.Contains(asm, "cset") {
		t.Error("Expected cset instruction")
	}
}

func TestBackendControlFlow(t *testing.T) {
	src := `main = () {
		val x = 10
		if x > 5 {
			exit(1)
		} else {
			exit(0)
		}
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for branch instructions
	if !strings.Contains(asm, "cbnz") || !strings.Contains(asm, "b ") {
		t.Error("Expected conditional and unconditional branch instructions")
	}
}

func TestBackendFunctionCall(t *testing.T) {
	src := `
	add = (a: s64, b: s64) -> s64 {
		return a + b
	}
	main = () {
		val result = add(5, 3)
		exit(result)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for function labels and calls
	if !strings.Contains(asm, "_add:") {
		t.Error("Expected _add: label")
	}
	if !strings.Contains(asm, "bl _add") {
		t.Error("Expected bl _add call")
	}
	if !strings.Contains(asm, "ret") {
		t.Error("Expected ret instruction")
	}
}

func TestBackendWhileLoop(t *testing.T) {
	src := `main = () {
		var i = 0
		while i < 5 {
			i = i + 1
		}
		exit(i)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for loop structure (blocks and branches)
	if !strings.Contains(asm, "b ") {
		t.Error("Expected branch instruction for loop")
	}
}

func TestBackendPrint(t *testing.T) {
	src := `main = () {
		print(42)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for print helper call
	if !strings.Contains(asm, "bl _sl_print_int") {
		t.Error("Expected bl _sl_print_int call")
	}
	// Check for write syscall in print helper
	if !strings.Contains(asm, "mov x16, #4") {
		t.Error("Expected write syscall (mov x16, #4)")
	}
}

func TestBackendStruct(t *testing.T) {
	src := `
	Point = struct {
		val x: s64
		val y: s64
	}
	main = () {
		val p = Point{ 10, 20 }
		exit(0)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for custom heap allocator call (struct allocation)
	if !strings.Contains(asm, "bl _sl_heap_alloc") {
		t.Error("Expected bl _sl_heap_alloc call for struct allocation")
	}
}

func TestBackendArray(t *testing.T) {
	src := `main = () {
		val arr = [1, 2, 3]
		val x = arr[0]
		exit(x)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for memory operations
	if !strings.Contains(asm, "bl _sl_heap_alloc") {
		t.Error("Expected bl _sl_heap_alloc call for array allocation")
	}
	if !strings.Contains(asm, "ldr") {
		t.Error("Expected ldr instruction for array access")
	}
}

func TestBackendName(t *testing.T) {
	b := New(nil)
	if b.Name() != "arm64" {
		t.Errorf("Expected backend name 'arm64', got '%s'", b.Name())
	}
}

// ============================================================================
// Nullable Type Tests
// ============================================================================

func TestBackendNullable(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "nullable with null",
			src: `main = () {
				val x: s64? = null
				exit(0)
			}`,
			contains: []string{"mov", "str"}, // Should store null representation
		},
		{
			name: "nullable with value",
			src: `main = () {
				val x: s64? = 42
				exit(0)
			}`,
			contains: []string{"mov", "str"}, // Should store value with tag
		},
		{
			name: "null check",
			src: `main = () {
				val x: s64? = null
				val isNull = x == null
				if isNull {
					exit(1)
				}
				exit(0)
			}`,
			contains: []string{"ldrb", "cbnz"}, // Load is-null byte, branch
		},
		{
			name: "elvis operator",
			src: `main = () {
				val x: s64? = null
				val y = x ?: 42
				exit(y)
			}`,
			contains: []string{"ldrb", "cbnz", "ldr"}, // Check null, branch, load value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)

			b := New(backend.DefaultConfig())
			asm, err := b.Generate(prog)
			if err != nil {
				t.Fatalf("Backend error: %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(asm, want) {
					t.Errorf("Expected assembly to contain %q\nGot:\n%s", want, asm)
				}
			}
		})
	}
}

func TestBackendSafeMethodCall(t *testing.T) {
	src := `
		Box = class {
			val value: s64

			getValue = (self: &Box) -> s64 {
				return self.value
			}

			create = (v: s64) -> *Box {
				return new Box{ v }
			}
		}
		main = () {
			val b: *Box? = Box.create(42)
			val v = b?.getValue()
			val result = v ?: 0
			exit(result)
		}
	`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for null check and branching
	if !strings.Contains(asm, "ldrb") {
		t.Error("Expected ldrb for null check")
	}
	if !strings.Contains(asm, "bl") {
		t.Error("Expected bl for method call")
	}
}

// ============================================================================
// Sleep Builtin Test
// ============================================================================

func TestBackendSleep(t *testing.T) {
	src := `main = () {
		sleep(1000000)  // 1ms in nanoseconds
		exit(0)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for sleep syscall (select with timeout)
	if !strings.Contains(asm, "mov x16, #93") {
		t.Error("Expected select syscall (mov x16, #93)")
	}
}

// ============================================================================
// Array Length Test
// ============================================================================

func TestBackendArrayLen(t *testing.T) {
	src := `main = () {
		val arr = [1, 2, 3, 4, 5]
		val length = len(arr)
		exit(length)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// len() is a compile-time constant, so check for the constant 5
	if !strings.Contains(asm, "#5") {
		t.Error("Expected constant 5 for array length")
	}
}

// ============================================================================
// Deep Copy Test
// ============================================================================

func TestBackendCopy(t *testing.T) {
	src := `
		Point = struct {
			var x: s64
			var y: s64
		}
		main = () {
			val p = new Point{ 10, 20 }
			val q = p.copy()
			exit(0)
		}
	`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Copy should call heap_alloc for new allocation
	// Count heap_alloc calls - should have at least 2 (original + copy)
	count := strings.Count(asm, "bl _sl_heap_alloc")
	if count < 2 {
		t.Errorf("Expected at least 2 heap_alloc calls for copy, got %d", count)
	}
}

// ============================================================================
// Field Access Test
// ============================================================================

func TestBackendFieldAccess(t *testing.T) {
	src := `
		Point = struct {
			val x: s64
			val y: s64
		}
		main = () {
			val p = Point{ 10, 20 }
			val x = p.x
			val y = p.y
			exit(x + y)
		}
	`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for ldr instructions (loading fields)
	if !strings.Contains(asm, "ldr") {
		t.Error("Expected ldr for field access")
	}
}

// ============================================================================
// Method Call Test
// ============================================================================

func TestBackendMethodCall(t *testing.T) {
	src := `
		Counter = class {
			var count: s64

			increment = (self: &&Counter) {
				self.count = self.count + 1
			}

			getValue = (self: &Counter) -> s64 {
				return self.count
			}
		}
		main = () {
			val c = new Counter{ 0 }
			c.increment()
			c.increment()
			val result = c.getValue()
			exit(result)
		}
	`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for method function labels
	if !strings.Contains(asm, "Counter_increment") {
		t.Error("Expected Counter_increment label")
	}
	if !strings.Contains(asm, "Counter_getValue") {
		t.Error("Expected Counter_getValue label")
	}
	// Check for method calls
	if !strings.Contains(asm, "bl") {
		t.Error("Expected bl for method calls")
	}
}

// ============================================================================
// For Loop Test
// ============================================================================

func TestBackendForLoop(t *testing.T) {
	src := `main = () {
		var sum = 0
		for (var i = 0; i < 10; i = i + 1) {
			sum = sum + i
		}
		exit(sum)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// Check for loop structure
	if !strings.Contains(asm, "b ") {
		t.Error("Expected branch instruction for loop")
	}
	if !strings.Contains(asm, "cmp") {
		t.Error("Expected comparison in loop condition")
	}
}

// ============================================================================
// When Expression Test
// ============================================================================

func TestBackendWhenExpression(t *testing.T) {
	src := `main = () {
		val x = 5
		val result = when {
			x < 0 -> 0 - 1
			x == 0 -> 0
			else -> 1
		}
		exit(result)
	}`

	prog := compileToIR(t, src)

	b := New(backend.DefaultConfig())
	asm, err := b.Generate(prog)
	if err != nil {
		t.Fatalf("Backend error: %v", err)
	}

	// When should generate multiple branches
	if !strings.Contains(asm, "cmp") {
		t.Error("Expected comparisons in when expression")
	}
}

// ============================================================================
// Stack Layout Tests
// ============================================================================

func TestComputeStackLayout(t *testing.T) {
	tests := []struct {
		name           string
		numParams      int
		numValues      int
		expectedSize   int
		checkAlignment bool
	}{
		{
			name:           "empty function",
			numParams:      0,
			numValues:      0,
			expectedSize:   16, // Just the saved x29, x30
			checkAlignment: true,
		},
		{
			name:           "single parameter",
			numParams:      1,
			numValues:      0,
			expectedSize:   32, // 16 (frame) + 8 (param) = 24, rounded to 32
			checkAlignment: true,
		},
		{
			name:           "multiple parameters",
			numParams:      3,
			numValues:      0,
			expectedSize:   48, // 16 (frame) + 24 (3 params) = 40, rounded to 48
			checkAlignment: true,
		},
		{
			name:           "parameters and values",
			numParams:      2,
			numValues:      3,
			expectedSize:   64, // 16 + 16 (params) + 24 (values) = 56, rounded to 64
			checkAlignment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test function with the specified parameters and values
			prog := ir.NewProgram()
			fn := prog.NewFunction("test", ir.TypeVoid)

			// Add parameters
			for i := 0; i < tt.numParams; i++ {
				fn.NewParam(ir.TypeS64)
			}

			// Add a block with values
			if tt.numValues > 0 {
				block := fn.NewBlock(ir.BlockPlain)
				for i := 0; i < tt.numValues; i++ {
					v := block.NewValue(ir.OpAdd, ir.TypeS64)
					// Add dummy args to make it a valid value
					c1 := block.NewValue(ir.OpConst, ir.TypeS64)
					c2 := block.NewValue(ir.OpConst, ir.TypeS64)
					v.AddArg(c1)
					v.AddArg(c2)
				}
			}

			layout := ComputeStackLayout(fn)

			// Check alignment
			if tt.checkAlignment && layout.Size%16 != 0 {
				t.Errorf("Stack size %d is not 16-byte aligned", layout.Size)
			}

			// Check that all parameters have offsets
			for _, param := range fn.Params {
				if _, ok := layout.Offsets[param]; !ok {
					t.Errorf("Parameter %v has no stack offset", param)
				}
			}

			// Check expected size
			if layout.Size != tt.expectedSize {
				t.Errorf("Expected stack size %d, got %d", tt.expectedSize, layout.Size)
			}
		})
	}
}

func TestComputeStackLayoutPurity(t *testing.T) {
	// Test that ComputeStackLayout is a pure function (no side effects)
	prog := ir.NewProgram()
	fn := prog.NewFunction("test", ir.TypeVoid)
	fn.NewParam(ir.TypeS64)
	fn.NewParam(ir.TypeS64)

	// Call twice
	layout1 := ComputeStackLayout(fn)
	layout2 := ComputeStackLayout(fn)

	// Results should be equivalent
	if layout1.Size != layout2.Size {
		t.Errorf("Stack sizes differ: %d vs %d", layout1.Size, layout2.Size)
	}

	// Modifying one shouldn't affect the other
	layout1.Offsets[fn.Params[0]] = 999
	if layout2.Offsets[fn.Params[0]] == 999 {
		t.Error("Modifying layout1 affected layout2 - not a pure function")
	}
}

func TestComputeStackLayoutEdgeCases(t *testing.T) {
	t.Run("phi_nodes_get_stack_slots", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		block := fn.NewBlock(ir.BlockPlain)

		// Create a phi node
		phi := block.NewValue(ir.OpPhi, ir.TypeS64)
		phi.PhiArgs = []*ir.PhiArg{} // Empty but valid

		layout := ComputeStackLayout(fn)

		if _, ok := layout.Offsets[phi]; !ok {
			t.Error("Phi node should have a stack slot")
		}
	})

	t.Run("constants_dont_get_stack_slots", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		block := fn.NewBlock(ir.BlockPlain)

		// Create a constant
		c := block.NewValue(ir.OpConst, ir.TypeS64)
		c.AuxInt = 42

		layout := ComputeStackLayout(fn)

		if _, ok := layout.Offsets[c]; ok {
			t.Error("Constant should NOT have a stack slot")
		}
	})

	t.Run("stores_dont_get_stack_slots", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		block := fn.NewBlock(ir.BlockPlain)

		// Create a store (no result type)
		store := block.NewValue(ir.OpStore, nil)

		layout := ComputeStackLayout(fn)

		if _, ok := layout.Offsets[store]; ok {
			t.Error("Store should NOT have a stack slot")
		}
	})

	t.Run("returns_dont_get_stack_slots", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		block := fn.NewBlock(ir.BlockPlain)

		// Create a return
		ret := block.NewValue(ir.OpReturn, nil)

		layout := ComputeStackLayout(fn)

		if _, ok := layout.Offsets[ret]; ok {
			t.Error("Return should NOT have a stack slot")
		}
	})

	t.Run("all_params_have_negative_offsets", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewParam(ir.TypeS64)
		fn.NewParam(ir.TypeS64)
		fn.NewParam(ir.TypeS64)

		layout := ComputeStackLayout(fn)

		for _, param := range fn.Params {
			offset, ok := layout.Offsets[param]
			if !ok {
				t.Error("Parameter missing from layout")
				continue
			}
			if offset >= 0 {
				t.Errorf("Parameter offset %d should be negative (below frame pointer)", offset)
			}
		}
	})

	t.Run("offsets_dont_overlap", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewParam(ir.TypeS64)
		fn.NewParam(ir.TypeS64)
		block := fn.NewBlock(ir.BlockPlain)
		block.NewValue(ir.OpAdd, ir.TypeS64)
		block.NewValue(ir.OpSub, ir.TypeS64)

		layout := ComputeStackLayout(fn)

		// Collect all offsets
		offsets := make(map[int]bool)
		for _, offset := range layout.Offsets {
			if offsets[offset] {
				t.Errorf("Duplicate offset: %d", offset)
			}
			offsets[offset] = true
		}
	})

	t.Run("large_function_alignment", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)

		// Add many parameters
		for i := 0; i < 20; i++ {
			fn.NewParam(ir.TypeS64)
		}

		// Add many values
		block := fn.NewBlock(ir.BlockPlain)
		for i := 0; i < 50; i++ {
			block.NewValue(ir.OpAdd, ir.TypeS64)
		}

		layout := ComputeStackLayout(fn)

		if layout.Size%16 != 0 {
			t.Errorf("Large stack size %d is not 16-byte aligned", layout.Size)
		}
	})
}

// ============================================================================
// Label Manager Tests
// ============================================================================

func TestLabelManager(t *testing.T) {
	t.Run("NextLabel generates unique labels", func(t *testing.T) {
		lm := NewLabelManager()

		seen := make(map[int]bool)
		for i := 0; i < 100; i++ {
			label := lm.NextLabel()
			if seen[label] {
				t.Errorf("Duplicate label: %d", label)
			}
			seen[label] = true
		}
	})

	t.Run("BlockLabel returns set label", func(t *testing.T) {
		lm := NewLabelManager()
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		block := fn.NewBlock(ir.BlockPlain)

		lm.SetBlockLabel(block, "_test_b0")

		if got := lm.BlockLabel(block); got != "_test_b0" {
			t.Errorf("Expected _test_b0, got %s", got)
		}
	})

	t.Run("InitBlockLabels sets all block labels", func(t *testing.T) {
		lm := NewLabelManager()
		prog := ir.NewProgram()
		fn := prog.NewFunction("myFunc", ir.TypeVoid)
		fn.NewBlock(ir.BlockPlain)
		fn.NewBlock(ir.BlockPlain)
		fn.NewBlock(ir.BlockPlain)

		lm.InitBlockLabels(fn)

		for _, block := range fn.Blocks {
			label := lm.BlockLabel(block)
			if label == "" {
				t.Errorf("Block %d has no label", block.ID)
			}
			if !strings.Contains(label, "myFunc") {
				t.Errorf("Label %s doesn't contain function name", label)
			}
		}
	})
}

// ============================================================================
// Prologue/Body/Epilogue Tests
// ============================================================================

func TestEmitPrologue(t *testing.T) {
	t.Run("main function has global directive", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("main", ir.TypeVoid)
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitPrologue()
		output := g.builder.String()

		if !strings.Contains(output, ".global _main") {
			t.Error("Expected .global _main for main function")
		}
		if !strings.Contains(output, "_main:") {
			t.Error("Expected _main: label")
		}
	})

	t.Run("non-main function has underscore prefix", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("helper", ir.TypeVoid)
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitPrologue()
		output := g.builder.String()

		if strings.Contains(output, ".global") {
			t.Error("Non-main function should not have .global")
		}
		if !strings.Contains(output, "_helper:") {
			t.Error("Expected _helper: label")
		}
	})

	t.Run("prologue saves frame pointer and link register", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitPrologue()
		output := g.builder.String()

		if !strings.Contains(output, "stp x29, x30, [sp, #-16]!") {
			t.Error("Expected frame pointer/link register save")
		}
		if !strings.Contains(output, "mov x29, sp") {
			t.Error("Expected frame pointer setup")
		}
	})

	t.Run("prologue allocates stack space when needed", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewParam(ir.TypeS64)
		fn.NewParam(ir.TypeS64)
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitPrologue()
		output := g.builder.String()

		if !strings.Contains(output, "sub sp, sp,") {
			t.Error("Expected stack allocation for parameters")
		}
	})

	t.Run("prologue stores first 8 params in registers", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		for i := 0; i < 3; i++ {
			fn.NewParam(ir.TypeS64)
		}
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitPrologue()
		output := g.builder.String()

		// Check that x0, x1, x2 are stored to stack
		if !strings.Contains(output, "str x0,") {
			t.Error("Expected x0 to be stored to stack")
		}
		if !strings.Contains(output, "str x1,") {
			t.Error("Expected x1 to be stored to stack")
		}
		if !strings.Contains(output, "str x2,") {
			t.Error("Expected x2 to be stored to stack")
		}
	})
}

func TestEmitEpilogue(t *testing.T) {
	t.Run("epilogue restores stack when allocated", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewParam(ir.TypeS64)
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitEpilogue()
		output := g.builder.String()

		if !strings.Contains(output, "add sp, sp,") {
			t.Error("Expected stack restoration")
		}
	})

	t.Run("epilogue restores frame pointer and returns", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewBlock(ir.BlockPlain)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitEpilogue()
		output := g.builder.String()

		if !strings.Contains(output, "ldp x29, x30, [sp], #16") {
			t.Error("Expected frame pointer/link register restore")
		}
		if !strings.Contains(output, "ret") {
			t.Error("Expected ret instruction")
		}
	})

	t.Run("epilogue skips stack restore when no allocation", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		fn.NewBlock(ir.BlockPlain)

		// Manually set layout with zero size
		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = &StackLayout{Size: 0, Offsets: make(map[*ir.Value]int)}

		g.emitEpilogue()
		output := g.builder.String()

		if strings.Contains(output, "add sp, sp,") {
			t.Error("Should not restore stack when size is 0")
		}
	})
}

func TestGenerateBody(t *testing.T) {
	t.Run("generates code for all blocks", func(t *testing.T) {
		src := `main = () {
			val x = 10
			if x > 5 {
				exit(1)
			} else {
				exit(0)
			}
		}`

		prog := compileToIR(t, src)

		g := newGenerator(prog, backend.DefaultConfig())
		fn := prog.Main()
		g.fn = fn
		g.layout = ComputeStackLayout(fn)
		g.labels.InitBlockLabels(fn)

		err := g.generateBody()
		if err != nil {
			t.Fatalf("generateBody error: %v", err)
		}

		output := g.builder.String()

		// Check for block labels (if/else creates multiple blocks)
		if !strings.Contains(output, "_main_b") {
			t.Error("Expected block labels")
		}
	})

	t.Run("returns error for malformed blocks", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)

		// Create a malformed if block (no successors)
		block := fn.NewBlock(ir.BlockIf)
		block.Control = block.NewValue(ir.OpConst, ir.TypeBool)
		// Don't set Succs - this should cause an error

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)
		g.labels.InitBlockLabels(fn)

		err := g.generateBody()
		if err == nil {
			t.Error("Expected error for malformed if block")
		}
	})
}

func TestEmitReturnValue(t *testing.T) {
	t.Run("loads simple return value into x0", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeS64)
		block := fn.NewBlock(ir.BlockReturn)

		// Create constant and return value
		c := block.NewValue(ir.OpConst, ir.TypeS64)
		c.AuxInt = 42
		ret := block.NewValue(ir.OpReturn, ir.TypeVoid)
		ret.AddArg(c)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		g.emitReturnValue(block)
		output := g.builder.String()

		// Should load constant into x0
		if !strings.Contains(output, "#42") && !strings.Contains(output, "x0") {
			t.Error("Expected return value to be loaded")
		}
	})

	t.Run("handles void return", func(t *testing.T) {
		prog := ir.NewProgram()
		fn := prog.NewFunction("test", ir.TypeVoid)
		block := fn.NewBlock(ir.BlockReturn)

		// Return with no value
		block.NewValue(ir.OpReturn, ir.TypeVoid)

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = fn
		g.layout = ComputeStackLayout(fn)

		// Should not panic
		g.emitReturnValue(block)
		// No specific output expected for void return
	})

	t.Run("handles nullable return value", func(t *testing.T) {
		src := `
			getValue = () -> s64? {
				return 42
			}
			main = () {
				val x = getValue()
				exit(0)
			}
		`

		prog := compileToIR(t, src)

		// Find the getValue function
		var getValueFn *ir.Function
		for _, fn := range prog.Functions {
			if fn.Name == "getValue" {
				getValueFn = fn
				break
			}
		}
		if getValueFn == nil {
			t.Fatal("getValue function not found")
		}

		g := newGenerator(prog, backend.DefaultConfig())
		g.fn = getValueFn
		g.layout = ComputeStackLayout(getValueFn)
		g.labels.InitBlockLabels(getValueFn)

		// Find return block
		var retBlock *ir.Block
		for _, block := range getValueFn.Blocks {
			if block.Kind == ir.BlockReturn {
				retBlock = block
				break
			}
		}
		if retBlock == nil {
			t.Fatal("Return block not found")
		}

		g.emitReturnValue(retBlock)
		// Should not panic, nullable handling is tested through integration
	})
}

// ============================================================================
// Panic Message Registry Tests
// ============================================================================

func TestPanicMessageLen(t *testing.T) {
	// Verify that Len() returns the actual length of the message.
	// Note: The original hardcoded lengths were incorrect! This refactoring fixed them.
	// Old values vs correct values:
	//   PanicDivZero: 25 -> 24 (actual)
	//   PanicModZero: 23 -> 22 (actual)
	//   PanicBounds: 35 -> 33 (actual)
	//   etc.
	tests := []struct {
		name    string
		msg     panicMessage
		wantLen int
	}{
		{"PanicDivZero", PanicDivZero, len("panic: division by zero\n")},
		{"PanicModZero", PanicModZero, len("panic: modulo by zero\n")},
		{"PanicBounds", PanicBounds, len("panic: array index out of bounds\n")},
		{"PanicOverflowAdd", PanicOverflowAdd, len("panic: integer overflow: addition\n")},
		{"PanicOverflowSub", PanicOverflowSub, len("panic: integer overflow: subtraction\n")},
		{"PanicOverflowMul", PanicOverflowMul, len("panic: integer overflow: multiplication\n")},
		{"PanicUnsignedOverAdd", PanicUnsignedOverAdd, len("panic: unsigned overflow: addition\n")},
		{"PanicUnsignedUnderSub", PanicUnsignedUnderSub, len("panic: unsigned underflow: subtraction\n")},
		{"PanicUnsignedOverMul", PanicUnsignedOverMul, len("panic: unsigned overflow: multiplication\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.Len(); got != tt.wantLen {
				t.Errorf("%s.Len() = %d, want %d (message: %q)", tt.name, got, tt.wantLen, tt.msg.Message)
			}
		})
	}
}

func TestPanicMessageLabels(t *testing.T) {
	// Verify labels start with underscore (assembly convention)
	for _, p := range allPanicMessages {
		if p.Label[0] != '_' {
			t.Errorf("Label %q should start with underscore", p.Label)
		}
	}
}

func TestPanicMessageEndsWithNewline(t *testing.T) {
	for _, p := range allPanicMessages {
		if p.Message[len(p.Message)-1] != '\n' {
			t.Errorf("Message %q should end with newline", p.Message)
		}
	}
}

func TestAllPanicMessagesContainsAll(t *testing.T) {
	// Verify allPanicMessages contains all defined panics
	expected := []panicMessage{
		PanicDivZero,
		PanicModZero,
		PanicBounds,
		PanicOverflowAdd,
		PanicOverflowSub,
		PanicOverflowMul,
		PanicUnsignedOverAdd,
		PanicUnsignedUnderSub,
		PanicUnsignedOverMul,
	}

	if len(allPanicMessages) != len(expected) {
		t.Errorf("allPanicMessages has %d entries, expected %d", len(allPanicMessages), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, got := range allPanicMessages {
			if got.Label == exp.Label {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("allPanicMessages missing %s", exp.Label)
		}
	}
}

func TestEmitPanic(t *testing.T) {
	prog := ir.NewProgram()
	fn := prog.NewFunction("test", ir.TypeVoid)
	fn.NewBlock(ir.BlockPlain)

	g := newGenerator(prog, backend.DefaultConfig())
	g.fn = fn
	g.layout = ComputeStackLayout(fn)

	g.emitPanic(PanicDivZero)
	output := g.builder.String()

	// Check that the panic label is referenced
	if !strings.Contains(output, PanicDivZero.Label) {
		t.Error("Expected panic label in output")
	}

	// Check that the correct length is emitted (24, not the old incorrect 25)
	expectedLen := fmt.Sprintf("#%d", PanicDivZero.Len())
	if !strings.Contains(output, expectedLen) {
		t.Errorf("Expected message length %s, got: %s", expectedLen, output)
	}

	// Check that panic helper is called
	if !strings.Contains(output, "bl _sl_panic") {
		t.Error("Expected call to _sl_panic")
	}
}

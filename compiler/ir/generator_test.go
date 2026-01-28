package ir

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/compiler/lexer"
	"github.com/seanrogers2657/slang/compiler/parser"
	"github.com/seanrogers2657/slang/compiler/semantic"
)

// compileToIR is a helper that compiles source code to IR.
func compileToIR(t *testing.T, src string) *Program {
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
	irProg, err := Generate(typed)
	if err != nil {
		t.Fatalf("IR generation error: %v", err)
	}

	return irProg
}

func TestGenerateLiterals(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "integer literal",
			src: `main = () {
				val x = 42
			}`,
			contains: []string{"Const 42", "s64"},
		},
		{
			name: "boolean true",
			src: `main = () {
				val x = true
			}`,
			contains: []string{"Const true", "bool"},
		},
		{
			name: "boolean false",
			src: `main = () {
				val x = false
			}`,
			contains: []string{"Const false", "bool"},
		},
		{
			name: "string literal",
			src: `main = () {
				val s = "hello"
			}`,
			contains: []string{"Const", "hello", "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateBinaryExpressions(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "addition",
			src: `main = () {
				val x = 1 + 2
			}`,
			contains: []string{"Add"},
		},
		{
			name: "subtraction",
			src: `main = () {
				val x = 5 - 3
			}`,
			contains: []string{"Sub"},
		},
		{
			name: "multiplication",
			src: `main = () {
				val x = 4 * 5
			}`,
			contains: []string{"Mul"},
		},
		{
			name: "division",
			src: `main = () {
				val x = 10 / 2
			}`,
			contains: []string{"Div"},
		},
		{
			name: "comparison",
			src: `main = () {
				val x = 1 < 2
			}`,
			contains: []string{"Lt", "bool"},
		},
		{
			name: "equality",
			src: `main = () {
				val x = 1 == 2
			}`,
			contains: []string{"Eq", "bool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateUnaryExpressions(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "negation",
			src: `main = () {
				val y = 42
				val x = 0 - y
			}`,
			contains: []string{"Sub"},
		},
		{
			name: "logical not",
			src: `main = () {
				val x = !true
			}`,
			contains: []string{"Not"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateVariables(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "variable use",
			src: `main = () {
				val x = 10
				val y = x
			}`,
			contains: []string{"Const 10"},
		},
		{
			name: "variable in expression",
			src: `main = () {
				val x = 5
				val y = x + 3
			}`,
			contains: []string{"Add"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateControlFlow(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "if statement",
			src: `main = () {
				if true {
					val x = 1
				}
			}`,
			contains: []string{"if v", "b0", "b1"},
		},
		{
			name: "if-else statement",
			src: `main = () {
				if true {
					val x = 1
				} else {
					val x = 2
				}
			}`,
			contains: []string{"if v", "b0", "b1", "b2"},
		},
		{
			name: "while loop",
			src: `main = () {
				var i = 0
				while i < 10 {
					i = i + 1
				}
			}`,
			contains: []string{"Lt", "if v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateFunctionCalls(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "exit builtin",
			src: `main = () {
				exit(0)
			}`,
			contains: []string{"Call @exit"},
		},
		{
			name: "print builtin",
			src: `main = () {
				print(42)
			}`,
			contains: []string{"Call @print"},
		},
		{
			name: "user function call",
			src: `
				add = (a: s64, b: s64) -> s64 {
					return a + b
				}
				main = () {
					val x = add(1, 2)
				}
			`,
			contains: []string{"Call @add"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateReturn(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "return value",
			src: `
				getValue = () -> s64 {
					return 42
				}
				main = () {
					val x = getValue()
				}
			`,
			contains: []string{"return v"},
		},
		{
			name: "void return",
			src: `main = () {
				return
			}`,
			contains: []string{"return"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateStructs(t *testing.T) {
	src := `
		Point = struct {
			val x: s64
			val y: s64
		}
		main = () {
			val p = Point{ 1, 2 }
		}
	`

	prog := compileToIR(t, src)
	ir := String(prog)

	// Should have struct definition
	if !strings.Contains(ir, "type Point struct") {
		t.Errorf("IR should contain struct definition\nGot:\n%s", ir)
	}

	// Should have Alloc for struct creation
	if !strings.Contains(ir, "Alloc") {
		t.Errorf("IR should contain Alloc for struct\nGot:\n%s", ir)
	}
}

func TestGenerateArrays(t *testing.T) {
	src := `main = () {
		val arr = [1, 2, 3]
		val x = arr[0]
	}`

	prog := compileToIR(t, src)
	ir := String(prog)

	// Should have Alloc for array
	if !strings.Contains(ir, "Alloc") {
		t.Errorf("IR should contain Alloc for array\nGot:\n%s", ir)
	}

	// Should have IndexPtr for element access
	if !strings.Contains(ir, "IndexPtr") {
		t.Errorf("IR should contain IndexPtr for element access\nGot:\n%s", ir)
	}

	// Should have Load for reading element
	if !strings.Contains(ir, "Load") {
		t.Errorf("IR should contain Load for reading element\nGot:\n%s", ir)
	}
}

func TestGenerateSSA(t *testing.T) {
	// Test that variable reassignment creates proper SSA form
	src := `main = () {
		var x = 1
		x = 2
		x = 3
		val y = x
	}`

	prog := compileToIR(t, src)

	// Should generate valid IR
	errs := Validate(prog)
	if len(errs) > 0 {
		ir := String(prog)
		t.Errorf("IR validation errors: %v\nIR:\n%s", errs, ir)
	}
}

func TestGenerateShortCircuit(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "logical and",
			src: `main = () {
				val x = true && false
			}`,
			contains: []string{"Phi", "if v"},
		},
		{
			name: "logical or",
			src: `main = () {
				val x = true || false
			}`,
			contains: []string{"Phi", "if v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestValidateGeneratedIR(t *testing.T) {
	// Test various programs and ensure the generated IR is valid
	programs := []string{
		`main = () { val x = 42 }`,
		`main = () { val x = 1 + 2 }`,
		`main = () { if true { val x = 1 } }`,
		`main = () {
			var i = 0
			while i < 10 { i = i + 1 }
		}`,
		`add = (a: s64, b: s64) -> s64 { return a + b }
		main = () { val x = add(1, 2) }`,
	}

	for i, src := range programs {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			prog := compileToIR(t, src)
			errs := Validate(prog)
			if len(errs) > 0 {
				ir := String(prog)
				t.Errorf("validation errors: %v\nIR:\n%s", errs, ir)
			}
		})
	}
}

// ============================================================================
// Class and Method Tests
// ============================================================================

func TestGenerateClasses(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "class with instance method",
			src: `
				Counter = class {
					var count: s64

					increment = (self: &&Counter) {
						self.count = self.count + 1
					}
				}
				main = () {
					val c = Heap.new(Counter{ 0 })
					c.increment()
				}
			`,
			contains: []string{"Counter_increment", "FieldPtr", "Load", "Store"},
		},
		{
			name: "class with static method",
			src: `
				Math = class {
					val dummy: s64

					add = (a: s64, b: s64) -> s64 {
						return a + b
					}
				}
				main = () {
					val result = Math.add(1, 2)
				}
			`,
			contains: []string{"Math_add", "Add", "return v"},
		},
		{
			name: "class with method returning value",
			src: `
				Point = class {
					val x: s64
					val y: s64

					sum = (self: &Point) -> s64 {
						return self.x + self.y
					}
				}
				main = () {
					val p = Heap.new(Point{ 10, 20 })
					val s = p.sum()
				}
			`,
			contains: []string{"Point_sum", "Add", "return v"},
		},
		{
			name: "class with self field access",
			src: `
				Box = class {
					var value: s64

					getValue = (self: &Box) -> s64 {
						return self.value
					}
				}
				main = () {
					val b = Heap.new(Box{ 42 })
					val v = b.getValue()
				}
			`,
			contains: []string{"FieldPtr", "Load"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}

			// Validate IR
			errs := Validate(prog)
			if len(errs) > 0 {
				t.Errorf("IR validation errors: %v\nIR:\n%s", errs, ir)
			}
		})
	}
}

func TestGenerateObjects(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "heap allocated object",
			src: `
				Point = class {
					var x: s64
					var y: s64
				}
				main = () {
					val p = Heap.new(Point{ 10, 20 })
					print(p.x)
				}
			`,
			contains: []string{"Alloc", "FieldPtr", "Load"},
		},
		{
			name: "object method call",
			src: `
				Counter = class {
					var count: s64

					get = (self: &Counter) -> s64 {
						return self.count
					}
				}
				main = () {
					val c = Heap.new(Counter{ 5 })
					val v = c.get()
				}
			`,
			contains: []string{"Counter_get", "Call"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

// ============================================================================
// Nullable Operation Tests
// ============================================================================

func TestGenerateNullableOperations(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "null literal",
			src: `main = () {
				val x: s64? = null
			}`,
			contains: []string{"WrapNull"},
		},
		{
			name: "nullable with value",
			src: `main = () {
				val x: s64? = 42
			}`,
			contains: []string{"Wrap"},
		},
		{
			name: "null comparison",
			src: `main = () {
				val x: s64? = null
				val isNull = x == null
			}`,
			contains: []string{"IsNull"},
		},
		{
			name: "elvis operator",
			src: `main = () {
				val x: s64? = null
				val y = x ?: 10
			}`,
			contains: []string{"IsNull", "Phi", "Unwrap"},
		},
		{
			name: "elvis with non-null",
			src: `main = () {
				val x: s64? = 42
				val y = x ?: 10
			}`,
			contains: []string{"IsNull", "Phi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateSafeNavigation(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "safe method call on nullable",
			src: `
				Box = class {
					val value: s64

					getValue = (self: &Box) -> s64 {
						return self.value
					}

					create = (v: s64) -> *Box {
						return Heap.new(Box{ v })
					}
				}
				main = () {
					val b: *Box? = Box.create(42)
					val v = b?.getValue()
				}
			`,
			contains: []string{"IsNull", "Phi"},
		},
		{
			name: "safe method call on null",
			src: `
				Counter = class {
					var count: s64

					getValue = (self: &Counter) -> s64 {
						return self.count
					}
				}
				main = () {
					val c: *Counter? = null
					val v = c?.getValue()
				}
			`,
			contains: []string{"IsNull", "WrapNull", "Phi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

// ============================================================================
// Field and Array Operation Tests
// ============================================================================

func TestGenerateFieldAccess(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "struct field read",
			src: `
				Point = struct {
					val x: s64
					val y: s64
				}
				main = () {
					val p = Point{ 10, 20 }
					val x = p.x
				}
			`,
			contains: []string{"FieldPtr", "Load"},
		},
		{
			name: "struct field write",
			src: `
				Point = struct {
					var x: s64
					var y: s64
				}
				main = () {
					var p = Point{ 10, 20 }
					p.x = 100
				}
			`,
			contains: []string{"FieldPtr", "Store"},
		},
		{
			name: "nested struct field access",
			src: `
				Inner = struct {
					val value: s64
				}
				Outer = struct {
					val inner: Inner
				}
				main = () {
					val o = Outer{ Inner{ 42 } }
					val v = o.inner.value
				}
			`,
			contains: []string{"FieldPtr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestGenerateArrayLen(t *testing.T) {
	src := `main = () {
		val arr = [1, 2, 3, 4, 5]
		val length = len(arr)
	}`

	prog := compileToIR(t, src)
	ir := String(prog)

	// len() is evaluated at compile-time for fixed-size arrays
	// It generates a constant value, not an ArrayLen operation
	if !strings.Contains(ir, "Const 5") {
		t.Errorf("IR should contain Const 5 for len() of 5-element array\nGot:\n%s", ir)
	}
}

func TestGenerateHeapNew(t *testing.T) {
	src := `
		Point = struct {
			var x: s64
			var y: s64
		}
		main = () {
			val p = Heap.new(Point{ 10, 20 })
			print(p.x)
		}
	`

	prog := compileToIR(t, src)
	ir := String(prog)

	if !strings.Contains(ir, "Alloc") {
		t.Errorf("IR should contain Alloc for Heap.new\nGot:\n%s", ir)
	}
	if !strings.Contains(ir, "Store") {
		t.Errorf("IR should contain Store for initializing heap object\nGot:\n%s", ir)
	}
}

func TestGenerateCopy(t *testing.T) {
	src := `
		Point = struct {
			var x: s64
			var y: s64
		}
		main = () {
			val p = Heap.new(Point{ 10, 20 })
			val q = p.copy()
			q.x = 100
		}
	`

	prog := compileToIR(t, src)
	ir := String(prog)

	if !strings.Contains(ir, "Copy") {
		t.Errorf("IR should contain Copy for .copy() method\nGot:\n%s", ir)
	}
}

// ============================================================================
// If Expression Tests
// ============================================================================

func TestGenerateIfExpressions(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "simple if expression",
			src: `main = () {
				val x = if true { 1 } else { 2 }
			}`,
			contains: []string{"Phi", "if v"},
		},
		{
			name: "if expression with condition",
			src: `main = () {
				val a = 5
				val x = if a > 3 { 10 } else { 20 }
			}`,
			contains: []string{"Gt", "Phi", "if v"},
		},
		{
			name: "chained if-else expression",
			src: `main = () {
				val a = 5
				val x = if a < 0 { 0 - 1 } else { if a == 0 { 0 } else { 1 } }
			}`,
			contains: []string{"Lt", "Eq", "Phi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

// ============================================================================
// When Expression Tests
// ============================================================================

func TestGenerateWhenExpressions(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "when statement",
			src: `main = () {
				val x = 5
				when {
					x > 10 -> print(1)
					x > 3 -> print(2)
					else -> print(3)
				}
			}`,
			contains: []string{"Gt", "if v"},
		},
		{
			name: "when expression",
			src: `main = () {
				val x = 5
				val result = when {
					x < 0 -> 0 - 1
					x == 0 -> 0
					else -> 1
				}
			}`,
			contains: []string{"Lt", "Eq", "Phi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

// ============================================================================
// For Loop Tests
// ============================================================================

func TestGenerateForLoops(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "simple for loop",
			src: `main = () {
				for (var i = 0; i < 10; i = i + 1) {
					print(i)
				}
			}`,
			contains: []string{"Lt", "Add", "Phi", "if v"},
		},
		{
			name: "for loop with break",
			src: `main = () {
				for (var i = 0; i < 100; i = i + 1) {
					if i > 5 {
						break
					}
				}
			}`,
			contains: []string{"Lt", "Gt"},
		},
		{
			name: "for loop with continue",
			src: `main = () {
				for (var i = 0; i < 10; i = i + 1) {
					if i == 5 {
						continue
					}
					print(i)
				}
			}`,
			contains: []string{"Eq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

// ============================================================================
// Type Conversion Tests
// ============================================================================

func TestGenerateTypeConversions(t *testing.T) {
	// Test that various integer types generate valid IR
	// Note: Constants may be printed as their base type (s64)
	// but the semantic type checking ensures type safety
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "s8 type",
			src: `main = () {
				val x: s8 = 10
				exit(0)
			}`,
		},
		{
			name: "s16 type",
			src: `main = () {
				val x: s16 = 1000
				exit(0)
			}`,
		},
		{
			name: "u8 type",
			src: `main = () {
				val x: u8 = 255
				exit(0)
			}`,
		},
		{
			name: "u64 type",
			src: `main = () {
				val x: u64 = 1000000
				exit(0)
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			// Just validate that IR compiles without errors
			errs := Validate(prog)
			if len(errs) > 0 {
				t.Errorf("IR validation errors: %v\nIR:\n%s", errs, ir)
			}
		})
	}
}

// ============================================================================
// Binary Operator Tests (refactored functions)
// ============================================================================

func TestBinaryOpMap(t *testing.T) {
	// Verify all expected operators are in the map
	expectedOps := []struct {
		op     string
		irOp   Op
	}{
		{"+", OpAdd},
		{"-", OpSub},
		{"*", OpMul},
		{"/", OpDiv},
		{"%", OpMod},
		{"==", OpEq},
		{"!=", OpNe},
		{"<", OpLt},
		{"<=", OpLe},
		{">", OpGt},
		{">=", OpGe},
	}

	for _, tt := range expectedOps {
		t.Run(tt.op, func(t *testing.T) {
			got, ok := binaryOpMap[tt.op]
			if !ok {
				t.Errorf("operator %q not in binaryOpMap", tt.op)
				return
			}
			if got != tt.irOp {
				t.Errorf("binaryOpMap[%q] = %v, want %v", tt.op, got, tt.irOp)
			}
		})
	}
}

func TestBinaryOpMapDoesNotContainSpecialOps(t *testing.T) {
	// These operators are handled specially and should NOT be in the map
	specialOps := []string{"&&", "||", "?:"}

	for _, op := range specialOps {
		if _, ok := binaryOpMap[op]; ok {
			t.Errorf("special operator %q should not be in binaryOpMap", op)
		}
	}
}

func TestNullComparison(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "equals null",
			src: `main = () {
				val x: s64? = 42
				val isNull = x == null
			}`,
			contains: []string{"IsNull"},
		},
		{
			name: "not equals null",
			src: `main = () {
				val x: s64? = 42
				val notNull = x != null
			}`,
			contains: []string{"IsNull", "Not"},
		},
		{
			name: "null on left side",
			src: `main = () {
				val x: s64? = 42
				val isNull = null == x
			}`,
			contains: []string{"IsNull"},
		},
		{
			name: "regular equality still uses Eq",
			src: `main = () {
				val x = 42
				val y = 42
				val eq = x == y
			}`,
			contains: []string{"Eq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestShortCircuitOperators(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string
	}{
		{
			name: "logical and",
			src: `main = () {
				val a = true
				val b = false
				val c = a && b
			}`,
			// Short-circuit creates multiple blocks with Phi
			contains: []string{"Phi"},
		},
		{
			name: "logical or",
			src: `main = () {
				val a = true
				val b = false
				val c = a || b
			}`,
			contains: []string{"Phi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := compileToIR(t, tt.src)
			ir := String(prog)

			for _, want := range tt.contains {
				if !strings.Contains(ir, want) {
					t.Errorf("IR does not contain %q\nGot:\n%s", want, ir)
				}
			}
		})
	}
}

func TestElvisOperator(t *testing.T) {
	src := `main = () {
		val x: s64? = null
		val y = x ?: 42
	}`

	prog := compileToIR(t, src)
	ir := String(prog)

	// Elvis operator should generate null check and phi
	if !strings.Contains(ir, "IsNull") {
		t.Errorf("IR should contain IsNull for elvis operator\nGot:\n%s", ir)
	}
	if !strings.Contains(ir, "Phi") {
		t.Errorf("IR should contain Phi for elvis operator\nGot:\n%s", ir)
	}
}

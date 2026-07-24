package arm64

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/compiler/ir"
	"github.com/seanrogers2657/slang/compiler/ir/backend"
)

// genAsm compiles Slang source to ARM64 assembly text for inspection.
func genAsm(t *testing.T, src string) string {
	t.Helper()
	prog := compileToIR(t, src)
	asm, err := New(backend.DefaultConfig()).Generate(prog)
	if err != nil {
		t.Fatalf("backend error: %v", err)
	}
	return asm
}

// mustContain asserts every needle appears in the assembly.
func mustContain(t *testing.T, asm string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(asm, n) {
			t.Errorf("expected assembly to contain %q", n)
		}
	}
}

// mustNotContain asserts none of the needles appear in the assembly.
func mustNotContain(t *testing.T, asm string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(asm, n) {
			t.Errorf("expected assembly NOT to contain %q", n)
		}
	}
}

func TestReprOf128Bit(t *testing.T) {
	if r := reprOf(ir.TypeS128); !r.multiWord() || r.words != 2 {
		t.Errorf("reprOf(s128) = %+v, want two words", r)
	}
	if r := reprOf(ir.TypeU128); !r.multiWord() || r.words != 2 {
		t.Errorf("reprOf(u128) = %+v, want two words", r)
	}
	if r := reprOf(ir.TypeS64); r.multiWord() || r.words != 1 {
		t.Errorf("reprOf(s64) = %+v, want single word", r)
	}
}

func TestIs128Predicates(t *testing.T) {
	if !is128(ir.TypeS128) || !is128(ir.TypeU128) {
		t.Error("is128 should be true for s128/u128")
	}
	if is128(ir.TypeS64) || is128(ir.TypeBool) {
		t.Error("is128 should be false for s64/bool")
	}
	if !is128Signed(ir.TypeS128) {
		t.Error("is128Signed(s128) should be true")
	}
	if is128Signed(ir.TypeU128) {
		t.Error("is128Signed(u128) should be false (unsigned)")
	}
	if is128Signed(ir.TypeS64) {
		t.Error("is128Signed(s64) should be false (not 128-bit)")
	}
}

// TestInt128HelpersAlwaysEmitted verifies the runtime helper routines are
// present in every program's output.
func TestInt128HelpersAlwaysEmitted(t *testing.T) {
	asm := genAsm(t, `main = () { print(0) }`)
	mustContain(t, asm,
		"_sl_u128_divmod:",
		"_sl_s128_divmod:",
		"_sl_u128_mul_ovf:",
		"_sl_s128_mul_ovf:",
		"_sl_print_u128:",
		"_sl_print_s128:",
		"_sl_u128_to_str:",
		"_sl_s128_to_str:",
	)
}

func TestInt128SignedAddOverflowCheck(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: s128 = 100
		val b: s128 = 7
		print(a + b)
	}`)
	// Signed 128-bit add traps via the sign-bit test and PanicOverflowAdd.
	mustContain(t, asm, "_sl_ok128_", "_sl_panic_overflow_add@PAGE")
	// It must NOT use the unsigned-overflow panic.
	mustNotContain(t, asm, "_sl_panic_unsigned_overflow_add@PAGE")
}

func TestInt128UnsignedAddOverflowCheck(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: u128 = 100
		val b: u128 = 7
		print(a + b)
	}`)
	// Unsigned add traps on carry-out (the _sl_ovf128_ label is unique to the
	// 128-bit unsigned compare) via PanicUnsignedOverAdd.
	mustContain(t, asm, "_sl_ovf128_", "_sl_panic_unsigned_overflow_add@PAGE")
}

func TestInt128UnsignedSubUnderflowCheck(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: u128 = 100
		val b: u128 = 7
		print(a - b)
	}`)
	mustContain(t, asm, "_sl_ovf128_", "_sl_panic_unsigned_underflow_sub@PAGE")
}

func TestInt128SignedNegOverflowCheck(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: s128 = 5
		print(-a)
	}`)
	// Negation traps on INT_MIN; the check has a unique label and the 2^127
	// constant is materialised with lsl #63.
	mustContain(t, asm, "_sl_negok128_", "lsl x16, x16, #63", "_sl_panic_overflow_neg@PAGE")
}

func TestInt128SignedDivOverflowAndHelper(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: s128 = 100
		val b: s128 = 7
		print(a / b)
	}`)
	// Signed div calls the signed divmod helper and has the INT_MIN/-1 trap.
	mustContain(t, asm, "bl _sl_s128_divmod", "_sl_divof128_", "_sl_panic_overflow_div@PAGE")
}

func TestInt128UnsignedDivUsesUnsignedHelper(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: u128 = 100
		val b: u128 = 7
		print(a / b)
	}`)
	// Unsigned div calls the unsigned divmod helper and does NOT emit the
	// signed INT_MIN/-1 overflow trap.
	mustContain(t, asm, "bl _sl_u128_divmod")
	mustNotContain(t, asm, "_sl_divof128_")
}

func TestInt128UnsignedMulOverflowHelper(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: u128 = 100
		val b: u128 = 7
		print(a * b)
	}`)
	mustContain(t, asm, "bl _sl_u128_mul_ovf", "_sl_mulok128_", "_sl_panic_unsigned_overflow_mul@PAGE")
	mustNotContain(t, asm, "bl _sl_s128_mul_ovf")
}

func TestInt128SignedMulOverflowHelper(t *testing.T) {
	asm := genAsm(t, `main = () {
		val a: s128 = 100
		val b: s128 = 7
		print(a * b)
	}`)
	mustContain(t, asm, "bl _sl_s128_mul_ovf", "_sl_mulok128_", "_sl_panic_overflow_mul@PAGE")
	mustNotContain(t, asm, "bl _sl_u128_mul_ovf")
}

// TestInt128ConstantNeedsSlot verifies a 128-bit constant is given a stack slot
// (so it can be materialized as two words in the prologue), while narrower
// constants remain inline (no slot).
func TestInt128ConstantNeedsSlot(t *testing.T) {
	if !needsStackSlot(&ir.Value{Op: ir.OpConst, Type: ir.TypeS128}) {
		t.Error("a 128-bit constant should need a stack slot")
	}
	if !needsStackSlot(&ir.Value{Op: ir.OpConst, Type: ir.TypeU128}) {
		t.Error("a 128-bit constant should need a stack slot")
	}
	if needsStackSlot(&ir.Value{Op: ir.OpConst, Type: ir.TypeS64}) {
		t.Error("a 64-bit constant should be materialized inline (no slot)")
	}
	// A 128-bit value occupies 16 bytes; the layout must allocate at least that.
	if got := valueSize(&ir.Value{Op: ir.OpConst, Type: ir.TypeS128}); got != 16 {
		t.Errorf("valueSize(s128 const) = %d, want 16", got)
	}
}

// TestInt128PrintDispatch verifies print routes 128-bit values to the dedicated
// width-aware helpers.
func TestInt128PrintDispatch(t *testing.T) {
	signed := genAsm(t, `main = () {
		val x: s128 = 5
		print(x)
	}`)
	mustContain(t, signed, "bl _sl_print_s128")
	mustNotContain(t, signed, "bl _sl_print_u128")

	unsigned := genAsm(t, `main = () {
		val x: u128 = 5
		print(x)
	}`)
	mustContain(t, unsigned, "bl _sl_print_u128")
}

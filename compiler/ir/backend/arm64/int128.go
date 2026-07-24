package arm64

import "github.com/seanrogers2657/slang/compiler/ir"

// 128-bit integer codegen.
//
// A 128-bit value occupies two 64-bit words (low then high) in a stack slot.
// These helpers load operands into scratch registers, compute with the carry /
// borrow propagated by hand (slasm has no adc/sbc/ccmp), and store the two-word
// result back to the value's slot.
//
// Register convention within a single op: a = (x10 lo, x11 hi), b = (x12 lo,
// x13 hi); result accumulates in x9 (lo) and x14 (hi); x15 is scratch.
//
// Note: 128-bit arithmetic wraps (mod 2^128) and does not trap on overflow. The
// 64-bit-and-narrower paths still trap. This fixes the previous behaviour where
// s128/u128 arithmetic falsely trapped at the 64-bit boundary on in-range
// values; adding true 128-bit overflow detection is a possible follow-up.

// is128Signed reports whether v's type is a signed 128-bit integer.
func is128Signed(t ir.Type) bool {
	it, ok := t.(*ir.IntType)
	return ok && it.Bits >= 128 && it.Signed
}

// emit128PanicIfULess panics with p when the 128-bit value (aLo,aHi) is
// unsigned-less-than (bLo,bHi). Used for add carry-out and sub borrow-out.
func (g *generator) emit128PanicIfULess(aLo, aHi, bLo, bHi string, p panicMessage) {
	lbl := g.labels.NextLabel()
	g.emit("    cmp %s, %s", aHi, bHi)
	g.emit("    b.lo _sl_ovf128_%d", lbl) // aHi < bHi -> a < b
	g.emit("    b.hi _sl_ok128_%d", lbl)  // aHi > bHi -> a >= b
	g.emit("    cmp %s, %s", aLo, bLo)
	g.emit("    b.lo _sl_ovf128_%d", lbl) // hi equal, aLo < bLo -> a < b
	g.emit("    b _sl_ok128_%d", lbl)
	g.emit("_sl_ovf128_%d:", lbl)
	g.emitPanic(p)
	g.emit("_sl_ok128_%d:", lbl)
}

// emit128PanicIfSignBit panics with p when bit 63 of (p^q)&(r^s) is set — the
// two's-complement signed add/sub overflow test on the high words. For add the
// operands are (ahi,rhi,bhi,rhi); for sub (ahi,bhi,ahi,rhi).
func (g *generator) emit128PanicIfSignBit(p, q, r, s string, msg panicMessage) {
	lbl := g.labels.NextLabel()
	g.emit("    eor x16, %s, %s", p, q)
	g.emit("    eor x17, %s, %s", r, s)
	g.emit("    and x16, x16, x17")
	g.emit("    lsr x16, x16, #63")
	g.emit("    cbz x16, _sl_ok128_%d", lbl)
	g.emitPanic(msg)
	g.emit("_sl_ok128_%d:", lbl)
}

// gen128Add computes r = a + b (mod 2^128), trapping on overflow.
func (g *generator) gen128Add(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	g.loadValue128(v.Args[1], "x12", "x13")
	g.emit("    add x9, x10, x12")  // rlo = alo + blo
	g.emit("    add x14, x11, x13") // rhi = ahi + bhi (carry added below)
	g.emit("    cmp x9, x10")       // low add carried iff rlo < alo (unsigned)
	g.emit("    cset x15, lo")
	g.emit("    add x14, x14, x15") // rhi += carry
	if is128Signed(v.Type) {
		// Signed overflow iff a and b share a sign that differs from the result:
		// bit 63 of (ahi^rhi)&(bhi^rhi).
		g.emit128PanicIfSignBit("x11", "x14", "x13", "x14", PanicOverflowAdd)
	} else {
		// Unsigned overflow (carry out) iff result < a.
		g.emit128PanicIfULess("x9", "x14", "x10", "x11", PanicUnsignedOverAdd)
	}
	g.storeToStack128("x9", "x14", g.stackOffset(v))
	return nil
}

// gen128Sub computes r = a - b (mod 2^128), trapping on overflow/underflow.
func (g *generator) gen128Sub(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	g.loadValue128(v.Args[1], "x12", "x13")
	g.emit("    sub x9, x10, x12")  // rlo = alo - blo
	g.emit("    sub x14, x11, x13") // rhi = ahi - bhi (borrow subtracted below)
	g.emit("    cmp x10, x12")      // low sub borrowed iff alo < blo (unsigned)
	g.emit("    cset x15, lo")
	g.emit("    sub x14, x14, x15") // rhi -= borrow
	if is128Signed(v.Type) {
		// Signed overflow iff a and b have different signs and the result's
		// sign differs from a: bit 63 of (ahi^bhi)&(ahi^rhi).
		g.emit128PanicIfSignBit("x11", "x13", "x11", "x14", PanicOverflowSub)
	} else {
		// Unsigned underflow (borrow out) iff a < b.
		g.emit128PanicIfULess("x10", "x11", "x12", "x13", PanicUnsignedUnderSub)
	}
	g.storeToStack128("x9", "x14", g.stackOffset(v))
	return nil
}

// gen128Neg computes r = -a (mod 2^128), trapping on the signed INT_MIN case.
func (g *generator) gen128Neg(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	if is128Signed(v.Type) {
		// -a overflows only for a == INT_MIN (lo == 0, hi == 0x8000000000000000).
		lbl := g.labels.NextLabel()
		g.emit("    cbnz x10, _sl_negok128_%d", lbl) // lo != 0 -> not INT_MIN
		g.emit("    mov x16, #1")
		g.emit("    lsl x16, x16, #63") // 0x8000000000000000
		g.emit("    cmp x11, x16")
		g.emit("    b.ne _sl_negok128_%d", lbl)
		g.emitPanic(PanicOverflowNeg)
		g.emit("_sl_negok128_%d:", lbl)
	}
	g.emit("    neg x9, x10")  // rlo = -alo
	g.emit("    neg x14, x11") // rhi = -ahi (borrow subtracted below)
	g.emit("    cmp x10, #0")  // negating the low word borrows iff alo != 0
	g.emit("    cset x15, ne")
	g.emit("    sub x14, x14, x15")
	g.storeToStack128("x9", "x14", g.stackOffset(v))
	return nil
}

// gen128Mul computes the low 128 bits of a * b, trapping on overflow. The
// product and an overflow flag are computed by a leaf runtime helper (the
// overflow test, especially signed, is too long to inline per site); the panic
// itself is emitted here so it carries the enclosing function's name.
func (g *generator) gen128Mul(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x0", "x1")
	g.loadValue128(v.Args[1], "x2", "x3")
	signed := is128Signed(v.Type)
	if signed {
		g.emit("    bl _sl_s128_mul_ovf")
	} else {
		g.emit("    bl _sl_u128_mul_ovf")
	}
	lbl := g.labels.NextLabel()
	g.emit("    cbz x4, _sl_mulok128_%d", lbl) // x4 = overflow flag
	if signed {
		g.emitPanic(PanicOverflowMul)
	} else {
		g.emitPanic(PanicUnsignedOverMul)
	}
	g.emit("_sl_mulok128_%d:", lbl)
	g.storeToStack128("x0", "x1", g.stackOffset(v)) // product low 128 in x0:x1
	return nil
}

// gen128Cmp computes a relational comparison of two 128-bit operands, producing
// a 0/1 bool in x9. cond is the signed condition code (eq/ne/lt/le/gt/ge); the
// operand type selects signed vs unsigned ordering of the high word.
func (g *generator) gen128Cmp(v *ir.Value, cond string) error {
	g.loadValue128(v.Args[0], "x10", "x11") // a lo, hi
	g.loadValue128(v.Args[1], "x12", "x13") // b lo, hi
	unsigned := isUnsignedInt(v.Args[0].Type) || isUnsignedInt(v.Args[1].Type)

	switch cond {
	case "eq":
		g.emit("    cmp x11, x13")
		g.emit("    cset x9, eq")
		g.emit("    cmp x10, x12")
		g.emit("    cset x14, eq")
		g.emit("    and x9, x9, x14")
	case "ne":
		g.emit("    cmp x11, x13")
		g.emit("    cset x9, ne")
		g.emit("    cmp x10, x12")
		g.emit("    cset x14, ne")
		g.emit("    orr x9, x9, x14")
	default:
		// Ordering: the high words decide unless they are equal, in which case
		// the (always unsigned) low words decide. When the high words differ the
		// comparison is strict, so lt/le both reduce to a strict hi "lt" and
		// gt/ge to a strict hi "gt".
		var hiCond string
		switch cond {
		case "lt", "le":
			hiCond = "lt"
		case "gt", "ge":
			hiCond = "gt"
		}
		if unsigned {
			hiCond = unsignedCondCode(hiCond)
		}
		loCond := unsignedCondCode(cond) // low words compare unsigned
		lblEq := g.labels.NextLabel()
		lblDone := g.labels.NextLabel()
		g.emit("    cmp x11, x13")
		g.emit("    b.eq _sl_cmp128_lo_%d", lblEq)
		g.emit("    cset x9, %s", hiCond)
		g.emit("    b _sl_cmp128_done_%d", lblDone)
		g.emit("_sl_cmp128_lo_%d:", lblEq)
		g.emit("    cmp x10, x12")
		g.emit("    cset x9, %s", loCond)
		g.emit("_sl_cmp128_done_%d:", lblDone)
	}
	g.storeToStack("x9", g.stackOffset(v))
	return nil
}

// gen128DivMod computes a / b (isMod=false) or a % b (isMod=true) for 128-bit
// operands via the software divmod runtime helpers. Truncated division: the
// remainder takes the sign of the dividend.
func (g *generator) gen128DivMod(v *ir.Value, isMod bool) error {
	signed := false
	if it, ok := v.Type.(*ir.IntType); ok {
		signed = it.Signed
	}
	g.loadValue128(v.Args[0], "x0", "x1") // dividend lo, hi
	g.loadValue128(v.Args[1], "x2", "x3") // divisor lo, hi

	// Division/modulo by zero traps, as in the 64-bit path.
	label := g.labels.NextLabel()
	g.emit("    orr x9, x2, x3") // zero iff both words are zero
	g.emit("    cbnz x9, _sl_div128_ok_%d", label)
	if isMod {
		g.emitPanic(PanicModZero)
	} else {
		g.emitPanic(PanicDivZero)
	}
	g.emit("_sl_div128_ok_%d:", label)

	// Signed division overflow: INT_MIN / -1 has no representable quotient.
	// (Modulo of INT_MIN % -1 is 0, so it never overflows.)
	if signed && !isMod {
		ovf := g.labels.NextLabel()
		g.emit("    add x16, x2, #1")               // divisor == -1 iff both words are -1
		g.emit("    add x17, x3, #1")
		g.emit("    orr x16, x16, x17")
		g.emit("    cbnz x16, _sl_divof128_%d", ovf) // divisor != -1
		g.emit("    cbnz x0, _sl_divof128_%d", ovf)  // dividend lo != 0 -> not INT_MIN
		g.emit("    mov x16, #1")
		g.emit("    lsl x16, x16, #63") // 0x8000000000000000
		g.emit("    cmp x1, x16")
		g.emit("    b.ne _sl_divof128_%d", ovf)
		g.emitPanic(PanicOverflowDiv)
		g.emit("_sl_divof128_%d:", ovf)
	}

	if signed {
		g.emit("    bl _sl_s128_divmod")
	} else {
		g.emit("    bl _sl_u128_divmod")
	}
	// Quotient is returned in x0:x1, remainder in x2:x3.
	if isMod {
		g.storeToStack128("x2", "x3", g.stackOffset(v))
	} else {
		g.storeToStack128("x0", "x1", g.stackOffset(v))
	}
	return nil
}

// emitInt128Helpers emits the software 128-bit divmod runtime routines.
//
// _sl_u128_divmod: unsigned binary long division. In: x0:x1 dividend (lo:hi),
// x2:x3 divisor. Out: x0:x1 quotient, x2:x3 remainder. Leaf (no stack, no bl).
//
// _sl_s128_divmod: signed wrapper. Takes absolute values, calls the unsigned
// routine, then applies signs (quotient sign = dividend^divisor; remainder sign
// = dividend). Same in/out registers.
func (g *generator) emitInt128Helpers() {
	// ---- _sl_u128_divmod ----
	g.emit("// Unsigned 128-bit divmod: x0:x1 / x2:x3 -> quotient x0:x1, remainder x2:x3")
	g.emit("_sl_u128_divmod:")
	g.emit("    mov x4, #0")   // Rlo
	g.emit("    mov x5, #0")   // Rhi
	g.emit("    mov x6, #128") // bit counter
	g.emit("    mov x15, #1")  // quotient-bit constant
	g.emit("_sl_u128_divmod_loop:")
	g.emit("    lsr x7, x1, #63") // topbit = N's MSB
	// N <<= 1
	g.emit("    lsr x8, x0, #63")
	g.emit("    lsl x1, x1, #1")
	g.emit("    orr x1, x1, x8")
	g.emit("    lsl x0, x0, #1")
	// R = (R << 1) | topbit
	g.emit("    lsr x8, x4, #63")
	g.emit("    lsl x5, x5, #1")
	g.emit("    orr x5, x5, x8")
	g.emit("    lsl x4, x4, #1")
	g.emit("    orr x4, x4, x7")
	// if R >= D: R -= D; set quotient bit
	g.emit("    cmp x5, x3")
	g.emit("    b.hi _sl_u128_divmod_ge")
	g.emit("    b.lo _sl_u128_divmod_next")
	g.emit("    cmp x4, x2")
	g.emit("    b.lo _sl_u128_divmod_next")
	g.emit("_sl_u128_divmod_ge:")
	g.emit("    cmp x4, x2") // borrow iff Rlo < Dlo
	g.emit("    cset x14, lo")
	g.emit("    sub x4, x4, x2")
	g.emit("    sub x5, x5, x3")
	g.emit("    sub x5, x5, x14")
	g.emit("    orr x0, x0, x15") // quotient bit into N's LSB
	g.emit("_sl_u128_divmod_next:")
	g.emit("    subs x6, x6, #1")
	g.emit("    b.ne _sl_u128_divmod_loop")
	g.emit("    mov x2, x4") // remainder lo
	g.emit("    mov x3, x5") // remainder hi
	g.emit("    ret")
	g.emit("")

	// ---- _sl_s128_divmod ----
	g.emit("// Signed 128-bit divmod: x0:x1 / x2:x3 -> quotient x0:x1, remainder x2:x3")
	g.emit("_sl_s128_divmod:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    asr x9, x1, #63")  // dividend sign (0 or -1)
	g.emit("    asr x10, x3, #63") // divisor sign
	g.emit("    eor x11, x9, x10") // quotient sign
	g.emit("    stp x11, x9, [sp, #-16]!")
	// abs(dividend)
	g.emit("    cbz x9, _sl_s128_absd_done")
	g.emit("    cmp x0, #0")
	g.emit("    cset x14, ne")
	g.emit("    neg x0, x0")
	g.emit("    neg x1, x1")
	g.emit("    sub x1, x1, x14")
	g.emit("_sl_s128_absd_done:")
	// abs(divisor)
	g.emit("    cbz x10, _sl_s128_absv_done")
	g.emit("    cmp x2, #0")
	g.emit("    cset x14, ne")
	g.emit("    neg x2, x2")
	g.emit("    neg x3, x3")
	g.emit("    sub x3, x3, x14")
	g.emit("_sl_s128_absv_done:")
	g.emit("    bl _sl_u128_divmod")
	g.emit("    ldp x11, x9, [sp], #16") // restore quotient sign, dividend sign
	// negate quotient if signs differ
	g.emit("    cbz x11, _sl_s128_negq_done")
	g.emit("    cmp x0, #0")
	g.emit("    cset x14, ne")
	g.emit("    neg x0, x0")
	g.emit("    neg x1, x1")
	g.emit("    sub x1, x1, x14")
	g.emit("_sl_s128_negq_done:")
	// negate remainder if dividend was negative
	g.emit("    cbz x9, _sl_s128_negr_done")
	g.emit("    cmp x2, #0")
	g.emit("    cset x14, ne")
	g.emit("    neg x2, x2")
	g.emit("    neg x3, x3")
	g.emit("    sub x3, x3, x14")
	g.emit("_sl_s128_negr_done:")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// ---- _sl_u128_mul_ovf (x0:x1 * x2:x3) -> product x0:x1, overflow flag x4 ----
	// Computes the low 128 bits of the product and whether the true 256-bit
	// product exceeds 128 bits. Overflow iff any of: a carry out of the
	// bits[64,127] column, the high halves of alo*bhi or ahi*blo are nonzero, or
	// both ahi and bhi are nonzero (which alone forces bits >= 128). Leaf.
	g.emit("// Unsigned 128-bit multiply with overflow flag")
	g.emit("_sl_u128_mul_ovf:")
	g.emit("    mul x5, x0, x2")   // product low word
	g.emit("    umulh x6, x0, x2") // lo_hi
	g.emit("    mul x7, x0, x3")   // m1_lo
	g.emit("    umulh x8, x0, x3") // m1_hi
	g.emit("    mul x9, x1, x2")   // m2_lo
	g.emit("    umulh x10, x1, x2") // m2_hi
	g.emit("    add x11, x6, x7")  // s1 = lo_hi + m1_lo
	g.emit("    cmp x11, x7")
	g.emit("    cset x12, lo") // carry of first add
	g.emit("    add x13, x11, x9") // product high word = s1 + m2_lo
	g.emit("    cmp x13, x9")
	g.emit("    cset x14, lo")     // carry of second add
	g.emit("    add x12, x12, x14") // c1 = total carry out of bits[64,127]
	g.emit("    mov x4, #0")
	g.emit("    cbnz x12, _sl_u128mulovf_yes")
	g.emit("    cbnz x8, _sl_u128mulovf_yes")  // m1_hi != 0
	g.emit("    cbnz x10, _sl_u128mulovf_yes") // m2_hi != 0
	g.emit("    cbz x1, _sl_u128mulovf_done")  // ahi == 0
	g.emit("    cbz x3, _sl_u128mulovf_done")  // bhi == 0
	g.emit("_sl_u128mulovf_yes:")
	g.emit("    mov x4, #1")
	g.emit("_sl_u128mulovf_done:")
	g.emit("    mov x0, x5")  // product low
	g.emit("    mov x1, x13") // product high
	g.emit("    ret")
	g.emit("")

	// ---- _sl_s128_mul_ovf (x0:x1 * x2:x3) -> product x0:x1, overflow flag x4 ----
	// Signs give the result sign; magnitudes multiply as unsigned (|INT_MIN| is
	// 2^127, exactly its unsigned bit pattern). Overflow iff the magnitude needs
	// > 128 bits, or is > 2^127, or is exactly 2^127 with a positive result
	// (only -2^127 is representable). The low-128 product is sign-agnostic, so it
	// is recomputed from the original operands. Leaf (avoids x18, reserved).
	g.emit("// Signed 128-bit multiply with overflow flag")
	g.emit("_sl_s128_mul_ovf:")
	g.emit("    asr x9, x1, #63")  // sign(a)
	g.emit("    asr x10, x3, #63") // sign(b)
	g.emit("    eor x11, x9, x10") // result sign (0 or -1)
	// |a| in x5:x6
	g.emit("    mov x5, x0")
	g.emit("    mov x6, x1")
	g.emit("    cbz x9, _sl_s128mul_adone")
	g.emit("    neg x5, x0")
	g.emit("    neg x6, x1")
	g.emit("    cmp x0, #0")
	g.emit("    cset x12, ne")
	g.emit("    sub x6, x6, x12")
	g.emit("_sl_s128mul_adone:")
	// |b| in x7:x8
	g.emit("    mov x7, x2")
	g.emit("    mov x8, x3")
	g.emit("    cbz x10, _sl_s128mul_bdone")
	g.emit("    neg x7, x2")
	g.emit("    neg x8, x3")
	g.emit("    cmp x2, #0")
	g.emit("    cset x12, ne")
	g.emit("    sub x8, x8, x12")
	g.emit("_sl_s128mul_bdone:")
	// magnitude product: Mlo=x13, Mhi=x14; m1_hi=x15, m2_hi=x16 held for the test
	g.emit("    mul x13, x5, x7")
	g.emit("    umulh x9, x5, x7") // lo_hi
	g.emit("    mul x10, x5, x8")  // m1_lo
	g.emit("    umulh x15, x5, x8") // m1_hi
	g.emit("    mul x12, x6, x7")  // m2_lo
	g.emit("    umulh x16, x6, x7") // m2_hi
	g.emit("    add x9, x9, x10")  // s1 = lo_hi + m1_lo
	g.emit("    cmp x9, x10")
	g.emit("    cset x10, lo") // carry1
	g.emit("    add x14, x9, x12") // Mhi = s1 + m2_lo
	g.emit("    cmp x14, x12")
	g.emit("    cset x17, lo")     // carry2
	g.emit("    add x10, x10, x17") // c1
	g.emit("    mov x4, #0")
	g.emit("    cbnz x10, _sl_s128mul_magovf")
	g.emit("    cbnz x15, _sl_s128mul_magovf")
	g.emit("    cbnz x16, _sl_s128mul_magovf")
	g.emit("    cbz x6, _sl_s128mul_magfit")
	g.emit("    cbz x8, _sl_s128mul_magfit")
	g.emit("_sl_s128mul_magovf:")
	g.emit("    mov x4, #1")
	g.emit("    b _sl_s128mul_result")
	g.emit("_sl_s128mul_magfit:")
	// M fits 128 bits (x13 lo, x14 hi). Check bit 127.
	g.emit("    lsr x9, x14, #63")
	g.emit("    cbz x9, _sl_s128mul_result") // M < 2^127 -> fits, flag stays 0
	g.emit("    cbnz x13, _sl_s128mul_setovf") // lo != 0 -> M > 2^127
	g.emit("    mov x9, #1")
	g.emit("    lsl x9, x9, #63")
	g.emit("    cmp x14, x9")
	g.emit("    b.ne _sl_s128mul_setovf") // hi != 2^127 -> M > 2^127
	// M == 2^127 exactly: valid only if the result is negative (-2^127).
	g.emit("    cbnz x11, _sl_s128mul_result") // negative -> ok
	g.emit("_sl_s128mul_setovf:")
	g.emit("    mov x4, #1")
	g.emit("_sl_s128mul_result:")
	// Product low 128 from the original operands (sign-agnostic low bits).
	g.emit("    mul x5, x0, x2")
	g.emit("    umulh x6, x0, x2")
	g.emit("    mul x7, x0, x3")
	g.emit("    add x6, x6, x7")
	g.emit("    mul x7, x1, x2")
	g.emit("    add x6, x6, x7")
	g.emit("    mov x0, x5")
	g.emit("    mov x1, x6")
	g.emit("    ret")
	g.emit("")

	// ---- _sl_print_u128 (x0:x1 = value) ----
	// Extract decimal digits by repeated division by 10. The buffer pointer
	// (x19) and digit count (x20) live in callee-saved registers because the
	// divmod helper is called each iteration; it preserves x9..x28.
	g.emit("// Print unsigned 128-bit integer (x0:x1 = value lo:hi)")
	g.emit("_sl_print_u128:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    sub sp, sp, #48") // digit buffer (u128 has at most 39 digits)
	g.emit("    mov x19, sp")
	g.emit("    add x19, x19, #46") // write digits backward
	g.emit("    mov x20, #0")       // digit count
	g.emit("_sl_print_u128_loop:")
	g.emit("    mov x2, #10")
	g.emit("    mov x3, #0")
	g.emit("    bl _sl_u128_divmod") // q -> x0:x1, remainder digit -> x2
	g.emit("    add x4, x2, #48")    // digit to ASCII
	g.emit("    strb w4, [x19]")
	g.emit("    sub x19, x19, #1")
	g.emit("    add x20, x20, #1")
	g.emit("    orr x4, x0, x1") // quotient nonzero?
	g.emit("    cbnz x4, _sl_print_u128_loop")
	g.emit("    add x19, x19, #1")
	g.emit("    mov x0, #1")  // stdout
	g.emit("    mov x1, x19") // buffer
	g.emit("    mov x2, x20") // length
	g.emit("    mov x16, #4") // write syscall
	g.emit("    svc #0")
	g.emit("    adrp x1, _sl_newline@PAGE")
	g.emit("    add x1, x1, _sl_newline@PAGEOFF")
	g.emit("    mov x0, #1")
	g.emit("    mov x2, #1")
	g.emit("    mov x16, #4")
	g.emit("    svc #0")
	g.emit("    add sp, sp, #48")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// ---- _sl_print_s128 (x0:x1 = value) ----
	g.emit("// Print signed 128-bit integer (x0:x1 = value lo:hi)")
	g.emit("_sl_print_s128:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    stp x21, x22, [sp, #-16]!")
	g.emit("    sub sp, sp, #48")
	g.emit("    mov x19, sp")
	g.emit("    add x19, x19, #46")
	g.emit("    mov x20, #0")
	g.emit("    mov x21, #0") // negative flag
	// Take absolute value if negative.
	g.emit("    asr x4, x1, #63")
	g.emit("    cbz x4, _sl_print_s128_absdone")
	g.emit("    mov x21, #1")
	g.emit("    cmp x0, #0")
	g.emit("    cset x4, ne")
	g.emit("    neg x0, x0")
	g.emit("    neg x1, x1")
	g.emit("    sub x1, x1, x4")
	g.emit("_sl_print_s128_absdone:")
	g.emit("_sl_print_s128_loop:")
	g.emit("    mov x2, #10")
	g.emit("    mov x3, #0")
	g.emit("    bl _sl_u128_divmod")
	g.emit("    add x4, x2, #48")
	g.emit("    strb w4, [x19]")
	g.emit("    sub x19, x19, #1")
	g.emit("    add x20, x20, #1")
	g.emit("    orr x4, x0, x1")
	g.emit("    cbnz x4, _sl_print_s128_loop")
	// Prepend '-' if negative.
	g.emit("    cbz x21, _sl_print_s128_write")
	g.emit("    mov x4, #45")
	g.emit("    strb w4, [x19]")
	g.emit("    sub x19, x19, #1")
	g.emit("    add x20, x20, #1")
	g.emit("_sl_print_s128_write:")
	g.emit("    add x19, x19, #1")
	g.emit("    mov x0, #1")
	g.emit("    mov x1, x19")
	g.emit("    mov x2, x20")
	g.emit("    mov x16, #4")
	g.emit("    svc #0")
	g.emit("    adrp x1, _sl_newline@PAGE")
	g.emit("    add x1, x1, _sl_newline@PAGEOFF")
	g.emit("    mov x0, #1")
	g.emit("    mov x2, #1")
	g.emit("    mov x16, #4")
	g.emit("    svc #0")
	g.emit("    add sp, sp, #48")
	g.emit("    ldp x21, x22, [sp], #16")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// ---- _sl_u128_to_str (x0:x1 = value -> x0 = heap string) ----
	// Builds a length-prefixed heap string {.quad len; bytes} for interpolation,
	// mirroring _sl_int_to_str but extracting digits via _sl_u128_divmod. The
	// buffer pointer (x19) and count (x20) are callee-saved because both the
	// divmod and heap-alloc calls run inside; x21 holds the source pointer.
	g.emit("// Unsigned 128-bit to heap string (x0:x1 = value) -> x0 = ptr to {len; bytes}")
	g.emit("_sl_u128_to_str:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    stp x21, x22, [sp, #-16]!")
	g.emit("    sub sp, sp, #48")
	g.emit("    mov x19, sp")
	g.emit("    add x19, x19, #46")
	g.emit("    mov x20, #0")
	g.emit("_sl_u128_to_str_loop:")
	g.emit("    mov x2, #10")
	g.emit("    mov x3, #0")
	g.emit("    bl _sl_u128_divmod")
	g.emit("    add x4, x2, #48")
	g.emit("    strb w4, [x19]")
	g.emit("    sub x19, x19, #1")
	g.emit("    add x20, x20, #1")
	g.emit("    orr x4, x0, x1")
	g.emit("    cbnz x4, _sl_u128_to_str_loop")
	g.emit("    add x19, x19, #1") // first char
	g.emit("    mov x21, x19")     // save source pointer
	g.emit("    add x0, x20, #8")  // alloc len + 8-byte header
	g.emit("    bl _sl_heap_alloc")
	g.emit("    str x20, [x0]")  // length header
	g.emit("    add x9, x0, #8") // dest cursor
	g.emit("    mov x10, x21")   // src cursor
	g.emit("    mov x11, x20")   // remaining
	g.emit("_sl_u128_to_str_copy:")
	g.emit("    cbz x11, _sl_u128_to_str_done")
	g.emit("    ldrb w12, [x10]")
	g.emit("    strb w12, [x9]")
	g.emit("    add x10, x10, #1")
	g.emit("    add x9, x9, #1")
	g.emit("    sub x11, x11, #1")
	g.emit("    b _sl_u128_to_str_copy")
	g.emit("_sl_u128_to_str_done:")
	g.emit("    add sp, sp, #48")
	g.emit("    ldp x21, x22, [sp], #16")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")

	// ---- _sl_s128_to_str (x0:x1 = value -> x0 = heap string) ----
	g.emit("// Signed 128-bit to heap string (x0:x1 = value) -> x0 = ptr to {len; bytes}")
	g.emit("_sl_s128_to_str:")
	g.emit("    stp x29, x30, [sp, #-16]!")
	g.emit("    mov x29, sp")
	g.emit("    stp x19, x20, [sp, #-16]!")
	g.emit("    stp x21, x22, [sp, #-16]!")
	g.emit("    sub sp, sp, #48")
	g.emit("    mov x19, sp")
	g.emit("    add x19, x19, #46")
	g.emit("    mov x20, #0")
	g.emit("    mov x22, #0") // negative flag
	g.emit("    asr x4, x1, #63")
	g.emit("    cbz x4, _sl_s128_to_str_absdone")
	g.emit("    mov x22, #1")
	g.emit("    cmp x0, #0")
	g.emit("    cset x4, ne")
	g.emit("    neg x0, x0")
	g.emit("    neg x1, x1")
	g.emit("    sub x1, x1, x4")
	g.emit("_sl_s128_to_str_absdone:")
	g.emit("_sl_s128_to_str_loop:")
	g.emit("    mov x2, #10")
	g.emit("    mov x3, #0")
	g.emit("    bl _sl_u128_divmod")
	g.emit("    add x4, x2, #48")
	g.emit("    strb w4, [x19]")
	g.emit("    sub x19, x19, #1")
	g.emit("    add x20, x20, #1")
	g.emit("    orr x4, x0, x1")
	g.emit("    cbnz x4, _sl_s128_to_str_loop")
	g.emit("    cbz x22, _sl_s128_to_str_alloc")
	g.emit("    mov x4, #45") // '-'
	g.emit("    strb w4, [x19]")
	g.emit("    sub x19, x19, #1")
	g.emit("    add x20, x20, #1")
	g.emit("_sl_s128_to_str_alloc:")
	g.emit("    add x19, x19, #1")
	g.emit("    mov x21, x19")
	g.emit("    add x0, x20, #8")
	g.emit("    bl _sl_heap_alloc")
	g.emit("    str x20, [x0]")
	g.emit("    add x9, x0, #8")
	g.emit("    mov x10, x21")
	g.emit("    mov x11, x20")
	g.emit("_sl_s128_to_str_copy:")
	g.emit("    cbz x11, _sl_s128_to_str_done")
	g.emit("    ldrb w12, [x10]")
	g.emit("    strb w12, [x9]")
	g.emit("    add x10, x10, #1")
	g.emit("    add x9, x9, #1")
	g.emit("    sub x11, x11, #1")
	g.emit("    b _sl_s128_to_str_copy")
	g.emit("_sl_s128_to_str_done:")
	g.emit("    add sp, sp, #48")
	g.emit("    ldp x21, x22, [sp], #16")
	g.emit("    ldp x19, x20, [sp], #16")
	g.emit("    ldp x29, x30, [sp], #16")
	g.emit("    ret")
	g.emit("")
}

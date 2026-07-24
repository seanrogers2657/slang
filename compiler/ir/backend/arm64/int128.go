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

// gen128Add computes r = a + b (mod 2^128).
func (g *generator) gen128Add(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	g.loadValue128(v.Args[1], "x12", "x13")
	g.emit("    add x9, x10, x12")  // rlo = alo + blo
	g.emit("    add x14, x11, x13") // rhi = ahi + bhi (carry added below)
	g.emit("    cmp x9, x10")       // low add carried iff rlo < alo (unsigned)
	g.emit("    cset x15, lo")
	g.emit("    add x14, x14, x15") // rhi += carry
	g.storeToStack128("x9", "x14", g.stackOffset(v))
	return nil
}

// gen128Sub computes r = a - b (mod 2^128).
func (g *generator) gen128Sub(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	g.loadValue128(v.Args[1], "x12", "x13")
	g.emit("    sub x9, x10, x12")  // rlo = alo - blo
	g.emit("    sub x14, x11, x13") // rhi = ahi - bhi (borrow subtracted below)
	g.emit("    cmp x10, x12")      // low sub borrowed iff alo < blo (unsigned)
	g.emit("    cset x15, lo")
	g.emit("    sub x14, x14, x15") // rhi -= borrow
	g.storeToStack128("x9", "x14", g.stackOffset(v))
	return nil
}

// gen128Neg computes r = -a (mod 2^128).
func (g *generator) gen128Neg(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	g.emit("    neg x9, x10")  // rlo = -alo
	g.emit("    neg x14, x11") // rhi = -ahi (borrow subtracted below)
	g.emit("    cmp x10, #0")  // negating the low word borrows iff alo != 0
	g.emit("    cset x15, ne")
	g.emit("    sub x14, x14, x15")
	g.storeToStack128("x9", "x14", g.stackOffset(v))
	return nil
}

// gen128Mul computes the low 128 bits of a * b. The low 128 bits of the product
// are identical for signed and unsigned operands, so umulh serves both.
func (g *generator) gen128Mul(v *ir.Value) error {
	g.loadValue128(v.Args[0], "x10", "x11")
	g.loadValue128(v.Args[1], "x12", "x13")
	g.emit("    mul x9, x10, x12")    // rlo = low(alo * blo)
	g.emit("    umulh x14, x10, x12") // rhi = high(alo * blo)
	g.emit("    mul x15, x10, x13")   // + low(alo * bhi)
	g.emit("    add x14, x14, x15")
	g.emit("    mul x15, x11, x12") // + low(ahi * blo)
	g.emit("    add x14, x14, x15")
	g.storeToStack128("x9", "x14", g.stackOffset(v))
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

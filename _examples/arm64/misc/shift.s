// @test: exit_code=40
// shift.s - Demonstrate shift instructions (lsl, lsr, asr)
// Calculates: ((10 << 2) >> 1) = (40 >> 1) = 20, then 20 << 1 = 40

.global _start
.align 4

_start:
    // Start with 10
    mov x0, #10

    // LSL: 10 << 2 = 40
    lsl x0, x0, #2

    // LSR: 40 >> 1 = 20
    lsr x0, x0, #1

    // LSL: 20 << 1 = 40 (final result)
    lsl x0, x0, #1

    // Exit with result as exit code
    mov x16, #1
    svc #0

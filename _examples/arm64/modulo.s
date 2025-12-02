// modulo.s - Test modulo operation using udiv and msub
// Tests: udiv, msub (compute remainder)
// Expected exit code: 7 (47 % 10 = 7)

.global _start
.align 4
_start:
    // Compute 47 % 10 using udiv and msub
    // msub Xd, Xn, Xm, Xa computes: Xd = Xa - (Xn * Xm)
    // So: remainder = dividend - (quotient * divisor)

    mov x0, #47         // dividend
    mov x1, #10         // divisor

    udiv x2, x0, x1     // x2 = 47 / 10 = 4 (quotient)
    msub x3, x2, x1, x0 // x3 = x0 - (x2 * x1) = 47 - (4 * 10) = 7

    // Exit with remainder
    mov x0, x3          // exit code = remainder
    mov x16, #1         // syscall: exit
    svc #0

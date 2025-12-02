// division.s - Test unsigned and signed division instructions
// Tests: udiv, sdiv
// Expected exit code: 5 (100 / 20 = 5)

.global _start
.align 4
_start:
    // Test unsigned division: 100 / 20 = 5
    mov x0, #100        // dividend
    mov x1, #20         // divisor
    udiv x2, x0, x1     // x2 = 100 / 20 = 5

    // Exit with result
    mov x0, x2          // exit code = result
    mov x16, #1         // syscall: exit
    svc #0

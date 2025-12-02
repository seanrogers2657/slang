// @test: exit_code=42
.global _start
.align 4

_start:
    // Negate a number: -42, then negate again to get 42
    mov x0, #42      // Original value
    neg x1, x0       // x1 = -42 (0 - 42)

    // Negate again to get positive (for valid exit code)
    neg x0, x1       // x0 = 42

    // Exit with the result
    mov x16, #1      // syscall number for exit
    svc #0           // Make syscall

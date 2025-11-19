.global _start
.align 4

_start:
    // Subtract two numbers: 50 - 25 = 25
    mov x0, #50      // First operand
    mov x1, #25      // Second operand
    sub x2, x0, x1   // Result in x2 (25)

    // Exit with the result
    mov x0, x2       // Move result to exit code
    mov x16, #1      // syscall number for exit
    svc #0           // Make syscall

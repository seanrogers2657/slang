// @test: exit_code=15
.global _start
.align 4

_start:
    // Multiply two numbers: 5 * 3 = 15
    mov x0, #5       // First operand
    mov x1, #3       // Second operand
    mul x2, x0, x1   // Result in x2 (15)

    // Exit with the result
    mov x0, x2       // Move result to exit code
    mov x16, #1      // syscall number for exit
    svc #0           // Make syscall

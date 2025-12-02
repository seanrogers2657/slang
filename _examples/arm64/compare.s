// @test: exit_code=1
.global _start
.align 4

_start:
    // Compare two numbers: is 10 < 20?
    mov x0, #10      // First operand
    mov x1, #20      // Second operand
    cmp x0, x1       // Compare x0 with x1
    cset x2, lt      // Set x2 to 1 if less than, 0 otherwise

    // Exit with the result (should be 1)
    mov x0, x2       // Move result to exit code
    mov x16, #1      // syscall number for exit
    svc #0           // Make syscall

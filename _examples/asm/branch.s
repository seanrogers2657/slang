.global _start
.align 4

_start:
    b main           // Branch to main

main:
    // Simple computation with a branch
    mov x0, #7       // First number
    mov x1, #3       // Second number
    add x2, x0, x1   // Add them (result: 10)

    // Exit with the result
    mov x0, x2       // Move result to exit code
    mov x16, #1      // syscall number for exit
    svc #0           // Make syscall

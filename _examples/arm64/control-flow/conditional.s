// @test: exit_code=1
// conditional.s - Demonstrate conditional execution and branches
// Shows cmp, cset, and conditional branches (beq, bne, blt, etc.)

.global _start
.align 4

_start:
    // Compare two numbers
    mov x0, #10
    mov x1, #20
    cmp x0, x1              // Compare x0 and x1

    // Conditional set
    cset x2, eq             // x2 = 1 if equal, 0 otherwise
    cset x3, lt             // x3 = 1 if less than, 0 otherwise
    cset x4, gt             // x4 = 1 if greater than, 0 otherwise

    // Conditional branch (branch if equal)
    mov x5, #5
    mov x6, #5
    cmp x5, x6
    beq equal_label         // Branch if equal
    mov x7, #99             // Should not execute
    b end

equal_label:
    mov x7, #100            // Should execute

end:
    // Test less than branch
    mov x8, #3
    mov x9, #7
    cmp x8, x9
    blt less_than_label     // Branch if x8 < x9
    mov x10, #0
    b exit

less_than_label:
    mov x10, #1             // Should execute

exit:
    // Exit with x10 as exit code (should be 1)
    mov x0, x10
    mov x16, #1
    svc #0

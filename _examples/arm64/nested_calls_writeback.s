// nested_calls_writeback.s - Nested function calls with stack frame management
// Demonstrates proper use of pre/post-indexed addressing for callee-saved registers

.global _start
.align 4

_start:
    mov x0, #5
    bl compute          // compute(5) should return 5 * 2 + 3 = 13
    mov x16, #1
    svc #0

// Function: compute
// Returns: (x0 * 2) + 3
// Uses nested calls to demonstrate stack frame management
.align 4
compute:
    // Prologue: save link register and frame pointer (pre-indexed)
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    // Save callee-saved register we'll use (pre-indexed)
    stp x19, x20, [sp, #-16]!

    // Save input in callee-saved register
    mov x19, x0

    // Call double(x0)
    bl double
    mov x20, x0         // Save result

    // Call add_three(result)
    mov x0, x20
    bl add_three

    // Epilogue: restore callee-saved registers (post-indexed)
    ldp x19, x20, [sp], #16

    // Restore link register and frame pointer (post-indexed)
    ldp x29, x30, [sp], #16
    ret

// Function: double
// Returns: x0 * 2
.align 4
double:
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    add x0, x0, x0      // x0 = x0 * 2

    ldp x29, x30, [sp], #16
    ret

// Function: add_three
// Returns: x0 + 3
.align 4
add_three:
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    add x0, x0, #3      // x0 = x0 + 3

    ldp x29, x30, [sp], #16
    ret

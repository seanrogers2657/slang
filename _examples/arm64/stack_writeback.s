// stack_writeback.s - Demonstrate pre-indexed and post-indexed addressing modes
// These are essential for stack operations (push/pop patterns)
//
// Pre-indexed:  [sp, #-16]!  - First adjusts sp, then accesses memory (push)
// Post-indexed: [sp], #16    - First accesses memory, then adjusts sp (pop)

.global _start
.align 4

_start:
    // Initialize test value
    mov x0, #10

    // Push x0 and x1 to stack using pre-indexed addressing
    // This is equivalent to: sp = sp - 16; store x0,x1 at sp
    mov x1, #20
    stp x0, x1, [sp, #-16]!

    // Modify registers to prove we restore from stack
    mov x0, #0
    mov x1, #0

    // Pop x0 and x1 from stack using post-indexed addressing
    // This is equivalent to: load x0,x1 from sp; sp = sp + 16
    ldp x0, x1, [sp], #16

    // x0 should be 10, x1 should be 20
    // Return x0 + x1 = 30 as exit code
    add x0, x0, x1
    mov x16, #1
    svc #0

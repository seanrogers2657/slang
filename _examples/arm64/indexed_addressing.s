// indexed_addressing.s - Test pre-indexed and post-indexed addressing modes
// Tests: str [base, #offset]!, ldr [base], #offset (writeback modes)
// Expected exit code: 42

.global _start
.align 4
_start:
    // Set up frame
    stp x29, x30, [sp, #-16]!   // Pre-indexed: decrement sp, then store
    mov x29, sp

    // Test pre-indexed store: str [base, #offset]!
    mov x0, #42
    str x0, [sp, #-16]!         // Push x0 onto stack (pre-indexed)

    // Test post-indexed load: ldr [base], #offset
    ldr x1, [sp], #16           // Pop into x1, then increment sp (post-indexed)

    // Restore frame
    ldp x29, x30, [sp], #16     // Post-indexed: load, then increment sp

    // Exit with loaded value (should be 42)
    mov x0, x1
    mov x16, #1
    svc #0

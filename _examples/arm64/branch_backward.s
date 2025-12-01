; Test backward branch (simple counted loop)
; Expected exit code: 5
; Counts down from 5 to 0 using backward branch
.global _start

_start:
    mov x0, #5          ; Counter starts at 5
    mov x1, #0          ; Accumulator
loop:
    add x1, x1, #1      ; Increment accumulator
    sub x0, x0, #1      ; Decrement counter
    cmp x0, #0          ; Compare counter to 0
    b.ne loop           ; Branch back if not zero
    mov x0, x1          ; Move result to x0 (exit code)
    mov x16, #1
    svc #0

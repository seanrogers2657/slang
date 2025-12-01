; Test function with stack frame (stp/ldp for frame pointer)
; Expected exit code: 100
; Simulates a function call with proper prologue/epilogue
.global _start

_start:
    mov x0, #50
    bl multiply_by_two
    mov x16, #1
    svc #0

multiply_by_two:
    ; Function prologue - save frame pointer and link register
    stp x29, x30, [sp]

    ; Function body: multiply x0 by 2
    add x0, x0, x0      ; x0 = x0 * 2 = 100

    ; Function epilogue - restore frame pointer and link register
    ldp x29, x30, [sp]
    ret

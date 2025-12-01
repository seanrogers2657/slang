; Test stack operations with offsets
; Expected exit code: 10
; Store two values and add them
.global _start

_start:
    mov x0, #3          ; First value
    str x0, [sp]        ; Store at [sp]
    mov x0, #7          ; Second value
    str x0, [sp, #8]    ; Store at [sp+8]

    ldr x1, [sp]        ; Load first value
    ldr x2, [sp, #8]    ; Load second value
    add x0, x1, x2      ; Add them: 3 + 7 = 10

    mov x16, #1
    svc #0

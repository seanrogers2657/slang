; @test: exit_code=7
; Test various memory offsets
; Store 1,2,4 at different offsets, sum them = 7
.global _start

_start:
    mov x0, #1
    str x0, [sp]        ; [sp+0] = 1
    mov x0, #2
    str x0, [sp, #8]    ; [sp+8] = 2
    mov x0, #4
    str x0, [sp, #16]   ; [sp+16] = 4

    ldr x1, [sp]        ; x1 = 1
    ldr x2, [sp, #8]    ; x2 = 2
    ldr x3, [sp, #16]   ; x3 = 4

    add x0, x1, x2      ; x0 = 1 + 2 = 3
    add x0, x0, x3      ; x0 = 3 + 4 = 7

    mov x16, #1
    svc #0

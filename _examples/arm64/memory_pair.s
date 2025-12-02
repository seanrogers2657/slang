; @test: exit_code=30
; Test pair memory operations (stp/ldp)
; Store (10, 20) and load back, then add
.global _start

_start:
    mov x0, #10         ; First value
    mov x1, #20         ; Second value
    stp x0, x1, [sp]    ; Store pair to stack

    mov x0, #0          ; Clear x0
    mov x1, #0          ; Clear x1

    ldp x2, x3, [sp]    ; Load pair from stack
    add x0, x2, x3      ; Add them: 10 + 20 = 30

    mov x16, #1
    svc #0

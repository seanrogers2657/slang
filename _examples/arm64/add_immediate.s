; @test: exit_code=15
; Add register and immediate: 10 + 5 = 15
.global _start

_start:
    mov x0, #10
    add x0, x0, #5
    mov x16, #1
    svc #0

; @test: exit_code=30
; Add two registers: 10 + 20 = 30
.global _start

_start:
    mov x0, #10
    mov x1, #20
    add x0, x0, x1
    mov x16, #1
    svc #0

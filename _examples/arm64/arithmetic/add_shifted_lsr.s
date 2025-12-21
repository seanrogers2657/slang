; @test: exit_code=34
; Test add with LSR (logical shift right): x0 + (x1 >> 2) = 10 + (96 >> 2) = 10 + 24 = 34
.global _start

_start:
    mov x0, #10
    mov x1, #96
    add x0, x0, x1, lsr #2    ; x0 = x0 + (x1 >> 2) = 10 + 24 = 34
    mov x16, #1
    svc #0

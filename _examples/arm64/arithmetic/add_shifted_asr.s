; @test: exit_code=18
; Test add with ASR (arithmetic shift right): x0 + (x1 >> 1) = 10 + (16 >> 1) = 10 + 8 = 18
.global _start

_start:
    mov x0, #10
    mov x1, #16
    add x0, x0, x1, asr #1    ; x0 = x0 + (x1 >> 1) = 10 + 8 = 18
    mov x16, #1
    svc #0

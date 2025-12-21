; @test: exit_code=42
; Test add with shifted register: x0 + (x1 << 2) = 10 + (8 << 2) = 10 + 32 = 42
.global _start

_start:
    mov x0, #10
    mov x1, #8
    add x0, x0, x1, lsl #2    ; x0 = x0 + (x1 << 2) = 10 + 32 = 42
    mov x16, #1
    svc #0

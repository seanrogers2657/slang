; @test: exit_code=8
; Test BIC instruction: 0x0F BIC 0x07 = 0x0F & ~0x07 = 0x0F & 0xF8 = 0x08
.global _start

_start:
    mov x0, #0x0F       ; x0 = 15 (0x0F = 1111)
    mov x1, #0x07       ; x1 = 7  (0x07 = 0111)
    bic x0, x0, x1      ; x0 = x0 & ~x1 = 1111 & 1000 = 1000 (8)
    mov x16, #1         ; syscall: exit
    svc #0

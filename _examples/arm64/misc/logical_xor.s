; @test: exit_code=6
; Test EOR instruction: 0x0F ^ 0x09 = 0x06
.global _start

_start:
    mov x0, #0x0F       ; x0 = 15 (0x0F = 1111)
    mov x1, #0x09       ; x1 = 9  (0x09 = 1001)
    eor x0, x0, x1      ; x0 = 0x0F ^ 0x09 = 0x06 (0110 = 6)
    mov x16, #1         ; syscall: exit
    svc #0

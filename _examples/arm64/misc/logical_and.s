; @test: exit_code=2
; Test AND instruction: 0xFF & 0x0F = 0x0F, then & 0x02 = 0x02
.global _start

_start:
    mov x0, #0xFF       ; x0 = 255 (0xFF)
    mov x1, #0x0F       ; x1 = 15 (0x0F)
    and x0, x0, x1      ; x0 = 0xFF & 0x0F = 0x0F (15)
    mov x2, #0x02       ; x2 = 2
    and x0, x0, x2      ; x0 = 0x0F & 0x02 = 0x02 (2)
    mov x16, #1         ; syscall: exit
    svc #0

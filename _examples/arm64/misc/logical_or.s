; @test: exit_code=7
; Test ORR instruction: 0x04 | 0x03 = 0x07
.global _start

_start:
    mov x0, #0x04       ; x0 = 4
    mov x1, #0x03       ; x1 = 3
    orr x0, x0, x1      ; x0 = 0x04 | 0x03 = 0x07 (7)
    mov x16, #1         ; syscall: exit
    svc #0

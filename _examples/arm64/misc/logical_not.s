; @test: exit_code=240
; Test MVN instruction: NOT 0x0F = 0xFFFFFFFFFFFFFFF0, mask to 0xF0 = 240
.global _start

_start:
    mov x0, #0x0F       ; x0 = 15 (0x0F)
    mvn x1, x0          ; x1 = NOT x0 = 0xFFFFFFFFFFFFFFF0
    mov x2, #0xFF       ; x2 = 255 (mask)
    and x0, x1, x2      ; x0 = x1 & 0xFF = 0xF0 (240)
    mov x16, #1         ; syscall: exit
    svc #0

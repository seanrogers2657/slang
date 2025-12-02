; @test: exit_code=0
; Basic exit with code 0
.global _start

_start:
    mov x0, #0
    mov x16, #1
    svc #0

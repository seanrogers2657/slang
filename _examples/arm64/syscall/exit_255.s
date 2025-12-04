; @test: exit_code=255
; Basic exit with code 255
.global _start

_start:
    mov x0, #255
    mov x16, #1
    svc #0

; @test: exit_code=42
; Basic exit with code 42
.global _start

_start:
    mov x0, #42
    mov x16, #1
    svc #0

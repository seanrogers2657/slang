; @test: exit_code=1
.global _start

_start:
    mov x0, #1
    mov x16, #1
    svc #0

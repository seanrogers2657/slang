; @test: exit_code=42
; Tests cbnz (compare and branch if not zero) and b (unconditional branch)
.global _start
.align 4
_start:
    mov x0, #1
    cbnz x0, nonzero    ; should branch (x0 is non-zero)
    mov x0, #99         ; should be skipped
    b exit_prog
nonzero:
    mov x0, #42
exit_prog:
    mov x16, #1
    svc #0

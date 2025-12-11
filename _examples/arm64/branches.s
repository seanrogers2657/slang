; @test: exit_code=42
; Tests cbz (compare and branch if zero) and b (unconditional branch)
.global _start
.align 4
_start:
    mov x0, #0
    cbz x0, zero        ; should branch (x0 is 0)
    mov x0, #99         ; should be skipped
    b exit_prog
zero:
    mov x0, #42
exit_prog:
    mov x16, #1
    svc #0

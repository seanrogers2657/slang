; @test: exit_code=15
; Sum of 1 to 5: 1+2+3+4+5 = 15
.global _start

_start:
    mov x0, #0       ; sum = 0
    mov x1, #1       ; i = 1
sum_loop:
    cmp x1, #6       ; while i < 6
    b.ge sum_done
    add x0, x0, x1   ; sum += i
    add x1, x1, #1   ; i++
    b sum_loop
sum_done:
    mov x16, #1      ; sum = 1+2+3+4+5 = 15
    svc #0

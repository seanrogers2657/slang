; @test: exit_code=6
; Factorial of 3: 3! = 6
.global _start

_start:
    mov x0, #3       ; n = 3
    mov x1, #1       ; result = 1
factorial_loop:
    cmp x0, #1
    b.le factorial_done
    mul x1, x1, x0   ; result *= n
    sub x0, x0, #1   ; n--
    b factorial_loop
factorial_done:
    mov x0, x1       ; return result (3! = 6)
    mov x16, #1
    svc #0

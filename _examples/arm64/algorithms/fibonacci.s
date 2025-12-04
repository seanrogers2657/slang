; @test: exit_code=8
; Fibonacci of 6: fib(6) = 8
.global _start

_start:
    mov x0, #6       ; compute fib(6)
    mov x1, #0       ; fib(0) = 0
    mov x2, #1       ; fib(1) = 1
fib_loop:
    cmp x0, #0
    b.eq fib_done
    add x3, x1, x2   ; next = a + b
    mov x1, x2       ; a = b
    mov x2, x3       ; b = next
    sub x0, x0, #1   ; n--
    b fib_loop
fib_done:
    mov x0, x1       ; return fib(6) = 8
    mov x16, #1
    svc #0

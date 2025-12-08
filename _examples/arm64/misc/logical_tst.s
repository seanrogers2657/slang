; @test: exit_code=1
; Test TST instruction: test if bit 0 is set, branch accordingly
.global _start

_start:
    mov x0, #5          ; x0 = 5 (binary 101, bit 0 is set)
    mov x1, #1          ; x1 = 1 (mask for bit 0)
    tst x0, x1          ; test x0 & x1, sets flags (Z=0 since result != 0)
    b.ne bit_set        ; branch if not equal (Z=0)
    mov x0, #0          ; bit not set path
    b done
bit_set:
    mov x0, #1          ; bit set path
done:
    mov x16, #1         ; syscall: exit
    svc #0

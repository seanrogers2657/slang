; @test: exit_code=0
; Test unconditional branch (b)
; The branch skips the instruction that sets x0 to 42
.global _start

_start:
    mov x0, #0          ; Set exit code to 0
    b done              ; Branch over next instruction
    mov x0, #42         ; This should be skipped
done:
    mov x16, #1         ; Exit syscall
    svc #0

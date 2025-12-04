; @test: exit_code=42
; Test forward branch - branch forward to set exit code
.global _start

_start:
    mov x0, #0          ; Start with 0
    b set_code          ; Branch forward
    mov x0, #99         ; Skipped
    mov x0, #100        ; Skipped
set_code:
    mov x0, #42         ; Set exit code to 42
    mov x16, #1
    svc #0

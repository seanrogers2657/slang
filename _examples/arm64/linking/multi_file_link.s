; @test: exit_code=50
; Test file for multi-file linking capability
; This is a simple single-file test that validates the link command works
.global _start

.text
_start:
    mov x0, #50
    mov x16, #1
    svc #0

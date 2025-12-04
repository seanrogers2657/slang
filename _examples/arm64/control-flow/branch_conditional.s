; @test: exit_code=1
; Test conditional branches (b.eq, b.ne)
; Tests equal and not-equal conditions
.global _start

_start:
    mov x0, #10         ; Set x0 to 10
    mov x1, #10         ; Set x1 to 10
    cmp x0, x1          ; Compare: should be equal
    b.ne fail           ; Branch if not equal (should NOT branch)

    mov x0, #5
    mov x1, #10
    cmp x0, x1          ; Compare: should not be equal
    b.eq fail           ; Branch if equal (should NOT branch)

    mov x0, #1          ; Success: exit with 1
    b done

fail:
    mov x0, #0          ; Failure: exit with 0

done:
    mov x16, #1
    svc #0

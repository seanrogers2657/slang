; @test: exit_code=42
; Test simple memory operations (str/ldr)
; Store a value to stack, load it back
.global _start

_start:
    mov x0, #42         ; Value to store
    str x0, [sp]        ; Store x0 to stack at [sp]
    mov x0, #0          ; Clear x0
    ldr x0, [sp]        ; Load value back from stack
    mov x16, #1
    svc #0

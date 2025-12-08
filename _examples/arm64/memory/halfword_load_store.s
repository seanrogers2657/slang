; @test: exit_code=171
; Test LDRH/STRH instructions
; Store 0xAB (171) as halfword, load it back and exit with that value
.global _start

.data
buffer:
    .space 16          ; 16 bytes of buffer space

.text
_start:
    ; Get address of buffer
    adrp x0, buffer@PAGE
    add x0, x0, buffer@PAGEOFF

    ; Store halfword value 0x00AB (171) at buffer
    mov x1, #171
    strh w1, [x0]       ; Store low 16 bits of x1

    ; Store another value at offset 2
    mov x2, #0x1234
    strh w2, [x0, #2]   ; Store at buffer+2

    ; Load back the first halfword
    ldrh w3, [x0]       ; Load from buffer (should be 171)

    ; Use loaded value as exit code
    mov x0, x3
    mov x16, #1
    svc #0

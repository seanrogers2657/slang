; main.s - Entry point that calls external math function
; @test: skip=requires linking with math.s

.global _start
.extern _math_add
.text
_start:
    mov x0, #30
    mov x1, #12
    bl _math_add
    mov x16, #1
    svc #0x80

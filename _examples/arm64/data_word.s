; Test .word and .quad data directives
; This file demonstrates multi-byte integer data declarations
; Note: Full .data section execution requires ADRP/ADR support (future enhancement)

.data
; 32-bit word
word_val:
    .word 0x12345678

; 64-bit quad
quad_val:
    .quad 0x123456789ABCDEF0

; Multiple values
integers:
    .word 100, 200, 300

; 64-bit address-sized values
addresses:
    .quad 0x100000000, 0x200000000

.text
.global _start

_start:
    ; Exit with constant
    mov x0, #1
    mov x16, #1
    svc #0

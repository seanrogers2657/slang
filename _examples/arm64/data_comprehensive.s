; Comprehensive test of all data directives
; This file demonstrates the full range of data declarations supported
; Note: Full .data section execution requires ADRP/ADR support (future enhancement)

.data
; String data
message:
    .asciz "Hello from slasm!"

; Byte values
byte_values:
    .byte 0x00, 0xFF, 0x7F, 0x80

; 16-bit half-words (2 bytes)
half_words:
    .hword 0x1234, 0x5678

; 32-bit words (4 bytes)
words:
    .word 0xDEADBEEF, 0xCAFEBABE

; 64-bit quad-words (8 bytes)
quads:
    .quad 0x123456789ABCDEF0

; Reserved space (zero-filled buffer)
buffer:
    .space 64

; Another buffer using .zero
zero_buffer:
    .zero 32

; ASCII string without null terminator
raw_string:
    .ascii "RawASCII"

; Multiple strings
greeting:
    .asciz "Welcome"

farewell:
    .asciz "Goodbye"

.text
.global _start

_start:
    ; This example demonstrates parsing of data sections
    ; Full data access would require ADRP/ADR instructions
    ; which are planned for future implementation

    ; For now, exit with a constant value
    mov x0, #42
    mov x16, #1
    svc #0

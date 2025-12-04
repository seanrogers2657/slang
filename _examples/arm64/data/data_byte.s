; @test: exit_code=42
; Test .byte data directive
; Demonstrates byte data declarations

.data
; Single byte values
byte_single:
    .byte 42

; Multiple bytes on one line
bytes_array:
    .byte 1, 2, 3, 4, 5

; Newline character
newline:
    .byte 10

.text
.global _start

_start:
    ; For now, just exit with code based on constant
    mov x0, #42
    mov x16, #1
    svc #0

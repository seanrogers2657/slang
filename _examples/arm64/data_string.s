; Test .asciz string data directive
; This file demonstrates null-terminated string declarations
; Note: Full .data section execution requires ADRP/ADR support (future enhancement)

.data
; Simple string
hello:
    .asciz "Hello, World!"

; String with escape sequences
escaped:
    .asciz "Line1\nLine2\tTabbed"

; Empty string (just null terminator)
empty:
    .asciz ""

; ASCII without null terminator
ascii_no_null:
    .ascii "NoNull"

.text
.global _start

_start:
    ; Exit with code 0 (success)
    mov x0, #0
    mov x16, #1
    svc #0

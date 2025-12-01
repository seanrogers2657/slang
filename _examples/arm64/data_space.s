; Test .space data directive
; This file demonstrates zero-filled buffer declarations
; Note: Full .data section execution requires ADRP/ADR support (future enhancement)

.data
; Buffer of 32 zero bytes
buffer32:
    .space 32

; Buffer of 256 bytes
large_buffer:
    .space 256

; Small buffer
tiny:
    .space 8

; Alternative: .zero directive
zeros:
    .zero 16

.text
.global _start

_start:
    ; Exit with code 0
    mov x0, #0
    mov x16, #1
    svc #0

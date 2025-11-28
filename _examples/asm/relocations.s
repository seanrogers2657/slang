// relocations.s - Demonstrate @PAGE and @PAGEOFF relocations
// These are required for accessing data on macOS ARM64

.data
.align 3
buffer:
    .space 32
value:
    .byte 42
string1:
    .asciz "First"
string2:
    .asciz "Second"

.text
.global _start
.align 4

_start:
    // Load address of buffer using @PAGE/@PAGEOFF
    adrp x0, buffer@PAGE
    add x0, x0, buffer@PAGEOFF

    // Load address of value
    adrp x1, value@PAGE
    add x1, x1, value@PAGEOFF

    // Load address of string1
    adrp x2, string1@PAGE
    add x2, x2, string1@PAGEOFF

    // Load address of string2
    adrp x3, string2@PAGE
    add x3, x3, string2@PAGEOFF

    // Exit
    mov x0, #0
    mov x16, #1
    svc #0

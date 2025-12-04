// @test: exit_code=42
// data_section.s - Demonstrate various data section directives
// Shows .byte, .space, .asciz, .align directives

.data
.align 3

// Single byte
single_byte:
    .byte 42

// Buffer space
buffer:
    .space 32

// Null-terminated string
message:
    .asciz "Hello from data section!"

// Another byte after alignment
.align 2
another_byte:
    .byte 255

// Multiple bytes
byte_array:
    .byte 1
    .byte 2
    .byte 3
    .byte 4

// Escaped string
escaped_string:
    .asciz "Line 1\nLine 2\tTabbed"

.text
.global _start
.align 4

_start:
    // Just load addresses to verify sections work
    adrp x0, single_byte@PAGE
    add x0, x0, single_byte@PAGEOFF
    ldrb w0, [x0]       // Load the byte value (42)

    adrp x1, buffer@PAGE
    add x1, x1, buffer@PAGEOFF

    adrp x2, message@PAGE
    add x2, x2, message@PAGEOFF

    // Exit with the loaded byte as exit code
    mov x16, #1
    svc #0

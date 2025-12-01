// hello.s - Print "Hello, World!" to stdout
// Demonstrates data section, string literals, and write syscall

.data
.align 3
message:
    .asciz "Hello, World!\n"
message_len = 14

.text
.global _start
.align 4

_start:
    // Write "Hello, World!\n" to stdout
    adrp x1, message@PAGE
    add x1, x1, message@PAGEOFF
    mov x2, #message_len    // Length
    mov x0, #1              // File descriptor: stdout
    mov x16, #4             // Syscall number: write
    svc #0x80               // Make syscall

    // Exit with status 0
    mov x0, #0
    mov x16, #1
    svc #0

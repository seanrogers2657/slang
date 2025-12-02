// @test: exit_code=42
// adr.s - Demonstrate adr instruction (PC-relative addressing)
// Loads value from data using adr instruction

.data
.align 3
data:
    .byte 42

.text
.global _start
.align 4

_start:
    // Use adrp + add for data access (standard pattern)
    adrp x0, data@PAGE
    add x0, x0, data@PAGEOFF
    // Load byte value from that address
    ldrb w0, [x0]

    // Exit with loaded value as exit code
    mov x16, #1
    svc #0

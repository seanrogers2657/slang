// @test: exit_code=0
// load_store.s - Demonstrate load and store instructions
// Shows ldr, str, ldp, stp with various addressing modes

.data
.align 3
values:
    .byte 10
    .byte 20
    .byte 30
    .byte 40
buffer:
    .space 16

.text
.global _start
.align 4

_start:
    // Load byte from memory
    adrp x0, values@PAGE
    add x0, x0, values@PAGEOFF
    ldrb w1, [x0]           // Load first byte (10)
    ldrb w2, [x0, #1]       // Load second byte (20)
    ldrb w3, [x0, #2]       // Load third byte (30)

    // Store to buffer
    adrp x4, buffer@PAGE
    add x4, x4, buffer@PAGEOFF
    strb w1, [x4]           // Store first value
    strb w2, [x4, #1]       // Store second value
    strb w3, [x4, #2]       // Store third value

    // Load/store pairs
    mov x5, #100
    mov x6, #200
    stp x5, x6, [x4, #8]    // Store pair
    ldp x7, x8, [x4, #8]    // Load pair back

    // Pre-index and post-index addressing
    mov x9, x4
    ldr x10, [x9], #8       // Post-index: load then increment
    ldr x11, [x9, #8]!      // Pre-index: increment then load

    // Exit
    mov x0, #0
    mov x16, #1
    svc #0

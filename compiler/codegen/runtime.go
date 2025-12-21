package codegen

// RuntimePanicCode returns the ARM64 assembly code for the runtime panic handler.
// This code provides:
// - _slang_panic: The main panic function that prints error message and stack trace
// - _slang_symtab_lookup: Lookup function address in symbol table
// - _slang_itoa: Integer to string conversion for line numbers
func RuntimePanicCode() string {
	return `
// ============================================================================
// Slang Runtime - Panic Handler
// ============================================================================

.data
.align 3

// Panic message strings
_panic_prefix:     .asciz "panic: "
_panic_prefix_len = 7
_panic_at:         .asciz "    at "
_panic_at_len = 7
_panic_paren:      .asciz "() "
_panic_paren_len = 3
_panic_colon:      .asciz ":"
_panic_colon_len = 1
_panic_newline:    .asciz "\n"
_panic_newline_len = 1
_panic_unknown:    .asciz "<unknown>"
_panic_unknown_len = 9

// Error message strings
.align 3
_err_msg_1:     .asciz "integer overflow: addition"
_err_msg_2:     .asciz "integer overflow: subtraction"
_err_msg_3:     .asciz "integer overflow: multiplication"
_err_msg_4:     .asciz "unsigned overflow: addition"
_err_msg_5:     .asciz "unsigned underflow: subtraction"
_err_msg_6:     .asciz "unsigned overflow: multiplication"
_err_msg_7:     .asciz "division by zero"
_err_msg_8:     .asciz "modulo by zero"
_err_msg_9:     .asciz "array index out of bounds"

// Buffer for line number conversion
.align 3
_panic_line_buffer: .space 32

.text
.align 4

// ============================================================================
// _slang_panic - Main panic handler
// Arguments: x0 = error code (1-8), x1 = line number where panic occurred
// ============================================================================
_slang_panic:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!
    stp x23, x24, [sp, #-16]!

    mov x19, x0                     // x19 = error code
    mov x23, x1                     // x23 = actual line number

    // Write "panic: " to stderr
    mov x0, #2
    adrp x1, _panic_prefix@PAGE
    add x1, x1, _panic_prefix@PAGEOFF
    mov x2, #7
    mov x16, #4
    svc #0x80

    // Look up error message by code using computed addresses (ASLR-safe)
    cmp x19, #1
    beq _panic_msg_1
    cmp x19, #2
    beq _panic_msg_2
    cmp x19, #3
    beq _panic_msg_3
    cmp x19, #4
    beq _panic_msg_4
    cmp x19, #5
    beq _panic_msg_5
    cmp x19, #6
    beq _panic_msg_6
    cmp x19, #7
    beq _panic_msg_7
    cmp x19, #8
    beq _panic_msg_8
    cmp x19, #9
    beq _panic_msg_9
    b _panic_msg_done

_panic_msg_1:
    adrp x1, _err_msg_1@PAGE
    add x1, x1, _err_msg_1@PAGEOFF
    mov x2, #26
    b _panic_msg_print

_panic_msg_2:
    adrp x1, _err_msg_2@PAGE
    add x1, x1, _err_msg_2@PAGEOFF
    mov x2, #29
    b _panic_msg_print

_panic_msg_3:
    adrp x1, _err_msg_3@PAGE
    add x1, x1, _err_msg_3@PAGEOFF
    mov x2, #32
    b _panic_msg_print

_panic_msg_4:
    adrp x1, _err_msg_4@PAGE
    add x1, x1, _err_msg_4@PAGEOFF
    mov x2, #27
    b _panic_msg_print

_panic_msg_5:
    adrp x1, _err_msg_5@PAGE
    add x1, x1, _err_msg_5@PAGEOFF
    mov x2, #31
    b _panic_msg_print

_panic_msg_6:
    adrp x1, _err_msg_6@PAGE
    add x1, x1, _err_msg_6@PAGEOFF
    mov x2, #33
    b _panic_msg_print

_panic_msg_7:
    adrp x1, _err_msg_7@PAGE
    add x1, x1, _err_msg_7@PAGEOFF
    mov x2, #16
    b _panic_msg_print

_panic_msg_8:
    adrp x1, _err_msg_8@PAGE
    add x1, x1, _err_msg_8@PAGEOFF
    mov x2, #14
    b _panic_msg_print

_panic_msg_9:
    adrp x1, _err_msg_9@PAGE
    add x1, x1, _err_msg_9@PAGEOFF
    mov x2, #25
    b _panic_msg_print

_panic_msg_print:
    // Write error message to stderr
    mov x0, #2
    mov x16, #4
    svc #0x80

_panic_msg_done:

    // Write newline
    mov x0, #2
    adrp x1, _panic_newline@PAGE
    add x1, x1, _panic_newline@PAGEOFF
    mov x2, #1
    mov x16, #4
    svc #0x80

    // Walk the stack - start with _slang_panic's caller
    // x29 points to our saved frame: [prev_fp, return_addr]
    mov x20, x29                    // start with our frame
    mov x24, #1                     // x24 = 1 means first frame (use passed line number)

_walk_loop:
    cbz x20, _walk_done
    ldr x21, [x20, #8]              // return address
    ldr x22, [x20]                  // previous frame pointer

    mov x0, x21
    bl _slang_symtab_lookup

    cbz x0, _walk_next

    mov x9, x0                      // x9 = name pointer
    mov x10, x1                     // x10 = name length
    mov x11, x2                     // x11 = file pointer
    mov x12, x3                     // x12 = file length

    // For the first frame, use the passed line number (x23)
    // For subsequent frames, use the line from symbol table (x4)
    cmp x24, #1
    bne _use_symtab_line
    mov x13, x23                    // Use passed line number
    mov x24, #0                     // Clear first frame flag
    b _got_line
_use_symtab_line:
    mov x13, x4                     // Use symbol table line
_got_line:

    // Print "    at "
    mov x0, #2
    adrp x1, _panic_at@PAGE
    add x1, x1, _panic_at@PAGEOFF
    mov x2, #7
    mov x16, #4
    svc #0x80

    // Print function name
    mov x0, #2
    mov x1, x9
    mov x2, x10
    mov x16, #4
    svc #0x80

    // Print "() "
    mov x0, #2
    adrp x1, _panic_paren@PAGE
    add x1, x1, _panic_paren@PAGEOFF
    mov x2, #3
    mov x16, #4
    svc #0x80

    // Print filename
    mov x0, #2
    mov x1, x11
    mov x2, x12
    mov x16, #4
    svc #0x80

    // Print ":"
    mov x0, #2
    adrp x1, _panic_colon@PAGE
    add x1, x1, _panic_colon@PAGEOFF
    mov x2, #1
    mov x16, #4
    svc #0x80

    // Convert and print line number
    mov x0, x13
    bl _slang_itoa
    mov x2, x1
    mov x1, x0
    mov x0, #2
    mov x16, #4
    svc #0x80

    // Print newline
    mov x0, #2
    adrp x1, _panic_newline@PAGE
    add x1, x1, _panic_newline@PAGEOFF
    mov x2, #1
    mov x16, #4
    svc #0x80

_walk_next:
    mov x20, x22
    b _walk_loop

_walk_done:
    ldp x23, x24, [sp], #16
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    mov x0, #1
    mov x16, #1
    svc #0

// ============================================================================
// _slang_symtab_lookup - Look up return address in symbol table
// Arguments: x0 = return address
// Returns: x0=name, x1=namelen, x2=file, x3=filelen, x4=line (or all 0)
//
// Note: This function computes the ASLR slide to correctly compare addresses.
// The symbol table stores absolute addresses based on preferred load address,
// but at runtime the binary may be loaded at a different address.
// Also looks up the line number table for exact call site line numbers.
// ============================================================================
.align 4
_slang_symtab_lookup:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!
    stp x23, x24, [sp, #-16]!
    stp x25, x26, [sp, #-16]!

    mov x19, x0                         // x19 = return address to look up

    // Compute the ASLR slide
    // x20 = actual runtime address of _slang_symtab
    adrp x20, _slang_symtab@PAGE
    add x20, x20, _slang_symtab@PAGEOFF

    // x21 = expected address stored in _slang_symtab_ref
    adrp x8, _slang_symtab_ref@PAGE
    add x8, x8, _slang_symtab_ref@PAGEOFF
    ldr x21, [x8]

    // x22 = slide = actual - expected
    sub x22, x20, x21

    // Now iterate through symbol table
    mov x8, x20                         // x8 = pointer into symbol table

_lookup_loop:
    ldr x9, [x8]                        // x9 = stored start address
    cbz x9, _lookup_notfound

    // Apply slide to stored addresses
    add x9, x9, x22                     // x9 = actual start address
    ldr x10, [x8, #8]
    add x10, x10, x22                   // x10 = actual end address

    // Compare return address against address range
    cmp x19, x9
    b.lt _lookup_next
    cmp x19, x10
    b.ge _lookup_next

    // Found a match - load and adjust pointer fields
    ldr x0, [x8, #16]
    add x0, x0, x22                     // name pointer (adjusted)
    ldr x1, [x8, #24]                   // name length (no adjustment)
    ldr x2, [x8, #32]
    add x2, x2, x22                     // file pointer (adjusted)
    ldr x3, [x8, #40]                   // file length (no adjustment)
    ldr x4, [x8, #48]                   // line number (default: function start)

    // Save function info before line table lookup
    mov x23, x0                         // save name ptr
    mov x24, x1                         // save name len
    mov x25, x2                         // save file ptr
    mov x26, x3                         // save file len
    // x4 = default line number

    // Now look up exact line number in line table
    adrp x8, _slang_linetab@PAGE
    add x8, x8, _slang_linetab@PAGEOFF

_linetab_loop:
    ldr x9, [x8]                        // address from line table (stored address)
    cbz x9, _linetab_done               // sentinel = end of table, use default line

    // Apply slide and check for exact match
    add x9, x9, x22                     // x9 = actual address
    cmp x19, x9
    b.ne _linetab_next

    // Found exact match! Load line number
    ldr x4, [x8, #8]                    // x4 = line number from line table
    b _linetab_done

_linetab_next:
    add x8, x8, #16                     // sizeof(LineEntry) = 2 * 8
    b _linetab_loop

_linetab_done:
    // Restore function info (x4 already has the correct line number)
    mov x0, x23                         // name ptr
    mov x1, x24                         // name len
    mov x2, x25                         // file ptr
    mov x3, x26                         // file len
    // x4 has line number (from line table if found, else function start)

    ldp x25, x26, [sp], #16
    ldp x23, x24, [sp], #16
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret

_lookup_next:
    add x8, x8, #56
    b _lookup_loop

_lookup_notfound:
    mov x0, #0
    mov x1, #0
    mov x2, #0
    mov x3, #0
    mov x4, #0

    ldp x25, x26, [sp], #16
    ldp x23, x24, [sp], #16
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret

// ============================================================================
// _slang_itoa - Convert integer to ASCII string
// Arguments: x0 = integer value
// Returns: x0 = pointer to string, x1 = string length
// ============================================================================
.align 4
_slang_itoa:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

    adrp x19, _panic_line_buffer@PAGE
    add x19, x19, _panic_line_buffer@PAGEOFF
    mov x20, x0
    mov x21, #0

    cmp x20, #0
    bne _itoa_check_negative
    mov w10, #48
    strb w10, [x19]
    mov x0, x19
    mov x1, #1
    b _itoa_done

_itoa_check_negative:
    cmp x20, #0
    bge _itoa_convert_setup
    mov x21, #1
    neg x20, x20

_itoa_convert_setup:
    mov x22, #0
    add x19, x19, #31

_itoa_convert_loop:
    mov x10, #10
    udiv x11, x20, x10
    msub x12, x11, x10, x20
    add x12, x12, #48
    strb w12, [x19]
    sub x19, x19, #1
    add x22, x22, #1
    mov x20, x11
    cmp x20, #0
    bne _itoa_convert_loop

    cmp x21, #1
    bne _itoa_finalize
    mov w10, #45
    strb w10, [x19]
    sub x19, x19, #1
    add x22, x22, #1

_itoa_finalize:
    add x19, x19, #1
    mov x0, x19
    mov x1, x22

_itoa_done:
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret

`
}

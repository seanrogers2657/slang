// Slang runtime panic handler for ARM64 macOS
//
// This file provides the _slang_panic function and supporting routines
// for runtime error handling with stack traces.

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

// Error messages table - indexed by (error_code - 1)
.align 3
_slang_error_messages:
    .quad _err_msg_1
    .quad _err_msg_2
    .quad _err_msg_3
    .quad _err_msg_4
    .quad _err_msg_5
    .quad _err_msg_6
    .quad _err_msg_7
    .quad _err_msg_8

_err_msg_1:     .asciz "integer overflow: addition"
_err_msg_1_len = 26
_err_msg_2:     .asciz "integer overflow: subtraction"
_err_msg_2_len = 29
_err_msg_3:     .asciz "integer overflow: multiplication"
_err_msg_3_len = 32
_err_msg_4:     .asciz "unsigned overflow: addition"
_err_msg_4_len = 27
_err_msg_5:     .asciz "unsigned underflow: subtraction"
_err_msg_5_len = 31
_err_msg_6:     .asciz "unsigned overflow: multiplication"
_err_msg_6_len = 33
_err_msg_7:     .asciz "division by zero"
_err_msg_7_len = 16
_err_msg_8:     .asciz "modulo by zero"
_err_msg_8_len = 14

// Error message lengths table
.align 3
_slang_error_lengths:
    .quad 26   // err_msg_1_len
    .quad 29   // err_msg_2_len
    .quad 32   // err_msg_3_len
    .quad 27   // err_msg_4_len
    .quad 31   // err_msg_5_len
    .quad 33   // err_msg_6_len
    .quad 16   // err_msg_7_len
    .quad 14   // err_msg_8_len

// Buffer for line number conversion
.align 3
_panic_line_buffer: .space 32

.text
.align 4

// ============================================================================
// _slang_panic - Main panic handler
//
// Arguments:
//   x0 = error code (1-8)
//
// This function prints the panic message with stack trace and exits.
// ============================================================================
.global _slang_panic
_slang_panic:
    // Save frame pointer for stack walking
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    // Save callee-saved registers we'll use
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

    // Save error code
    mov x19, x0

    // Write "panic: " to stderr (fd=2)
    mov x0, #2                          // stderr
    adrp x1, _panic_prefix@PAGE
    add x1, x1, _panic_prefix@PAGEOFF
    mov x2, #7                          // _panic_prefix_len
    mov x16, #4                         // write syscall
    svc #0x80

    // Look up error message by code
    // x19 = error code (1-based)
    sub x20, x19, #1                    // Convert to 0-based index

    // Load message pointer from table
    adrp x8, _slang_error_messages@PAGE
    add x8, x8, _slang_error_messages@PAGEOFF
    ldr x1, [x8, x20, lsl #3]           // x1 = message pointer

    // Load message length from table
    adrp x8, _slang_error_lengths@PAGE
    add x8, x8, _slang_error_lengths@PAGEOFF
    ldr x2, [x8, x20, lsl #3]           // x2 = message length

    // Write error message to stderr
    mov x0, #2
    mov x16, #4
    svc #0x80

    // Write newline
    mov x0, #2
    adrp x1, _panic_newline@PAGE
    add x1, x1, _panic_newline@PAGEOFF
    mov x2, #1
    mov x16, #4
    svc #0x80

    // Walk the stack and print trace
    // Start with our caller's frame (skip panic frame)
    ldr x20, [x29]                      // x20 = caller's frame pointer

.Lwalk_loop:
    cbz x20, .Lwalk_done                // null frame pointer = done

    // Load return address
    ldr x21, [x20, #8]                  // x21 = return address

    // Load previous frame pointer (for next iteration)
    ldr x22, [x20]                      // x22 = previous frame pointer

    // Look up return address in symbol table
    mov x0, x21
    bl _slang_symtab_lookup
    // Returns: x0=name, x1=namelen, x2=file, x3=filelen, x4=line (or x0=0 if not found)

    cbz x0, .Lwalk_next                 // skip if not found

    // Save lookup results
    mov x9, x0                          // name ptr
    mov x10, x1                         // name len
    mov x11, x2                         // file ptr
    mov x12, x3                         // file len
    mov x13, x4                         // line number

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

    // Convert line number to string and print
    mov x0, x13
    bl _slang_itoa
    // Returns: x0 = buffer ptr, x1 = length

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

.Lwalk_next:
    mov x20, x22                        // Move to previous frame
    b .Lwalk_loop

.Lwalk_done:
    // Exit with code 1
    mov x0, #1
    mov x16, #1
    svc #0

// ============================================================================
// _slang_symtab_lookup - Look up return address in symbol table
//
// Arguments:
//   x0 = return address to look up
//
// Returns:
//   x0 = function name pointer (or 0 if not found)
//   x1 = function name length
//   x2 = filename pointer
//   x3 = filename length
//   x4 = line number (from line table if exact match, else function start line)
// ============================================================================
.align 4
_slang_symtab_lookup:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x23, x24, [sp, #-16]!

    mov x19, x0                         // Save return address

    // Load symbol table base address
    adrp x8, _slang_symtab@PAGE
    add x8, x8, _slang_symtab@PAGEOFF

.Llookup_loop:
    ldr x9, [x8]                        // start address
    cbz x9, .Llookup_notfound           // sentinel = end of table

    ldr x10, [x8, #8]                   // end address

    // Check if return_addr >= start
    cmp x19, x9
    b.lt .Llookup_next

    // Check if return_addr < end
    cmp x19, x10
    b.ge .Llookup_next

    // Found function! Load the entry data
    ldr x0, [x8, #16]                   // name pointer
    ldr x1, [x8, #24]                   // name length
    ldr x2, [x8, #32]                   // file pointer
    ldr x3, [x8, #40]                   // file length
    ldr x4, [x8, #48]                   // line number (default: function start)

    // Save function info before line table lookup
    mov x20, x0                         // save name ptr
    mov x23, x1                         // save name len
    mov x24, x2                         // save file ptr
    // x3 = file len (we'll restore it later)
    // x4 = default line number

    // Now look up exact line number in line table
    // x19 = return address we're looking for
    adrp x8, _slang_linetab@PAGE
    add x8, x8, _slang_linetab@PAGEOFF

.Llinetab_loop:
    ldr x9, [x8]                        // address from line table
    cbz x9, .Llinetab_done              // sentinel = end of table, use default line

    // Check for exact match
    cmp x19, x9
    b.ne .Llinetab_next

    // Found exact match! Load line number
    ldr x4, [x8, #8]                    // x4 = line number from line table
    b .Llinetab_done

.Llinetab_next:
    add x8, x8, #16                     // sizeof(LineEntry) = 2 * 8
    b .Llinetab_loop

.Llinetab_done:
    // Restore function info (x4 already has the line number)
    mov x0, x20                         // name ptr
    mov x1, x23                         // name len
    mov x2, x24                         // file ptr
    // x3 still has file len from symbol table lookup
    // x4 has line number (from line table if found, else function start)

    ldp x23, x24, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret

.Llookup_next:
    add x8, x8, #56                     // sizeof(SymbolEntry) = 7 * 8
    b .Llookup_loop

.Llookup_notfound:
    mov x0, #0
    mov x1, #0
    mov x2, #0
    mov x3, #0
    mov x4, #0

    ldp x23, x24, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret

// ============================================================================
// _slang_itoa - Convert integer to ASCII string
//
// Arguments:
//   x0 = integer value
//
// Returns:
//   x0 = pointer to string (in _panic_line_buffer)
//   x1 = string length
// ============================================================================
.align 4
_slang_itoa:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

    adrp x19, _panic_line_buffer@PAGE
    add x19, x19, _panic_line_buffer@PAGEOFF
    mov x20, x0                         // value
    mov x21, #0                         // is_negative flag

    // Handle zero case
    cmp x20, #0
    bne .Litoa_check_negative
    mov w10, #48                        // '0'
    strb w10, [x19]
    mov x0, x19
    mov x1, #1
    b .Litoa_done

.Litoa_check_negative:
    cmp x20, #0
    bge .Litoa_convert_setup
    mov x21, #1
    neg x20, x20

.Litoa_convert_setup:
    mov x22, #0                         // digit count
    add x19, x19, #31                   // point to end of buffer

.Litoa_convert_loop:
    mov x10, #10
    udiv x11, x20, x10
    msub x12, x11, x10, x20             // remainder = value % 10
    add x12, x12, #48                   // convert to ASCII
    strb w12, [x19]
    sub x19, x19, #1
    add x22, x22, #1
    mov x20, x11
    cmp x20, #0
    bne .Litoa_convert_loop

    // Add minus sign if negative
    cmp x21, #1
    bne .Litoa_finalize
    mov w10, #45                        // '-'
    strb w10, [x19]
    sub x19, x19, #1
    add x22, x22, #1

.Litoa_finalize:
    add x19, x19, #1                    // point to first char
    mov x0, x19
    mov x1, x22

.Litoa_done:
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret

// ============================================================================
// Default empty symbol table (will be replaced by generated code)
// ============================================================================
.data
.align 3
.weak _slang_symtab
_slang_symtab:
    .quad 0                             // sentinel (empty table)

// ============================================================================
// Default empty line table (will be replaced by generated code)
// ============================================================================
.align 3
.weak _slang_linetab
_slang_linetab:
    .quad 0                             // sentinel (empty table)

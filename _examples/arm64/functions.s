// @test: exit_code=60
// @test: skip=uses lsl instruction not supported by slasm
// functions.s - Demonstrate function calls with bl and ret
// (10+20)*2 = 60

.global _start
.align 4

_start:
    // Call add_numbers function
    mov x0, #10
    mov x1, #20
    bl add_numbers      // Result in x0

    // Call multiply_by_two function
    bl multiply_by_two  // x0 = x0 * 2

    // Exit with result as exit code
    mov x16, #1
    svc #0

// Function: add_numbers
// Adds two numbers
// Input: x0 = first number, x1 = second number
// Output: x0 = sum
.align 4
add_numbers:
    // Save frame pointer and link register
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    // Add the numbers
    add x0, x0, x1

    // Restore frame pointer and link register
    ldp x29, x30, [sp], #16
    ret

// Function: multiply_by_two
// Multiplies a number by 2
// Input: x0 = number
// Output: x0 = number * 2
.align 4
multiply_by_two:
    // Save frame pointer and link register
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    // Multiply by 2 (shift left by 1)
    lsl x0, x0, #1

    // Restore frame pointer and link register
    ldp x29, x30, [sp], #16
    ret

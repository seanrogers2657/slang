// @test: exit_code=50
// constants.s - Demonstrate constant definitions
// Uses = assignment syntax for assembly-time constants

.global _start
.align 4

// Define constants
EXIT_CODE = 50
ADD_VALUE = 10
MULTIPLY = 5

_start:
    // Use constants in instructions
    mov x0, #ADD_VALUE      // x0 = 10
    mov x1, #MULTIPLY       // x1 = 5
    mul x0, x0, x1          // x0 = 10 * 5 = 50

    // Verify it matches EXIT_CODE
    cmp x0, #EXIT_CODE
    b.ne fail

    // Exit with EXIT_CODE
    mov x0, #EXIT_CODE
    mov x16, #1
    svc #0

fail:
    // Exit with error code if verification failed
    mov x0, #1
    mov x16, #1
    svc #0

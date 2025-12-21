; @test: exit_code=42
; Test array-like access pattern: base + (-index * 16)
; Simulates: arr[2] where arr is at x4 and each element is 16 bytes
; base=100, index=2, expected offset = 100 + (-2 * 16) = 100 - 32 = 68
; We store 42 at that calculated position and verify
.global _start
.align 4

_start:
    ; Simulate base address (100) and index (2)
    mov x4, #100          ; base address
    mov x2, #2            ; index

    ; Calculate element address: base + (-index << 4)
    neg x3, x2            ; x3 = -2
    add x4, x4, x3, lsl #4 ; x4 = 100 + (-2 * 16) = 100 - 32 = 68

    ; Verify x4 == 68 by computing 68 + (-26) = 42
    sub x0, x4, #26       ; x0 = 68 - 26 = 42

    mov x16, #1
    svc #0

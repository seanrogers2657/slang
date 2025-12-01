; Test all comparison condition codes
; Expected exit code: 0 (all tests pass)
; Tests: lt, gt, le, ge for signed comparisons
.global _start

_start:
    mov x0, #0          ; Test counter (success = 0)

    ; Test less than (lt): 5 < 10 should be true
    mov x1, #5
    mov x2, #10
    cmp x1, x2
    b.lt test2          ; Should branch
    add x0, x0, #1      ; Fail: increment counter

test2:
    ; Test greater than (gt): 10 > 5 should be true
    mov x1, #10
    mov x2, #5
    cmp x1, x2
    b.gt test3          ; Should branch
    add x0, x0, #1      ; Fail: increment counter

test3:
    ; Test less than or equal (le): 5 <= 5 should be true
    mov x1, #5
    mov x2, #5
    cmp x1, x2
    b.le test4          ; Should branch
    add x0, x0, #1      ; Fail: increment counter

test4:
    ; Test greater than or equal (ge): 10 >= 5 should be true
    mov x1, #10
    mov x2, #5
    cmp x1, x2
    b.ge done           ; Should branch
    add x0, x0, #1      ; Fail: increment counter

done:
    ; x0 should be 0 if all tests passed
    mov x16, #1
    svc #0

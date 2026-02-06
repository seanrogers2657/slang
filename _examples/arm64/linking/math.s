; math.s - Math functions library for linking test
; @test: skip=library file, not standalone

.global _math_add
.text
_math_add:
    add x0, x0, x1
    ret

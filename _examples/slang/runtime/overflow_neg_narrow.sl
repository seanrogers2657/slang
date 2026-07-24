// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: negation
// Regression: negating narrow signed MIN produces an out-of-range value
// (-(s8 -128) = 128), which must trap like the arithmetic narrow checks.
main = () {
    val a: s8 = -128
    print(-a)
}

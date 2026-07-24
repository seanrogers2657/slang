// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: division
// Regression: narrow signed MIN / -1 produces a value out of the narrow range
// (s8 -128 / -1 = 128), which must trap like the add/sub/mul narrow checks.
main = () {
    val a: s8 = -128
    val b: s8 = -1
    print(a / b)
}

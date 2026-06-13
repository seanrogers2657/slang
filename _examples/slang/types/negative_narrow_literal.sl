// @test: exit_code=0
// @test: stdout=-5\n-128\n-100\n
// Regression: an in-range negative literal must be assignable to a narrow
// signed type. `-5` parsed as a unary-minus on an s64 literal and bypassed the
// literal-bounds machinery, so `val a: s8 = -5` was wrongly rejected.
main = () {
    val a: s8 = -5
    print(a)

    val min: s8 = -128
    print(min)

    val c: s16 = -100
    print(c)
}

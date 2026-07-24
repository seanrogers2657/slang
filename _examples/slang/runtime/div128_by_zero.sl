// @test: exit_code=1
// @test: stderr_contains=panic: division by zero
// 128-bit division by zero traps like the 64-bit path (the divmod helper is
// only reached after the zero check).
main = () {
    val a: s128 = 170141183460469231731687303715884105727
    val b: s128 = 0
    print(a / b)
}

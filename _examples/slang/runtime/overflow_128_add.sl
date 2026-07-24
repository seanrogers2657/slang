// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
// 128-bit signed addition traps on overflow (parity with the 64-bit path).
main = () {
    val max: s128 = 170141183460469231731687303715884105727
    val one: s128 = 1
    print(max + one)
}

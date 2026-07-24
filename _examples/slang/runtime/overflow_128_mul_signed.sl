// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: multiplication
// 128-bit signed multiplication traps: INT_MIN * -1 would be +2^127, which is
// not representable (only -2^127 is).
main = () {
    val min: s128 = -170141183460469231731687303715884105728
    val neg1: s128 = -1
    print(min * neg1)
}

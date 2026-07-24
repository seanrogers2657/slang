// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: negation
// Negating s128 INT_MIN overflows (its magnitude 2^127 is not representable).
main = () {
    val min: s128 = -170141183460469231731687303715884105728
    print(-min)
}

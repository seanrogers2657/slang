// @test: exit_code=0
// @test: stdout=5\n3\n0\n
// Nullable function parameters with elvis operator

try_elvis = (a: s64?, b: s64?) -> s64 {
    return a ?: b ?: 0
}

main = () {
    print(try_elvis(null, 5))
    print(try_elvis(3, 5))
    print(try_elvis(null, null))
}

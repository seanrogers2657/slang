// @test: exit_code=0
// @test: stdout=10\n10\n0\n
// When expression with bare variable results

pick = (a: s64, b: s64) -> s64 {
    return when {
        a > b -> a
        a < b -> b
        else -> 0
    }
}

main = () {
    print(pick(10, 5))
    print(pick(5, 10))
    print(pick(7, 7))
}

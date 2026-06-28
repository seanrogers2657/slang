// @test: exit_code=0
// @test: stdout=7\n12\n
// Strategy pattern. Slang has no first-class functions, so a strategy is an
// integer tag dispatched with `when` rather than a function value passed around.
apply = (op: s64, a: s64, b: s64) -> s64 {
    return when {
        op == 0 -> a + b
        op == 1 -> a * b
        else -> 0
    }
}

main = () {
    print(apply(0, 3, 4))
    print(apply(1, 3, 4))
}

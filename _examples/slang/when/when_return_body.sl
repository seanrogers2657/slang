// @test: exit_code=0
// @test: stdout=-1\n0\n1\n
// Regression: a bare `return` as a when-case body must parse (previously hung
// the parser) and return from the enclosing function.
classify = (x: s64) -> s64 {
    when {
        x < 0 -> return -1
        x == 0 -> return 0
        else -> return 1
    }
    return 99
}

main = () {
    print(classify(-5))
    print(classify(0))
    print(classify(7))
}

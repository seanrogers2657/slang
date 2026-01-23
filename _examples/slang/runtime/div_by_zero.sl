// @test: exit_code=1
// @test: stderr_contains=panic: division by zero
main = () {
    val a: s64 = 42
    val b: s64 = 0
    call4(a, b)
}

call4 = (x: int, y: int) -> int {
    return call3(x, y)
}

call3 = (x: int, y: int) -> int {
    // comment
    // another one
    // comment
    // another one
    return call2(x, y)
}

call2 = (x: int, y: int) -> int {
    // comment
    // another one
    return call1(x, y)
}

call1 = (x: int, y: int) -> int {
    return div(x, y)
}

div = (x: int, y: int) -> int {
    return x / y
}

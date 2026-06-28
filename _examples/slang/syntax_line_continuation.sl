// @test: exit_code=0
// @test: stdout=6\n9\n60\ntrue\n
// Line continuation: a statement continues across newlines when a line ends
// with an operator, when the next line begins with a binary operator, and
// inside ( ) / [ ].
add3 = (a: s64, b: s64, c: s64) -> s64 {
    return a
        + b
        + c
}

main = () {
    print(add3(1, 2, 3))          // 6 (leading-operator continuation)

    val trailing = 4 +
        5
    print(trailing)               // 9 (trailing-operator continuation)

    val inside = add3(
        10,
        20,
        30
    )
    print(inside)                 // 60 (newlines inside parentheses)

    val ok = inside > 0
        && trailing > 0
    print(ok)                     // 1 -> true
}

// @test: exit_code=0
// @test: stdout=3\n2\n3\n2\n5\n
// Tests returning a struct by value from a function: directly as a literal,
// via a local variable, and consuming the returned struct in another call.
Pair = struct {
    val quotient: s64
    val remainder: s64
}

divmod = (a: s64, b: s64) -> Pair {
    return Pair{ a / b, a % b }
}

divmod_via_var = (a: s64, b: s64) -> Pair {
    val result = Pair{ a / b, a % b }
    return result
}

sum_pair = (p: Pair) -> s64 {
    return p.quotient + p.remainder
}

main = () {
    val r = divmod(17, 5)
    print(r.quotient)
    print(r.remainder)

    val v = divmod_via_var(17, 5)
    print(v.quotient)
    print(v.remainder)

    print(sum_pair(divmod(17, 5)))
}

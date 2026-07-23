// @test: exit_code=0
// @test: stdout=v1\nx1\nx1\nd\n
// Regression: elvis on a boxed string? must copy the unwrapped inner — the
// inner IS the box's buffer, which the box's owner (binding scope exit or the
// call-temp free) also releases. Aliasing it double-freed. Both edges of the
// elvis yield an owned value (the default is copied if borrowed), so binding
// or printing the result is balanced.
get_s = (c: bool) -> string? {
    if c {
        return "v${1}"
    }
    return null
}

main = () {
    val s = get_s(true) ?: "d"    // call-temp LHS: inner copied before temp free
    print(s)                       // v1

    val m: string? = "x${1}"
    val t = m ?: "d"               // binding LHS: result owns an independent copy
    print(t)                       // x1
    print(m ?: "d")                // x1 — unbound elvis temp freed after print

    val u = get_s(false) ?: "d"    // null edge takes the default
    print(u)                       // d
}

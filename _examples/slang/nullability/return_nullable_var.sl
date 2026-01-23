// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test that nullable parameter passing preserves null state
passThrough = (x: s64?) -> s64? {
    return x
}

main = () {
    val a: s64? = 42
    val b: s64? = passThrough(a)
    print(b != null)  // true - passed through non-null

    val c: s64? = null
    val d: s64? = passThrough(c)
    print(d != null)  // false - passed through null
}

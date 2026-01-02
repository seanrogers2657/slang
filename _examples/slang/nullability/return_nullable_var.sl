// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test that nullable parameter passing preserves null state
passThrough = (x: i64?) -> i64? {
    return x
}

main = () {
    val a: i64? = 42
    val b: i64? = passThrough(a)
    print(b != null)  // true - passed through non-null

    val c: i64? = null
    val d: i64? = passThrough(c)
    print(d != null)  // false - passed through null
}

// @test: exit_code=0
// @test: stdout=10\ntrue\n20\nfalse\n
// Test mixed nullable and non-nullable parameters
process = (a: i64, b: i64?, c: i64) -> i64 {
    return a + c
}

main = () {
    val x: i64? = 42
    val result1 = process(3, x, 7)
    print(result1)  // 10
    print(x != null)  // true

    val y: i64? = null
    val result2 = process(5, y, 15)
    print(result2)  // 20
    print(y != null)  // false (y is still null, wasn't modified)
}

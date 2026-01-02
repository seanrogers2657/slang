// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test null comparison with nullable types
main = () {
    val x: i64? = null
    val y: i64? = 42
    print(x == null)
    print(y == null)
}

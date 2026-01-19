// @test: exit_code=0
// @test: stdout=10\n
// Elvis operator with non-null left operand
main = () {
    val x: i64? = 10
    val result = x ?: 42
    print(result)
}

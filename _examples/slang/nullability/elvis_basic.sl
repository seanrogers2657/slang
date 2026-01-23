// @test: exit_code=0
// @test: stdout=42\n
// Basic elvis operator with null left operand
main = () {
    val x: s64? = null
    val result = x ?: 42
    print(result)
}

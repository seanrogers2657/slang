// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test nullable variable initialized with non-null value
main = () {
    val x: i64? = 42
    print(x != null)
    print(x == null)
}

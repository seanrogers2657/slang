// @test: exit_code=1
// @test: stderr_contains=at main()
main = () {
    val a: i64 = 42
    val b: i64 = 0
    val c = a / b
    print(c)
}

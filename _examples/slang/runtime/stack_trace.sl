// @test: exit_code=1
// @test: stderr_contains=at main()
main = () {
    val a: s64 = 42
    val b: s64 = 0
    val c = a / b
    print(c)
}

// @test: exit_code=0
// @test: stdout=88\n
main = () {
    val a: s64 = 42
    val b: s64 = 2
    val c = a + b  // c = 44
    val d = c * b  // d = 88
    print(d)
}

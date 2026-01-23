// @test: exit_code=1
// @test: stderr_contains=panic: modulo by zero
main = () {
    val a: s64 = 42
    val b: s64 = 0
    val c = a % b
    print(c)
}

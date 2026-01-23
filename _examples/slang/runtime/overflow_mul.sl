// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: multiplication
main = () {
    val a: s64 = 9223372036854775807
    val b: s64 = 2
    val c = a * b
    print(c)
}

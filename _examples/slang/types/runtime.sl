// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
main = () {
    var a: s64 = 9223372036854775807
    var b: s64 = 1
    a = a + b
    print(a)
}

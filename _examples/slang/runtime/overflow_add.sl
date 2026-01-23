// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
main = () {
    val max: s64 = 9223372036854775807
    val result = max + 1
    print(result)
}

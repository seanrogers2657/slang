// @test: exit_code=1
// @test: stderr_contains=panic: unsigned overflow: addition
main = () {
    val max: u64 = 18446744073709551615
    val one: u64 = 1
    val result = max + one
    print(result)
}

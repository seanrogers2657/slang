// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
fn main(): void {
    val max: i64 = 9223372036854775807
    val result = max + 1
    print(result)
}

// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
fn main(): void {
    var a: i64 = 9223372036854775807
    var b: i64 = 1
    a = a + b
    print(a)
}

// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
// @test: requires_system_asm=true
fn main(): void {
    var a: i64 = 9223372036854775807
    var b: i64 = 1
    a = a + b
    print(a)
}

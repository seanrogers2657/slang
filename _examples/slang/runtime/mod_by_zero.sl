// @test: exit_code=1
// @test: stderr_contains=panic: modulo by zero
fn main(): void {
    val a: i64 = 42
    val b: i64 = 0
    val c = a % b
    print(c)
}

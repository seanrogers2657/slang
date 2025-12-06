// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: multiplication
fn main(): void {
    val a: i64 = 9223372036854775807
    val b: i64 = 2
    val c = a * b
    print(c)
}

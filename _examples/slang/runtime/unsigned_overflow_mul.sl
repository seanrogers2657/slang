// @test: exit_code=1
// @test: stderr_contains=panic: unsigned overflow: multiplication
// @test: requires_system_asm=true
fn main(): void {
    val big: u64 = 18446744073709551615
    val two: u64 = 2
    val result = big * two
    print(result)
}

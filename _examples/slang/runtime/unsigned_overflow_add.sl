// @test: exit_code=1
// @test: stderr_contains=panic: unsigned overflow: addition
// @test: requires_system_asm=true
fn main(): void {
    val max: u64 = 18446744073709551615
    val one: u64 = 1
    val result = max + one
    print(result)
}

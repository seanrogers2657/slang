// @test: exit_code=0
// @test: stdout=42\n52\n
// @test: requires_system_asm=true
fn main() {
    val x = 42
    print x
    val y = 10
    val z = x + y
    print z
}

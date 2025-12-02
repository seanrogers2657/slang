// @test: exit_code=0
// @test: stdout=-5\n-42\n-100\n
// @test: requires_system_asm=true
fn main() {
    print 5 - 10
    print 0 - 42
    print 100 - 200
}

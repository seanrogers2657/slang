// @test: exit_code=0
// @test: stdout=42\n0\n123\n42\n42\n42\n2\n1\n1\n1\n1\n
// @test: requires_system_asm=true
fn main() {
    print 42
    print 0
    print 100 + 23
    print 50 - 8
    print 6 * 7
    print 84 / 2
    print 100 % 7
    print 10 == 10
    print 5 != 3
    print 3 < 10
    print 10 > 3
}

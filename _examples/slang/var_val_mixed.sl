// @test: exit_code=0
// @test: stdout=5\n10\n15\n25\n
// @test: requires_system_asm=true
fn main() {
    val constant = 5
    var counter = 10
    print constant
    print counter
    counter = counter + constant
    print counter
    counter = counter + constant + constant
    print counter
}

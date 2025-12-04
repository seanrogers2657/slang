// @test: exit_code=0
// @test: stdout=5\n10\n15\n
fn main(): void {
    var x = 5
    print x
    x = 10
    print x
    x = x + 5
    print x
}

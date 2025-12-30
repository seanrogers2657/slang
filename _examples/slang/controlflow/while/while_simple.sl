// @test: exit_code=0
// @test: stdout=0\n1\n2\n3\n4\n
fn main(): void {
    var i = 0
    while i < 5 {
        print(i)
        i = i + 1
    }
}

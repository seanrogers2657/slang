// @test: exit_code=0
// @test: stdout=25\n
fn square(n: int): int {
    return n * n
}

fn main(): void {
    val result = square(5)
    print result
}

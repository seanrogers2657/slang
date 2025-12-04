// @test: exit_code=0
// @test: stdout=10\n20\n200\n
fn add(a: int, b: int): int {
    return a + b
}

fn multiply(a: int, b: int): int {
    return a * b
}

fn main(): void {
    val x = add(3, 7)
    print x
    val y = multiply(2, 10)
    print y
    val z = multiply(x, y)
    print z
}

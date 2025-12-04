// @test: exit_code=0
// @test: stdout=22\n
fn double(x: int): int {
    return x * 2
}

fn add_one(x: int): int {
    return x + 1
}

fn main(): void {
    val result = double(add_one(double(5)))
    print(result)
}

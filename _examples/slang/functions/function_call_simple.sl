// @test: exit_code=0
// @test: stdout=42\n
fn get_value(): int {
    return 42
}

fn main(): void {
    val result = get_value()
    print(result)
}

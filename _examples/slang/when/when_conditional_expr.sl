// @test: exit_code=42
fn main(): void {
    val x = 5
    val result = when {
        x > 10 -> 100
        x == 5 -> 42
        else -> 0
    }
    exit(result)
}

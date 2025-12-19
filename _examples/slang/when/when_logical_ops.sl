// @test: exit_code=2
// Tests when with logical operators in conditions
fn main(): void {
    val x = 5
    val y = 10
    val result = when {
        x > 10 && y > 10 -> 1
        x > 3 && y > 5 -> 2
        x > 0 || y > 100 -> 3
        else -> 0
    }
    exit(result)
}

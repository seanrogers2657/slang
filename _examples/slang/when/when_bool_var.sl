// @test: exit_code=42
// Tests when with boolean variable in condition
fn main(): void {
    val isLarge = false
    val isSmall = true
    val result = when {
        isLarge -> 100
        isSmall -> 42
        else -> 0
    }
    exit(result)
}

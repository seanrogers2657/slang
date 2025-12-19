// @test: exit_code=99
// Tests that else branch is taken when no conditions match
fn main(): void {
    val x = 1
    val result = when {
        x > 100 -> 1
        x > 50 -> 2
        x > 10 -> 3
        else -> 99
    }
    exit(result)
}

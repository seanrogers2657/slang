// @test: exit_code=99
// Tests that `true` literal makes when exhaustive (no else needed)
fn main(): void {
    val x = 5
    when {
        x > 100 -> exit(1)
        x > 50 -> exit(2)
        true -> exit(99)
    }
}

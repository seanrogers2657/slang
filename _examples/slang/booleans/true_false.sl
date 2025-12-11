// @test: exit_code=1
// Tests boolean variable declarations
fn main(): void {
    val t = true
    val f = false
    // true is 1, false is 0 - exit with true (1)
    exit(1)
}

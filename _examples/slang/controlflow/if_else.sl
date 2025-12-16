// @test: exit_code=1
// Tests if-else statement
fn main(): void {
    val x = 10
    if x > 5 {
        exit(1)
    } else {
        exit(2)
    }
}

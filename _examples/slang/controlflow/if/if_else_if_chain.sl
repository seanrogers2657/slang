// @test: exit_code=3
// Tests longer else-if chain
fn main(): void {
    val x = 15
    if x < 0 {
        exit(1)
    } else if x < 10 {
        exit(2)
    } else if x < 20 {
        exit(3)
    } else {
        exit(4)
    }
}

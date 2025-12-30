// @test: exit_code=10
// Tests if expression with condition
fn main(): void {
    val x = 5
    val result = if x > 3 { 10 } else { 20 }
    exit(result)
}

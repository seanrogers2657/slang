// @test: exit_code=42
// Tests basic if expression assignment
fn main(): void {
    val x = if true { 42 } else { 0 }
    exit(x)
}

// @test: exit_code=42
// Tests basic if statement (condition true)
fn main(): void {
    if true {
        exit(42)
    }
    exit(0)
}

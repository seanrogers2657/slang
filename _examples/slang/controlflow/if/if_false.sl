// @test: exit_code=0
// Tests basic if statement (condition false)
fn main(): void {
    if false {
        exit(42)
    }
    exit(0)
}

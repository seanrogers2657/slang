// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=while-loop condition must be boolean
fn main(): void {
    while 42 {
        print("should not compile")
    }
}

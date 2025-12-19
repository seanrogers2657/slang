// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=when (subject) { } syntax is not supported
fn main(): void {
    val flag = true
    when (flag) {
        true -> exit(1)
        false -> exit(0)
    }
}

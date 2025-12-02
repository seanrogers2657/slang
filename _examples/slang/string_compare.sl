// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires integer operands
fn main() {
    "hello" == "hello"
}

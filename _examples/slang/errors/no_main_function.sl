// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=program must have a 'main' function
fn foo(): void {
    print(42)
}

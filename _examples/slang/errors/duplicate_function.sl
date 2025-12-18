// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=function 'foo' is already declared
fn foo(): void {}

fn foo(): void {}

fn main(): void {}

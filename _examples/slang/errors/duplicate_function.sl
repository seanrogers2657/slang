// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=function 'foo' is already declared
foo = () {}

foo = () {}

main = () {}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=unknown type 'FooBar'
main = () {
    val x: FooBar = 5
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=null cannot be an array literal element
main = () {
    val a = [null, null]
}

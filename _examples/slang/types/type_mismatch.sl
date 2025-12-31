// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires operands of the same type
main = () {
    val x: i32 = 10
    val y: i64 = 20
    val z = x + y
}

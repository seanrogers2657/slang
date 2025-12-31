// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=expected expression after '='
main = () {
    BadStruct = struct {
        val x: i64
    }
    val p = BadStruct{ 10 }
    print(p.x)
}

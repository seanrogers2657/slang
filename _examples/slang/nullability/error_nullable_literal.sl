// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=a literal cannot be nullable
// Nullability belongs to the binding's type, not to a literal.
main = () {
    val x = 5u8?
    print(x)
}

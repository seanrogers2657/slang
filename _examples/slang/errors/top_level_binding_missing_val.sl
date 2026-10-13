// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=expected declaration (Name = (...) or Name = struct/class/object {...}), got 'g'
// A top-level binding without val/var is reported at the binding itself,
// not as a confusing error in the declaration that follows it.
g = 200
main = () {
    print(g)
}

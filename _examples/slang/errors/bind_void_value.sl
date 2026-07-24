// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind a void value
// Regression: a call to a function with no return type produces no value, so
// binding it (val x = f()) must be rejected at the declaration rather than
// silently creating a void-typed variable.
do_thing = () {
    print(1)
}

main = () {
    val x = do_thing()
    print(2)
}

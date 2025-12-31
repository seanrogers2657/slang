// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=must be boolean
main = () {
    for (var i = 0; 5; i = i + 1) {
        print(i)
    }
}

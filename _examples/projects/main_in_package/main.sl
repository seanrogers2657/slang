// @test: expect_error=true
// @test: error_stage=module
// @test: error_contains=must not declare a 'main' function
import "bad"

main = () {
    print(1)
}

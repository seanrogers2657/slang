// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot use package 'math' as a value
import "math"

main = () {
    print(math)
}

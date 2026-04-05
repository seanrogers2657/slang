// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=package 'math' has no declaration 'nonexistent'
import "math"

main = () {
    print(math.nonexistent(1, 2))
}

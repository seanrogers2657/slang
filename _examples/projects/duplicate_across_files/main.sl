// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=already declared
import "math"

main = () {
    print(math.add(1, 2))
}

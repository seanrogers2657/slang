// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=conflicts with import
import "math"

math = () -> s64 {
    return 42
}

main = () {
    print(1)
}

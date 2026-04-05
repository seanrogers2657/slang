// @test: expect_error=true
// @test: error_stage=module
// @test: error_contains=circular dependency
import "a"

main = () {
    print(1)
}

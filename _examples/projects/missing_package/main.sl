// @test: expect_error=true
// @test: error_stage=module
// @test: error_contains=packages
import "nonexistent"

main = () {
    print(1)
}

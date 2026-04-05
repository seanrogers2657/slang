// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=circular initialization
import "config"

main = () {
    print(config.x)
}

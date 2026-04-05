// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=import name 'color' is already used
import "graphics/color"
import "theme/color"

main = () {
    print(1)
}

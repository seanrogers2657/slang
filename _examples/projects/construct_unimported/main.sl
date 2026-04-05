// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined package 'geometry'
import "factory"

main = () {
    // Cannot construct geometry.Point without importing geometry
    val p = geometry.Point{ 1, 2 }
    print(p.x)
}

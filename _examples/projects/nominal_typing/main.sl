// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=expected
import "geometry"
import "physics"

// Takes a geometry.Point, NOT a physics.Vector
use_point = (p: geometry.Point) -> s64 {
    return p.x
}

main = () {
    val v = physics.Vector{ 1, 2 }
    print(use_point(v))
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined package 'geometry'
import "factory"

// Cannot name geometry.Point without importing geometry
make_origin = () -> geometry.Point {
    return factory.make_point(0, 0)
}

main = () {
    val p = make_origin()
    print(p.x)
}

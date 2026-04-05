// @test: stdout=3\n4\n7\n
import "geometry"

main = () {
    val p = geometry.Point{ 3, 4 }
    print(p.x)
    print(p.y)
    print(geometry.sum_coords(p.x, p.y))
}

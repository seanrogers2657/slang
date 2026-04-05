// @test: stdout=10\n20\n
import "factory"

main = () {
    // Can use the returned Point without importing geometry
    val p = factory.make_point(10, 20)
    print(p.x)
    print(p.y)
}

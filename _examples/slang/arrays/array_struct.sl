// @test: exit_code=0
// @test: stdout=1\n2\n3\n4\n5\n6\n
struct Point(val x: i64, val y: i64)

main = () {
    val points = [Point(1, 2), Point(3, 4), Point(5, 6)]
    print(points[0].x)
    print(points[0].y)
    print(points[1].x)
    print(points[1].y)
    print(points[2].x)
    print(points[2].y)
}

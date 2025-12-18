// @test: exit_code=0
// @test: stdout=0\n100\n
struct Point(val x: i64, val y: i64)
struct Rectangle(val topLeft: Point, val bottomRight: Point)

fn main(): void {
    val rect = Rectangle(Point(0, 0), Point(100, 100))
    print(rect.topLeft.x)
    print(rect.bottomRight.x)
}

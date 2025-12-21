// @test: exit_code=0
// @test: stdout=3\n4\n
struct Point(val x: i64, val y: i64)

fn main(): void {
    val points = [Point(1, 2), Point(3, 4), Point(5, 6)]
    var i = 1
    print(points[i].x)
    print(points[i].y)
}

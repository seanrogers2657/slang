// @test: exit_code=0
// @test: stdout=3\n
struct Point(val x: i64, val y: i64)

fn main(): void {
    val points = [Point(1, 2), Point(3, 4), Point(5, 6)]
    print(len(points))
}

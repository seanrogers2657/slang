// @test: exit_code=0
// @test: stdout=38\n
struct Point(
    val x: i64,
    var y: i64,
)

fn main(): void {
    val p = Point(10, 20)
    p.y = 25
    p.y = p.y + 1
    p.y = p.y + 1
    p.y = p.y + 1
    p.y = p.y + p.x
    print(p.y)
}

// @test: exit_code=10
struct Point(var x: i64, var y: i64)

fn mutate(p: Point) {
    p.x = p.x + 12
}

fn main(): void {
    val p = Point(10, 20)
    mutate(p)
    exit(p.x)
}

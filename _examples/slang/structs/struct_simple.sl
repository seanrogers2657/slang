// @test: exit_code=0
// @test: stdout=10\n20\n
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Point{ 10, 20 }
    print(p.x)
    print(p.y)
}

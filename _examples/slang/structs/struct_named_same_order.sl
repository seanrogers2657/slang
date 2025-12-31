// @test: exit_code=0
// @test: stdout=100\n200\n
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Point{ x: 100, y: 200 }
    print(p.x)
    print(p.y)
}

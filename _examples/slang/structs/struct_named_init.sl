// @test: exit_code=0
// @test: stdout=10\n5\n
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Point{ y: 5, x: 10 }
    print(p.x)
    print(p.y)
}

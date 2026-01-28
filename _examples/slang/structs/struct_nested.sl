// @test: exit_code=0
// @test: stdout=0\n100\n
Point = struct {
    val x: s64
    val y: s64
}

Rectangle = struct {
    val top_left: Point
    val bottom_right: Point
}

main = () {
    val rect = Rectangle{ Point{ 0, 0 }, Point{ 100, 100 } }
    print(rect.top_left.x)
    print(rect.bottom_right.x)
}

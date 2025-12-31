// @test: exit_code=0
// @test: stdout=0\n100\n
Point = struct {
    val x: i64
    val y: i64
}

Rectangle = struct {
    val topLeft: Point
    val bottomRight: Point
}

main = () {
    val rect = Rectangle{ Point{ 0, 0 }, Point{ 100, 100 } }
    print(rect.topLeft.x)
    print(rect.bottomRight.x)
}

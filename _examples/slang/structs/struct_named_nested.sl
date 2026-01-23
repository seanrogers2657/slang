// @test: exit_code=0
// @test: stdout=0\n100\n
Point = struct {
    val x: s64
    val y: s64
}

Rectangle = struct {
    val topLeft: Point
    val bottomRight: Point
}

main = () {
    val rect = Rectangle{
        bottomRight: Point{ x: 100, y: 100 },
        topLeft: Point{ y: 0, x: 0 },
    }
    print(
        rect
            .topLeft
            .x
    )
    print(rect.bottomRight.x)
}

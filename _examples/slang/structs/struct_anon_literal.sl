// @test: exit_code=0
// @test: stdout=10\n20\n30\n40\n
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    // Anonymous struct literal with type annotation
    val p1: Point = { x: 10, y: 20 }
    print(p1.x)
    print(p1.y)

    // Can also use named struct literal
    val p2 = Point{ x: 30, y: 40 }
    print(p2.x)
    print(p2.y)
}

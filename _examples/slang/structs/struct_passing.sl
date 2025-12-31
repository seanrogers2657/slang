// @test: exit_code=0
// @test: stdout=10\n
// Note: This tests passing a struct to a function
// Currently tests reading a single field from the passed struct
Point = struct {
    val x: i64
    val y: i64
}

getX = (p: Point) -> i64 {
    return p.x
}

main = () {
    val p = Point{ 10, 20 }
    val result = getX(p)
    print(result)
}

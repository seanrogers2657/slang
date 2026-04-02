// @test: exit_code=0
// @test: stdout=10\n25\n
Point = struct {
    val x: s64
    var y: s64
}

main = () {
    var p = new Point{ 10, 20 }
    print(p.x)
    p.y = 25
    print(p.y)
}

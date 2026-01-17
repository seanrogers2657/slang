// @test: exit_code=0
// @test: stdout=10\n25\n
Point = struct {
    val x: i64
    var y: i64
}

main = () {
    var p = Heap.new(Point{ 10, 20 })
    print(p.x)
    p.y = 25
    print(p.y)
}

// @test: exit_code=0
// @test: stdout=10\n20\n100\n200\n
Point = struct {
    var x: i64
    var y: i64
}

main = () {
    var p = Heap.new(Point{ 10, 20 })
    val q = p.copy()

    // Modify the original
    p.x = 100
    p.y = 200

    // The copy should still have the original values
    print(q.x)
    print(q.y)

    // The original should have the new values
    print(p.x)
    print(p.y)
}

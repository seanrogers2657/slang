// @test: exit_code=0
// @test: stdout=10\n20\n10\n20\n
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Heap.new(Point{ 10, 20 })
    val q = p.copy()

    // Both should have the same values
    print(p.x)
    print(p.y)
    print(q.x)
    print(q.y)
}

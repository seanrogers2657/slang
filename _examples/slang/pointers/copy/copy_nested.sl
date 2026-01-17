// @test: exit_code=0
// @test: stdout=5\n42\n99\n42\n
Point = struct {
    var x: i64
    var y: i64
}

Container = struct {
    val id: i64
    var inner: *Point
}

main = () {
    var c1 = Heap.new(Container{ 5, Heap.new(Point{ 42, 100 }) })
    val c2 = c1.copy()

    // Modify c1's nested pointer
    c1.inner.x = 99

    // Print id (should be same in both)
    print(c2.id)

    // c2's inner should still have original value
    print(c2.inner.x)

    // c1's inner should have new value
    print(c1.inner.x)

    // c2's inner.x should still be 42 (independent copy)
    print(c2.inner.x)
}

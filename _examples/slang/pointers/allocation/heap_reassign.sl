// @test: exit_code=0
// @test: stdout=10\n20\n30\n40\n
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    // First allocation
    var p = Heap.new(Point{ 10, 20 })
    print(p.x)
    print(p.y)

    // Reassign - old value should be freed
    p = Heap.new(Point{ 30, 40 })
    print(p.x)
    print(p.y)

    // p will be freed at function exit
}

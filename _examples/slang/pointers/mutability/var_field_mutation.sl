// @test: exit_code=0
// @test: stdout=10\n100\n
// Test: Var binding can mutate var fields
Point = struct {
    var x: s64
    var y: s64
}

main = () {
    var p = Heap.new(Point{ 10, 20 })
    print(p.x)  // 10

    p.x = 100
    print(p.x)  // 100
}

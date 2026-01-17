// @test: exit_code=0
// @test: stdout=10\n20\n
// Test: Mutable borrowing with &&T
Point = struct {
    var x: i64
    var y: i64
}

doubleX = (p: &&Point) {
    p.x = p.x * 2
}

main = () {
    val p = Heap.new(Point{ 10, 20 })
    print(p.x)  // 10 before mutation

    doubleX(p)  // mutate through reference

    print(p.x)  // 20 after mutation (10 * 2 = 20)
}

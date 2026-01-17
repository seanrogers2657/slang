// @test: exit_code=0
// @test: stdout=100\n
// Test: Val binding CAN create mutable borrow (val only controls reassignment)
// With the && refactor, val/var only control reassignability, not mutability
Point = struct {
    var x: i64
    var y: i64
}

mutatePoint = (p: &&Point) {
    p.x = 100
}

main = () {
    val p = Heap.new(Point{ 10, 20 })  // val binding
    mutatePoint(p)  // OK: val only prevents reassigning p, not mutating through it
    print(p.x)  // 100
}

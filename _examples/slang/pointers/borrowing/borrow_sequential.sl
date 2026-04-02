// @test: exit_code=0
// @test: stdout=4\n
// Test: Sequential mutable borrows are OK (no overlap)
Point = struct {
    var x: s64
}

increment = (p: &&Point) {
    p.x = p.x + 1
}

main = () {
    val p = new Point{ 1 }

    // Sequential mutable borrows - each ends before the next starts
    increment(p)  // First borrow ends when function returns
    increment(p)  // Second borrow - no conflict
    increment(p)  // Third borrow - still OK

    print(p.x)  // 1 + 3 increments = 4
}

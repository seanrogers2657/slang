// @test: exit_code=0
// @test: stdout=3\n
// Test: Multiple immutable borrows of same variable are OK
Point = struct {
    var x: s64
    var y: s64
}

sum = (a: &Point, b: &Point) -> s64 {
    return a.x + b.y
}

main = () {
    val p = new Point{ 1, 2 }

    // Multiple &T borrows of same variable - OK
    val result = sum(p, p)  // Both are immutable borrows

    print(result)  // 1 + 2 = 3
}

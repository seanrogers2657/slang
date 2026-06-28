// @test: exit_code=0
// @test: stdout=30\n15\n15\n
// Test: passing an owned pointer to a &T parameter borrows it — the caller
// keeps ownership and the value stays usable afterwards (no transfer).
Point = struct {
    val x: s64
    val y: s64
}

sum_point = (p: &Point) -> s64 {
    return p.x + p.y
}

main = () {
    val p = new Point{ 10, 20 }
    val sum = sum_point(p)
    print(sum)

    // Create another to verify we can still allocate
    val q = new Point{ 15, 15 }
    print(q.x)
    print(q.y)
}

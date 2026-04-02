// @test: exit_code=0
// @test: stdout=30\n15\n15\n
// Test: Passing *T to a function transfers ownership
Point = struct {
    val x: s64
    val y: s64
}

consumePoint = (p: *Point) -> s64 {
    return p.x + p.y
}

main = () {
    val p = new Point{ 10, 20 }
    val sum = consumePoint(p)
    print(sum)

    // Create another to verify we can still allocate
    val q = new Point{ 15, 15 }
    print(q.x)
    print(q.y)
}

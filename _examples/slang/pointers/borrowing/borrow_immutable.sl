// @test: exit_code=0
// @test: stdout=10\n20\n10\n20\n
// Test: Borrowing with &T (immutable reference)
Point = struct {
    val x: s64
    val y: s64
}

printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
}

main = () {
    val p = new Point{ 10, 20 }
    printPoint(p)  // auto-borrow: *Point -> &Point

    // p is still usable after borrowing
    print(p.x)
    print(p.y)
}

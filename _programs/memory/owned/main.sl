// Tests ownership transfer with *Point
// Demonstrates that ownership can be transferred multiple times

Point = struct {
    var x: s64
    var y: s64
}

transfer = (p: *Point) -> *Point {
    return p
}

main = () {
    var a = new Point{ 2, 3 }
    assert(a.x == 2, "initial x should be 2")
    assert(a.y == 3, "initial y should be 3")

    a = transfer(a)
    assert(a.x == 2, "x should be 2 after transfer 1")

    a = transfer(a)
    assert(a.x == 2, "x should be 2 after transfer 2")

    a = transfer(a)
    assert(a.x == 2, "x should be 2 after transfer 3")

    a = transfer(a)
    assert(a.x == 2, "x should be 2 after transfer 4")

    a = transfer(a)
    assert(a.x == 2, "x should be 2 after transfer 5")

    a = transfer(a)
    assert(a.x == 2, "x should be 2 after transfer 6")
    assert(a.y == 3, "y should still be 3")

    // Pointer still valid after multiple transfers
    // Memory automatically freed when a goes out of scope
    print("Ownership transfer test passed!")
}

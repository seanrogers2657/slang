// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot borrow
// Test: Cannot mix mutable and immutable borrows
Point = struct {
    var x: i64
    var y: i64
}

mixedBorrow = (a: &&Point, b: &Point) {
    a.x = b.x + 1
}

main = () {
    var p = Heap.new(Point{ 10, 20 })
    mixedBorrow(p, p)  // Error: cannot mix mutable and immutable borrows
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot borrow
// Test: Multiple mutable borrows should produce an error
Point = struct {
    var x: s64
    var y: s64
}

mutateBoth = (a: &&Point, b: &&Point) {
    a.x = 1
    b.x = 2
}

main = () {
    var p = Heap.new(Point{ 10, 20 })
    mutateBoth(p, p)  // Error: cannot have two mutable borrows
}

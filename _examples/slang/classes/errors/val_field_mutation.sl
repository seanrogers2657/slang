// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=immutable
// Error: cannot modify val field even with mutable borrow

Point = class {
    val x: s64    // immutable field
    var y: s64    // mutable field

    // Mutable borrow should allow modifying var fields, but not val fields
    try_modify_x = (self: &&Point) {
        self.x = 100  // ERROR: cannot modify immutable field 'x'
    }
}

main = () {
    val p = new Point{ 10, 20 }
    p.try_modify_x()
}

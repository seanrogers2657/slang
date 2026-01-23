// @test: exit_code=25
// Test stack-allocated class instances (no Heap.new)
// This is a future feature

Point = class {
    var x: s64
    var y: s64

    magnitude = (self: &Point) -> s64 {
        return self.x * self.x + self.y * self.y
    }
}

main = () {
    // Stack-allocated instance - future feature
    var p = Point{ 3, 4 }
    val m1 = p.magnitude()
    exit(m1)
}

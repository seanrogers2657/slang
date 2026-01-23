// @test: exit_code=42
// Class with static factory method

Point = class {
    var x: s64
    var y: s64

    // Static factory method
    create = (x: s64, y: s64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    // Instance method
    sum = (self: &Point) -> s64 {
        return self.x + self.y
    }
}

main = () {
    val p = Point.create(20, 22)
    exit(p.sum())
}

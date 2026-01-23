// @test: exit_code=42
// Class with static factory method

Point = class {
    var x: i64
    var y: i64

    // Static factory method
    create = (x: i64, y: i64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    // Instance method
    sum = (self: &Point) -> i64 {
        return self.x + self.y
    }
}

main = () {
    val p = Point.create(20, 22)
    exit(p.sum())
}

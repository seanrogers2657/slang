// @test: exit_code=42
// Test class with val (immutable) fields

Point = class {
    val x: s64   // immutable
    val y: s64   // immutable

    create = (x: s64, y: s64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    // Can read val fields
    sum = (self: &Point) -> s64 {
        return self.x + self.y
    }

    // Can access individual fields
    get_x = (self: &Point) -> s64 {
        return self.x
    }

    get_y = (self: &Point) -> s64 {
        return self.y
    }
}

main = () {
    val p = Point.create(20, 22)
    exit(p.sum())  // 42
}

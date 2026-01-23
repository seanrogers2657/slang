// @test: exit_code=42
// Test class with val (immutable) fields

Point = class {
    val x: i64   // immutable
    val y: i64   // immutable

    create = (x: i64, y: i64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    // Can read val fields
    sum = (self: &Point) -> i64 {
        return self.x + self.y
    }

    // Can access individual fields
    getX = (self: &Point) -> i64 {
        return self.x
    }

    getY = (self: &Point) -> i64 {
        return self.y
    }
}

main = () {
    val p = Point.create(20, 22)
    exit(p.sum())  // 42
}

// @test: exit_code=42
// Test returning class instances by value

Point = class {
    val x: i64
    val y: i64

    // Returns a new Point by value (copied to caller)
    origin = () -> Point {
        return Point{ 0, 0 }
    }

    // Returns a copy of self's data
    clone = (self: &Point) -> Point {
        return Point{ self.x, self.y }
    }

    sum = (self: &Point) -> i64 {
        return self.x + self.y
    }
}

main = () {
    val p1 = Point.origin()
    val p2 = Point{ 20, 22 }
    val p3 = p2.clone()
    exit(p3.sum())  // 42
}

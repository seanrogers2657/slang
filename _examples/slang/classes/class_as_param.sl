// @test: exit_code=42
// Test passing class instances to free functions

Point = class {
    var x: i64
    var y: i64

    create = (x: i64, y: i64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    getX = (self: &Point) -> i64 {
        return self.x
    }
}

// Free function taking immutable borrow of class
sumPoints = (p1: &Point, p2: &Point) -> i64 {
    return p1.x + p1.y + p2.x + p2.y
}

// Free function taking mutable borrow
doublePoint = (p: &&Point) {
    p.x = p.x * 2
    p.y = p.y * 2
}

// Free function returning value derived from class
manhattan = (p: &Point) -> i64 {
    val absX = if p.x < 0 { 0 - p.x } else { p.x }
    val absY = if p.y < 0 { 0 - p.y } else { p.y }
    return absX + absY
}

main = () {
    val p1 = Point.create(5, 7)
    val p2 = Point.create(10, 3)

    // Pass to free function with immutable borrow
    val sum = sumPoints(p1, p2)  // 5 + 7 + 10 + 3 = 25

    // Pass to free function with mutable borrow
    doublePoint(p1)  // p1 becomes (10, 14)

    // Use method after mutation
    val x = p1.getX()  // 10

    // Use another free function
    val dist = manhattan(p1)  // |10| + |14| = 24

    exit(sum + x + dist - 17)  // 25 + 10 + 24 - 17 = 42
}

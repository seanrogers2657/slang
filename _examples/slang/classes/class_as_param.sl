// @test: exit_code=42
// Test passing class instances to free functions

Point = class {
    var x: s64
    var y: s64

    create = (x: s64, y: s64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    get_x = (self: &Point) -> s64 {
        return self.x
    }
}

// Free function taking immutable borrow of class
sum_points = (p1: &Point, p2: &Point) -> s64 {
    return p1.x + p1.y + p2.x + p2.y
}

// Free function taking mutable borrow
double_point = (p: &&Point) {
    p.x = p.x * 2
    p.y = p.y * 2
}

// Free function returning value derived from class
manhattan = (p: &Point) -> s64 {
    val abs_x = if p.x < 0 { 0 - p.x } else { p.x }
    val abs_y = if p.y < 0 { 0 - p.y } else { p.y }
    return abs_x + abs_y
}

main = () {
    val p1 = Point.create(5, 7)
    val p2 = Point.create(10, 3)

    // Pass to free function with immutable borrow
    val sum = sum_points(p1, p2)  // 5 + 7 + 10 + 3 = 25

    // Pass to free function with mutable borrow
    double_point(p1)  // p1 becomes (10, 14)

    // Use method after mutation
    val x = p1.get_x()  // 10

    // Use another free function
    val dist = manhattan(p1)  // |10| + |14| = 24

    exit(sum + x + dist - 17)  // 25 + 10 + 24 - 17 = 42
}

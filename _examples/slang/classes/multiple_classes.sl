// @test: exit_code=75
// Test multiple classes interacting with each other

// First class
Point = class {
    var x: s64
    var y: s64

    create = (x: s64, y: s64) -> *Point {
        return new Point{ x, y }
    }

    get_x = (self: &Point) -> s64 {
        return self.x
    }

    get_y = (self: &Point) -> s64 {
        return self.y
    }
}

// Second class that uses the first
Line = class {
    var length: s64

    create = (len: s64) -> *Line {
        return new Line{ len }
    }

    get_length = (self: &Line) -> s64 {
        return self.length
    }
}

// Free function using multiple classes
compute_distance = (p1: &Point, p2: &Point) -> s64 {
    val dx = p2.get_x() - p1.get_x()
    val dy = p2.get_y() - p1.get_y()
    val abs_dx = if dx < 0 { 0 - dx } else { dx }
    val abs_dy = if dy < 0 { 0 - dy } else { dy }
    return abs_dx + abs_dy
}

main = () {
    val p1 = Point.create(0, 0)
    val p2 = Point.create(10, 20)
    val p3 = Point.create(15, 35)

    // Distance from p1 to p2: |10-0| + |20-0| = 30
    val d1 = compute_distance(p1, p2)

    // Distance from p2 to p3: |15-10| + |35-20| = 5 + 15 = 20
    val d2 = compute_distance(p2, p3)

    // Distance from p1 to p3: |15| + |35| = 50
    val d3 = compute_distance(p1, p3)

    // Create a Line
    val line = Line.create(d1 + d2)  // 30 + 20 = 50

    exit(d1 + d2 + line.get_length() - 25)  // 30 + 20 + 50 - 25 = 75
}

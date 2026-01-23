// @test: exit_code=75
// Test multiple classes interacting with each other

// First class
Point = class {
    var x: i64
    var y: i64

    create = (x: i64, y: i64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    getX = (self: &Point) -> i64 {
        return self.x
    }

    getY = (self: &Point) -> i64 {
        return self.y
    }
}

// Second class that uses the first
Line = class {
    var length: i64

    create = (len: i64) -> *Line {
        return Heap.new(Line{ len })
    }

    getLength = (self: &Line) -> i64 {
        return self.length
    }
}

// Free function using multiple classes
computeDistance = (p1: &Point, p2: &Point) -> i64 {
    val dx = p2.getX() - p1.getX()
    val dy = p2.getY() - p1.getY()
    val absDx = if dx < 0 { 0 - dx } else { dx }
    val absDy = if dy < 0 { 0 - dy } else { dy }
    return absDx + absDy
}

main = () {
    val p1 = Point.create(0, 0)
    val p2 = Point.create(10, 20)
    val p3 = Point.create(15, 35)

    // Distance from p1 to p2: |10-0| + |20-0| = 30
    val d1 = computeDistance(p1, p2)

    // Distance from p2 to p3: |15-10| + |35-20| = 5 + 15 = 20
    val d2 = computeDistance(p2, p3)

    // Distance from p1 to p3: |15| + |35| = 50
    val d3 = computeDistance(p1, p3)

    // Create a Line
    val line = Line.create(d1 + d2)  // 30 + 20 = 50

    exit(d1 + d2 + line.getLength() - 25)  // 30 + 20 + 50 - 25 = 75
}

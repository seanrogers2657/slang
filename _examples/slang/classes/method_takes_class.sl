// @test: exit_code=50
// Test method taking another class instance as parameter

Point = class {
    var x: s64
    var y: s64

    create = (x: s64, y: s64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    getX = (self: &Point) -> s64 {
        return self.x
    }

    getY = (self: &Point) -> s64 {
        return self.y
    }

    setX = (self: &&Point, newX: s64) {
        self.x = newX
    }

    setY = (self: &&Point, newY: s64) {
        self.y = newY
    }

    sum = (self: &Point) -> s64 {
        return self.x + self.y
    }
}

// Free function that takes two class instances
addPoints = (p1: &Point, p2: &Point) -> s64 {
    return p1.getX() + p1.getY() + p2.getX() + p2.getY()
}

main = () {
    val p1 = Point.create(10, 20)
    val p2 = Point.create(5, 5)

    // Use free function with class parameters
    val sum = addPoints(p1, p2)  // 10 + 20 + 5 + 5 = 40

    // Modify p1
    p1.setX(15)  // p1 = (15, 20)

    val newSum = p1.sum()  // 15 + 20 = 35

    exit(sum + newSum - 25)  // 40 + 35 - 25 = 50
}

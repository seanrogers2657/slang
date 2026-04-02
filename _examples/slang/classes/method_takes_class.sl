// @test: exit_code=50
// Test method taking another class instance as parameter

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

    set_x = (self: &&Point, newX: s64) {
        self.x = newX
    }

    set_y = (self: &&Point, newY: s64) {
        self.y = newY
    }

    sum = (self: &Point) -> s64 {
        return self.x + self.y
    }
}

// Free function that takes two class instances
add_points = (p1: &Point, p2: &Point) -> s64 {
    return p1.get_x() + p1.get_y() + p2.get_x() + p2.get_y()
}

main = () {
    val p1 = Point.create(10, 20)
    val p2 = Point.create(5, 5)

    // Use free function with class parameters
    val sum = add_points(p1, p2)  // 10 + 20 + 5 + 5 = 40

    // Modify p1
    p1.set_x(15)  // p1 = (15, 20)

    val new_sum = p1.sum()  // 15 + 20 = 35

    exit(sum + new_sum - 25)  // 40 + 35 - 25 = 50
}

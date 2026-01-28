// @test: exit_code=25
// Test instance method calling another method on self

Point = class {
    val x: s64
    val y: s64

    // Method that calls another method on self
    squared_magnitude = (self: &Point) -> s64 {
        val x_sq = self.get_x_squared()
        val y_sq = self.get_y_squared()
        return x_sq + y_sq
    }

    get_x_squared = (self: &Point) -> s64 {
        return self.x * self.x
    }

    get_y_squared = (self: &Point) -> s64 {
        return self.y * self.y
    }
}

main = () {
    val p = Heap.new(Point{ 3, 4 })
    val mag = p.squared_magnitude()  // 3*3 + 4*4 = 9 + 16 = 25
    exit(mag)
}

// @test: exit_code=25
// Test instance method calling another method on self

Point = class {
    val x: i64
    val y: i64

    // Method that calls another method on self
    squaredMagnitude = (self: &Point) -> i64 {
        val xSq = self.getXSquared()
        val ySq = self.getYSquared()
        return xSq + ySq
    }

    getXSquared = (self: &Point) -> i64 {
        return self.x * self.x
    }

    getYSquared = (self: &Point) -> i64 {
        return self.y * self.y
    }
}

main = () {
    val p = Heap.new(Point{ 3, 4 })
    val mag = p.squaredMagnitude()  // 3*3 + 4*4 = 9 + 16 = 25
    exit(mag)
}

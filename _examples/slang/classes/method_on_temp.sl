// @test: exit_code=25
// Test method call on temporary (literal) value

Point = class {
    val x: i64
    val y: i64

    magnitude = (self: &Point) -> i64 {
        return self.x * self.x + self.y * self.y
    }
}

main = () {
    // Future: Point{ 3, 4 }.magnitude()
    val mag = Point{ 3, 4 }.magnitude()
    exit(mag)
}

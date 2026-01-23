// @test: exit_code=35
// Test nested class composition (class with field of another class type)

Point = class {
    val x: i64
    val y: i64
}

Rectangle = class {
    val width: i64
    val height: i64

    getArea = (self: &Rectangle) -> i64 {
        return self.width * self.height
    }
}

main = () {
    // Create rectangle with direct dimensions
    val rect = Heap.new(Rectangle{ 5, 7 })

    val area = rect.getArea()  // 5 * 7 = 35

    exit(area)  // 35
}

// @test: exit_code=35
// Test nested class composition (class with field of another class type)

Point = class {
    val x: s64
    val y: s64
}

Rectangle = class {
    val width: s64
    val height: s64

    get_area = (self: &Rectangle) -> s64 {
        return self.width * self.height
    }
}

main = () {
    // Create rectangle with direct dimensions
    val rect = Heap.new(Rectangle{ 5, 7 })

    val area = rect.get_area()  // 5 * 7 = 35

    exit(area)  // 35
}

// @test: skip=BUG: nested struct heap allocation is broken
// Error: nested structs in heap-allocated classes store stack pointers instead of values
// The chained field access code (self.field.subfield) is fixed, but heap allocation needs fixing

Point = struct {
    val x: i64
    val y: i64
}

Rectangle = class {
    val topLeft: Point
    val bottomRight: Point

    getWidth = (self: &Rectangle) -> i64 {
        return self.bottomRight.x - self.topLeft.x  // This access pattern is now supported
    }
}

main = () {
    // BUG: Heap.new with nested structs allocates wrong size and stores stack pointers
    val rect = Heap.new(Rectangle{ Point{ 0, 0 }, Point{ 5, 7 } })
    exit(rect.getWidth())
}

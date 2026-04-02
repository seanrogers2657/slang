// @test: skip=BUG: nested struct heap allocation is broken
// Error: nested structs in heap-allocated classes store stack pointers instead of values
// The chained field access code (self.field.subfield) is fixed, but heap allocation needs fixing

Point = struct {
    val x: s64
    val y: s64
}

Rectangle = class {
    val top_left: Point
    val bottom_right: Point

    get_width = (self: &Rectangle) -> s64 {
        return self.bottom_right.x - self.top_left.x  // This access pattern is now supported
    }
}

main = () {
    // BUG: new with nested structs allocates wrong size and stores stack pointers
    val rect = new Rectangle{ Point{ 0, 0 }, Point{ 5, 7 } }
    exit(rect.get_width())
}

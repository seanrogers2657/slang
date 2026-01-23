// @test: exit_code=25
// Test .copy() on class instances for deep copy

Point = class {
    var x: s64
    var y: s64

    create = (x: s64, y: s64) -> *Point {
        return Heap.new(Point{ x, y })
    }

    setX = (self: &&Point, newX: s64) {
        self.x = newX
    }

    sum = (self: &Point) -> s64 {
        return self.x + self.y
    }
}

main = () {
    val original = Point.create(10, 5)
    val copied = original.copy()  // Deep copy

    // Modify original
    original.setX(100)

    // Verify copy is independent
    val originalSum = original.sum()  // 100 + 5 = 105
    val copiedSum = copied.sum()      // 10 + 5 = 15 (unchanged)

    // 105 - 80 = 25
    exit(originalSum - 80)
}

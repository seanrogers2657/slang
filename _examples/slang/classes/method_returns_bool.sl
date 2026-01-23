// @test: exit_code=3
// Test methods returning boolean values

Range = class {
    var min: i64
    var max: i64

    create = (min: i64, max: i64) -> *Range {
        return Heap.new(Range{ min, max })
    }

    // Method returning bool
    contains = (self: &Range, value: i64) -> bool {
        return value >= self.min && value <= self.max
    }

    // Method returning bool
    isEmpty = (self: &Range) -> bool {
        return self.min > self.max
    }

    getMin = (self: &Range) -> i64 {
        return self.min
    }

    getMax = (self: &Range) -> i64 {
        return self.max
    }
}

main = () {
    val r1 = Range.create(0, 10)

    var count: i64 = 0

    // Test contains
    if r1.contains(5) {
        count = count + 1  // should hit
    }
    if r1.contains(15) {
        count = count + 10  // should NOT hit
    }

    // Test isEmpty (use intermediate variable to avoid precedence issue with !)
    val empty = r1.isEmpty()
    if !empty {
        count = count + 1  // should hit (0-10 is not empty)
    }

    // Test with another range
    val r2 = Range.create(5, 15)
    if r2.contains(10) {
        count = count + 1  // should hit
    }

    exit(count)  // 1 + 1 + 1 = 3
}

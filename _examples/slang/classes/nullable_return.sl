// @test: exit_code=22
// Test methods that return nullable types directly

Box = class {
    var value: s64

    // Method returning nullable type
    get_nullable = (self: &Box) -> s64? {
        return self.value
    }

    // Method returning null for zero values
    get_non_zero_or_null = (self: &Box) -> s64? {
        if self.value == 0 {
            return null
        }
        return self.value
    }
}

main = () {
    // Test 1: Method returns non-null value
    val b1 = new Box{ 10 }
    val r1 = b1.get_nullable()
    val v1 = r1 ?: 99  // Should be 10

    // Test 2: Method returns null
    val b2 = new Box{ 0 }
    val r2 = b2.get_non_zero_or_null()
    val v2 = r2 ?: 12  // Should be 12 (null -> default)

    exit(v1 + v2)  // 10 + 12 = 22
}

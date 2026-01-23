// @test: exit_code=42
// Test safe navigation for method calls (?.method())

Counter = class {
    var value: i64

    // Static factory
    create = (initial: i64) -> *Counter {
        return Heap.new(Counter{ initial })
    }

    // Instance method that returns a value
    getValue = (self: &Counter) -> i64 {
        return self.value
    }

    // Instance method that modifies state
    increment = (self: &&Counter) {
        self.value = self.value + 1
    }
}

main = () {
    // Test 1: Safe call on non-null returns the value
    val counter: *Counter? = Counter.create(10)
    val v1 = counter?.getValue()
    // v1 is i64? = 10

    // Test 2: Safe call on null returns null (no crash)
    val nullCounter: *Counter? = null
    val v2 = nullCounter?.getValue()
    // v2 is i64? = null

    // For exit code: v1 should be 10, v2 should be 0 (null)
    // If we can use v1 as the base...
    // We use elvis operator to unwrap nullable
    val result = (v1 ?: 0) + (v2 ?: 100)
    // result = 10 + 100 = 110 if v1 was null (wrong)
    // result = 10 + 0 = 10 if v2 was not null (wrong)
    // result = 10 + 100 = 110 if both work correctly...

    // Simpler: just check that v1 works and v2 didn't crash
    // Using v1: 10, using ?: to provide default
    val base = v1 ?: 0    // Should be 10
    val add = v2 ?: 32    // Should be 32 (null -> default)
    exit(base + add)      // 10 + 32 = 42
}

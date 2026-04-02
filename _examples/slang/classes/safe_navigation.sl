// @test: exit_code=42
// Test safe navigation for method calls (?.method())

Counter = class {
    var value: s64

    // Static factory
    create = (initial: s64) -> *Counter {
        return new Counter{ initial }
    }

    // Instance method that returns a value
    get_value = (self: &Counter) -> s64 {
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
    val v1 = counter?.get_value()
    // v1 is s64? = 10

    // Test 2: Safe call on null returns null (no crash)
    val null_counter: *Counter? = null
    val v2 = null_counter?.get_value()
    // v2 is s64? = null

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

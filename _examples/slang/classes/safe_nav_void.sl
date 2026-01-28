// @test: exit_code=20
// Test safe navigation with void method

Counter = class {
    var value: s64

    create = (initial: s64) -> *Counter {
        return Heap.new(Counter{ initial })
    }

    // Void method - modifies state
    increment = (self: &&Counter) {
        self.value = self.value + 1
    }

    get_value = (self: &Counter) -> s64 {
        return self.value
    }
}

main = () {
    // Test: Safe call on null with void method should be no-op
    val null_counter: *Counter? = null
    null_counter?.increment()  // Should do nothing, not crash

    // Test: Safe call on non-null with void method should work
    val counter: *Counter? = Counter.create(10)
    counter?.increment()  // Should increment to 11
    counter?.increment()  // Should increment to 12

    // Verify: use regular access to check value
    // We need to unwrap - but we can't easily do that without more features
    // So let's just verify we didn't crash and counter works
    val v = counter?.get_value() ?: 0  // Should be 12

    exit(v + 8)  // 12 + 8 = 20
}

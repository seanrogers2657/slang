// @test: exit_code=35
// Test safe navigation with method arguments

Counter = class {
    var value: s64

    create = (initial: s64) -> *Counter {
        return Heap.new(Counter{ initial })
    }

    // Method with argument
    add = (self: &&Counter, x: s64) {
        self.value = self.value + x
    }

    // Method with multiple arguments
    addTwo = (self: &&Counter, x: s64, y: s64) {
        self.value = self.value + x + y
    }

    getValue = (self: &Counter) -> s64 {
        return self.value
    }
}

main = () {
    // Test: Safe call on null with args should be no-op
    val nullCounter: *Counter? = null
    nullCounter?.add(100)  // Should do nothing

    // Test: Safe call on non-null with args should work
    val counter: *Counter? = Counter.create(10)
    counter?.add(5)         // 10 + 5 = 15
    counter?.addTwo(10, 10) // 15 + 10 + 10 = 35

    val v = counter?.getValue() ?: 0  // Should be 35
    exit(v)
}

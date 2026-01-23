// @test: exit_code=35
// Test that multiple class instances are independent

Counter = class {
    var count: s64

    create = (initial: s64) -> *Counter {
        return Heap.new(Counter{ initial })
    }

    increment = (self: &&Counter) {
        self.count = self.count + 1
    }

    add = (self: &&Counter, n: s64) {
        self.count = self.count + n
    }

    get = (self: &Counter) -> s64 {
        return self.count
    }
}

main = () {
    // Create three independent instances
    val c1 = Counter.create(0)
    val c2 = Counter.create(10)
    val c3 = Counter.create(20)

    // Modify c1
    c1.increment()
    c1.increment()
    c1.add(3)  // c1 = 0 + 1 + 1 + 3 = 5

    // Modify c2
    c2.add(5)  // c2 = 10 + 5 = 15

    // c3 unchanged = 20

    // Verify independence: modifying one doesn't affect others
    val sum = c1.get() + c2.get() + c3.get() - 5  // 5 + 15 + 20 - 5 = 35
    exit(sum)
}

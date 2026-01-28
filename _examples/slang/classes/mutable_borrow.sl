// @test: exit_code=50
// Test mutable borrow (&&T) modifying fields

Counter = class {
    var count: s64

    // Mutable borrow - can modify var fields
    add_many = (self: &&Counter, a: s64, b: s64, c: s64) {
        self.count = self.count + a + b + c
    }

    // Immutable borrow - read-only
    get_count = (self: &Counter) -> s64 {
        return self.count
    }
}

main = () {
    val c = Heap.new(Counter{ 0 })

    c.add_many(10, 15, 25)  // 0 + 10 + 15 + 25 = 50

    exit(c.get_count())  // 50
}

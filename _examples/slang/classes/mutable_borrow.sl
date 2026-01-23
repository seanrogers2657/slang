// @test: exit_code=50
// Test mutable borrow (&&T) modifying fields

Counter = class {
    var count: i64

    // Mutable borrow - can modify var fields
    addMany = (self: &&Counter, a: i64, b: i64, c: i64) {
        self.count = self.count + a + b + c
    }

    // Immutable borrow - read-only
    getCount = (self: &Counter) -> i64 {
        return self.count
    }
}

main = () {
    val c = Heap.new(Counter{ 0 })

    c.addMany(10, 15, 25)  // 0 + 10 + 15 + 25 = 50

    exit(c.getCount())  // 50
}

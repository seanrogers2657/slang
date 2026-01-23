// @test: exit_code=25
// Test field mutation through methods

Counter = class {
    var value: i64

    // Read-only access with &Counter
    get = (self: &Counter) -> i64 {
        return self.value
    }

    // Mutating access with &&Counter
    set = (self: &&Counter, v: i64) {
        self.value = v
    }

    // Multiple mutations
    double = (self: &&Counter) {
        self.value = self.value * 2
    }

    addAmount = (self: &&Counter, amount: i64) {
        self.value = self.value + amount
    }
}

main = () {
    val c = Heap.new(Counter{ 10 })
    c.double()         // 20
    c.addAmount(5)     // 25
    exit(c.get())
}

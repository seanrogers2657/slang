// @test: exit_code=25
// Test field mutation through methods

Counter = class {
    var value: s64

    // Read-only access with &Counter
    get = (self: &Counter) -> s64 {
        return self.value
    }

    // Mutating access with &&Counter
    set = (self: &&Counter, v: s64) {
        self.value = v
    }

    // Multiple mutations
    double = (self: &&Counter) {
        self.value = self.value * 2
    }

    add_amount = (self: &&Counter, amount: s64) {
        self.value = self.value + amount
    }
}

main = () {
    val c = Heap.new(Counter{ 10 })
    c.double()         // 20
    c.add_amount(5)     // 25
    exit(c.get())
}

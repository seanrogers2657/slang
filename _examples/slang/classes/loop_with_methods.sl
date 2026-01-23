// @test: exit_code=55
// Test calling methods inside loops

Accumulator = class {
    var total: i64

    create = () -> *Accumulator {
        return Heap.new(Accumulator{ 0 })
    }

    add = (self: &&Accumulator, n: i64) {
        self.total = self.total + n
    }

    get = (self: &Accumulator) -> i64 {
        return self.total
    }

    reset = (self: &&Accumulator) {
        self.total = 0
    }
}

main = () {
    val acc = Accumulator.create()

    // For loop calling methods
    for (var i = 1; i <= 10; i = i + 1) {
        acc.add(i)  // Add 1+2+3+...+10 = 55
    }

    exit(acc.get())
}

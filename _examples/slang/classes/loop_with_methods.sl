// @test: exit_code=55
// Test calling methods inside loops

Accumulator = class {
    var total: s64

    create = () -> *Accumulator {
        return new Accumulator{ 0 }
    }

    add = (self: &&Accumulator, n: s64) {
        self.total = self.total + n
    }

    get = (self: &Accumulator) -> s64 {
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
        acc.add(i)  // Add 1+2+3+...+10 = 55 (add is fine as-is)
    }

    exit(acc.get())  // get is fine as-is
}

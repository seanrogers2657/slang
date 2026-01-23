// @test: exit_code=30
// Test calling different methods based on conditions

Ops = class {
    var value: s64

    create = (v: s64) -> *Ops {
        return Heap.new(Ops{ v })
    }

    add = (self: &&Ops, n: s64) {
        self.value = self.value + n
    }

    sub = (self: &&Ops, n: s64) {
        self.value = self.value - n
    }

    mul = (self: &&Ops, n: s64) {
        self.value = self.value * n
    }

    get = (self: &Ops) -> s64 {
        return self.value
    }
}

main = () {
    val ops = Ops.create(10)

    // Conditional method calls
    val condition1 = true
    if condition1 {
        ops.add(5)  // value = 15
    } else {
        ops.sub(5)
    }

    val condition2 = false
    if condition2 {
        ops.add(100)
    } else {
        ops.mul(2)  // value = 30
    }

    exit(ops.get())  // 30
}

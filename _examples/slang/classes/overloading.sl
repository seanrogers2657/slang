// @test: exit_code=45
// Test method overloading with different argument counts

Calculator = class {
    var value: i64

    // Static factory
    create = () -> *Calculator {
        return Heap.new(Calculator{ 0 })
    }

    // Overload 1: no args - returns current value
    add = (self: &Calculator) -> i64 {
        return self.value
    }

    // Overload 2: one arg - adds to value
    add = (self: &&Calculator, x: i64) {
        self.value = self.value + x
    }

    // Overload 3: two args - adds both to value
    add = (self: &&Calculator, x: i64, y: i64) {
        self.value = self.value + x + y
    }
}

main = () {
    val c = Calculator.create()
    c.add(10)           // add one value: 10
    c.add(15, 20)       // add two values: 10 + 15 + 20 = 45
    exit(c.add())       // get current value: 45
}

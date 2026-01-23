// @test: exit_code=42
// Test that method declaration order doesn't matter

Calculator = class {
    var value: i64

    // Method uses getDouble which is defined later
    addDouble = (self: &&Calculator, x: i64) {
        self.value = self.value + self.getDouble(x)
    }

    // This is defined after addDouble uses it
    getDouble = (self: &Calculator, x: i64) -> i64 {
        return x * 2
    }

    // Static method uses another static defined later
    compute = (a: i64, b: i64) -> i64 {
        return Calculator.helper(a) + Calculator.helper(b)
    }

    // Defined after compute uses it
    helper = (x: i64) -> i64 {
        return x + 1
    }

    getValue = (self: &Calculator) -> i64 {
        return self.value
    }
}

main = () {
    val c = Heap.new(Calculator{ 10 })

    // Test instance method calling later-defined method
    c.addDouble(5)  // value = 10 + 10 = 20
    c.addDouble(6)  // value = 20 + 12 = 32

    // Test static method calling later-defined static
    val r = Calculator.compute(4, 5)  // (4+1) + (5+1) = 5 + 6 = 11

    // Hmm, 32 + 11 = 43, need 42
    exit(c.getValue() + r - 1)  // 32 + 11 - 1 = 42
}

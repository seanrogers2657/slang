// @test: exit_code=42
// Test that method declaration order doesn't matter

Calculator = class {
    var value: s64

    // Method uses get_double which is defined later
    add_double = (self: &&Calculator, x: s64) {
        self.value = self.value + self.get_double(x)
    }

    // This is defined after add_double uses it
    get_double = (self: &Calculator, x: s64) -> s64 {
        return x * 2
    }

    // Static method uses another static defined later
    compute = (a: s64, b: s64) -> s64 {
        return Calculator.helper(a) + Calculator.helper(b)
    }

    // Defined after compute uses it
    helper = (x: s64) -> s64 {
        return x + 1
    }

    get_value = (self: &Calculator) -> s64 {
        return self.value
    }
}

main = () {
    val c = new Calculator{ 10 }

    // Test instance method calling later-defined method
    c.add_double(5)  // value = 10 + 10 = 20
    c.add_double(6)  // value = 20 + 12 = 32

    // Test static method calling later-defined static
    val r = Calculator.compute(4, 5)  // (4+1) + (5+1) = 5 + 6 = 11

    // Hmm, 32 + 11 = 43, need 42
    exit(c.get_value() + r - 1)  // 32 + 11 - 1 = 42
}

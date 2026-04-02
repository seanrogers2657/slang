// @test: exit_code=89
// Test methods with local variables

Calculator = class {
    var base: s64

    // Method with multiple local variables
    compute = (self: &Calculator, x: s64, y: s64) -> s64 {
        val sum = x + y
        val product = x * y
        val combined = sum + product
        return self.base + combined
    }

    // Method with mutable local variable
    accumulate = (self: &Calculator, n: s64) -> s64 {
        var total: s64 = self.base
        var i: s64 = 1
        while i <= n {
            val increment = i * 2
            total = total + increment
            i = i + 1
        }
        return total
    }

    // Method with shadowing (local shadows parameter-like behavior)
    transform = (self: &Calculator, value: s64) -> s64 {
        val temp = value * 2
        val result = temp + self.base
        return result
    }
}

main = () {
    val calc = new Calculator{ 10 }

    // compute: base(10) + (3+5) + (3*5) = 10 + 8 + 15 = 33
    val r1 = calc.compute(3, 5)

    // accumulate: base(10) + 2 + 4 + 6 + 8 = 10 + 20 = 30
    val r2 = calc.accumulate(4)

    // transform: (8 * 2) + 10 = 26
    // r1(33) + r2(30) + r3(26) = 89
    val r3 = calc.transform(8)  // 8*2 + 10 = 26

    exit(r1 + r2 + r3)  // 33 + 30 + 26 = 89
}

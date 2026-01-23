// @test: exit_code=100
// Test methods with complex expressions in body

Calculator = class {
    var a: i64
    var b: i64

    create = (a: i64, b: i64) -> *Calculator {
        return Heap.new(Calculator{ a, b })
    }

    // Method with complex arithmetic expression
    compute = (self: &Calculator, x: i64) -> i64 {
        return (self.a * x + self.b) * (x - 1) + self.a
    }

    // Method with nested conditionals in expression
    complexConditional = (self: &Calculator, x: i64) -> i64 {
        val base = if x > 0 {
            if x > 10 { self.a * 2 } else { self.a }
        } else {
            self.b
        }
        return base + x
    }

    // Method with multiple field accesses in expression
    combine = (self: &Calculator) -> i64 {
        return self.a * self.a + self.b * self.b + self.a * self.b
    }
}

main = () {
    val c = Calculator.create(3, 4)

    // compute(5) = (3*5 + 4) * (5-1) + 3 = 19 * 4 + 3 = 76 + 3 = 79
    val r1 = c.compute(5)

    // complexConditional(5) = base(3, since 5>0 but not >10) + 5 = 8
    val r2 = c.complexConditional(5)

    // combine() = 3*3 + 4*4 + 3*4 = 9 + 16 + 12 = 37
    val r3 = c.combine()

    // Need 100: 79 + 8 + 37 = 124... subtract 24
    exit(r1 + r2 + r3 - 24)
}

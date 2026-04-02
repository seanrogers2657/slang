// @test: exit_code=42
// Test methods with multiple return points (early return)

Validator = class {
    var threshold: s64

    create = (threshold: s64) -> *Validator {
        return new Validator{ threshold }
    }

    // Method with early return
    clamp = (self: &Validator, value: s64) -> s64 {
        if value < 0 {
            return 0
        }
        if value > self.threshold {
            return self.threshold
        }
        return value
    }

    // Method with multiple conditions
    classify = (self: &Validator, value: s64) -> s64 {
        if value < 0 {
            return 0
        }
        if value == 0 {
            return 1
        }
        if value <= self.threshold {
            return 2
        }
        return 3
    }
}

main = () {
    val v = Validator.create(100)

    // Test clamp (using 0-5 instead of -5 for negative)
    val neg5 = 0 - 5
    val r1 = v.clamp(neg5)  // returns 0
    val r2 = v.clamp(42)    // returns 42
    val r3 = v.clamp(200)   // returns 100

    exit(r2)  // 42
}

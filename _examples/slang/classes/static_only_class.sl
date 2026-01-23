// @test: exit_code=42
// Test class with only static methods (utility class pattern)

MathUtils = class {
    var placeholder: i64

    abs = (x: i64) -> i64 {
        if x < 0 {
            return 0 - x
        }
        return x
    }

    max = (a: i64, b: i64) -> i64 {
        if a > b {
            return a
        }
        return b
    }

    min = (a: i64, b: i64) -> i64 {
        if a < b {
            return a
        }
        return b
    }

    clamp = (value: i64, minVal: i64, maxVal: i64) -> i64 {
        return MathUtils.min(MathUtils.max(value, minVal), maxVal)
    }
}

main = () {
    // Test MathUtils (using 0-15 instead of -15)
    val neg15 = 0 - 15
    val a = MathUtils.abs(neg15)      // 15
    val b = MathUtils.max(10, 20)     // 20
    val c = MathUtils.min(10, 20)     // 10
    val d = MathUtils.clamp(50, 0, 30)  // 30

    // 15 + 20 + 10 + 30 = 75, need 42, subtract 33
    exit(a + b + c + d - 33)
}

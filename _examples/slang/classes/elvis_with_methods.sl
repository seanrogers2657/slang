// @test: exit_code=42
// Test elvis operator with nullable method results

Counter = class {
    var value: s64

    create = (v: s64) -> *Counter {
        return new Counter{ v }
    }

    get_value = (self: &Counter) -> s64 {
        return self.value
    }
}

main = () {
    // Elvis with safe method call on non-null
    val c1: *Counter? = Counter.create(10)
    val v1 = c1?.get_value() ?: 0   // 10

    // Elvis with safe method call on null
    val c2: *Counter? = null
    val v2 = c2?.get_value() ?: 32  // 32 (null -> default)

    exit(v1 + v2)  // 10 + 32 = 42
}

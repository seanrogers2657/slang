// @test: exit_code=42
// Test elvis operator with nullable method results

Counter = class {
    var value: i64

    create = (v: i64) -> *Counter {
        return Heap.new(Counter{ v })
    }

    getValue = (self: &Counter) -> i64 {
        return self.value
    }
}

main = () {
    // Elvis with safe method call on non-null
    val c1: *Counter? = Counter.create(10)
    val v1 = c1?.getValue() ?: 0   // 10

    // Elvis with safe method call on null
    val c2: *Counter? = null
    val v2 = c2?.getValue() ?: 32  // 32 (null -> default)

    exit(v1 + v2)  // 10 + 32 = 42
}

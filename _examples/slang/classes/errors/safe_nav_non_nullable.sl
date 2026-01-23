// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=safe navigation
// Test error: using ?. on non-nullable type

Counter = class {
    var value: i64

    create = () -> *Counter {
        return Heap.new(Counter{ 0 })
    }

    getValue = (self: &Counter) -> i64 {
        return self.value
    }
}

main = () {
    val counter = Counter.create()  // *Counter (not nullable!)
    val v = counter?.getValue()     // ERROR: can't use ?. on non-nullable
    exit(v ?: 0)
}

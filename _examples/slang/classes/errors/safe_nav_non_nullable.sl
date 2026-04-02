// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=safe navigation
// Test error: using ?. on non-nullable type

Counter = class {
    var value: s64

    create = () -> *Counter {
        return new Counter{ 0 }
    }

    get_value = (self: &Counter) -> s64 {
        return self.value
    }
}

main = () {
    val counter = Counter.create()  // *Counter (not nullable!)
    val v = counter?.get_value()     // ERROR: can't use ?. on non-nullable
    exit(v ?: 0)
}

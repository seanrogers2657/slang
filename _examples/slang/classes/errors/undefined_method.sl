// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined method
// Error: calling undefined method

Counter = class {
    var count: s64

    create = () -> *Counter {
        return new Counter{ 0 }
    }
}

main = () {
    val c = Counter.create()
    c.increment()  // ERROR: undefined method 'increment'
}

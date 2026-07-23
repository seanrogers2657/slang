// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined method
// Error: calling undefined method

Counter = class {
    var count: s64
}

main = () {
    val c = new Counter{ 0 }
    c.increment()  // ERROR: undefined method 'increment'
}

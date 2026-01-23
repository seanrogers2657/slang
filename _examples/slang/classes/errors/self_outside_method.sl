// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=self
// Error: using 'self' outside a method

Counter = class {
    var count: i64
}

main = () {
    print(self.count)  // ERROR: 'self' can only be used inside a method
}

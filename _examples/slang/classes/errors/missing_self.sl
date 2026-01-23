// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=not a static method
// Error: instance method called on class without instance

Counter = class {
    var count: s64

    // Instance method requires self
    increment = (self: &&Counter) {
        self.count = self.count + 1
    }

    get = (self: &Counter) -> s64 {
        return self.count
    }
}

main = () {
    // Error: trying to call instance method as static (no self)
    Counter.increment()
}

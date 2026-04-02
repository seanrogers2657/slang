// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=argument
// Error: wrong number of arguments to method

Counter = class {
    var count: s64

    create = () -> *Counter {
        return new Counter{ 0 }
    }

    add = (self: &&Counter, x: s64) {
        self.count = self.count + x
    }
}

main = () {
    val c = Counter.create()
    c.add()  // ERROR: missing argument
}

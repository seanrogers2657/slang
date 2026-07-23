// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=argument
// Error: wrong number of arguments to method

Counter = class {
    var count: s64

    add = (self: &&Counter, x: s64) {
        self.count = self.count + x
    }
}

main = () {
    val c = new Counter{ 0 }
    c.add()  // ERROR: missing argument
}

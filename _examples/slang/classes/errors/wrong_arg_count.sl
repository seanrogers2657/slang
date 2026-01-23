// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=argument
// Error: wrong number of arguments to method

Counter = class {
    var count: i64

    create = () -> *Counter {
        return Heap.new(Counter{ 0 })
    }

    add = (self: &&Counter, x: i64) {
        self.count = self.count + x
    }
}

main = () {
    val c = Counter.create()
    c.add()  // ERROR: missing argument
}

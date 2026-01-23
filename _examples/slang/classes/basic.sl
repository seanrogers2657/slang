// @test: exit_code=3
// Basic class with instance methods

Counter = class {
    var count: s64

    increment = (self: &&Counter) {
        self.count = self.count + 1
    }

    getCount = (self: &Counter) -> s64 {
        return self.count
    }
}

main = () {
    val c = Heap.new(Counter{ 0 })
    c.increment()
    c.increment()
    c.increment()
    exit(c.getCount())
}

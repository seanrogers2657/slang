// @test: exit_code=30
// Instance method call on a class

Counter = class {
    var count: s64

    get_count = (self: &Counter) -> s64 {
        return self.count
    }

    add = (self: &Counter, n: s64) -> s64 {
        return self.count + n
    }
}

main = () {
    val c = Heap.new(Counter{ 10 })
    val result = c.add(20)
    exit(result)
}

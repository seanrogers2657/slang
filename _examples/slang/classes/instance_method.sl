// @test: exit_code=30
// Instance method call on a class

Counter = class {
    var count: i64

    getCount = (self: &Counter) -> i64 {
        return self.count
    }

    add = (self: &Counter, n: i64) -> i64 {
        return self.count + n
    }
}

main = () {
    val c = Heap.new(Counter{ 10 })
    val result = c.add(20)
    exit(result)
}

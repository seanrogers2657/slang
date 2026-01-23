// @test: exit_code=30
// Test chained method calls

Builder = class {
    var value: i64

    create = () -> *Builder {
        return Heap.new(Builder{ 0 })
    }

    // Methods that return self for chaining
    add = (self: &&Builder, x: i64) -> *Builder {
        self.value = self.value + x
        return Heap.new(Builder{ self.value })
    }

    getValue = (self: &Builder) -> i64 {
        return self.value
    }
}

main = () {
    // Chained method calls
    val b = Builder.create()
    val result = b.add(10).add(20).getValue()  // 0 + 10 = 10, 10 + 20 = 30
    exit(result)
}

// @test: exit_code=30
// Test chained method calls

Builder = class {
    var value: s64

    create = () -> *Builder {
        return new Builder{ 0 }
    }

    // Methods that return self for chaining
    add = (self: &&Builder, x: s64) -> *Builder {
        self.value = self.value + x
        return new Builder{ self.value }
    }

    get_value = (self: &Builder) -> s64 {
        return self.value
    }
}

main = () {
    // Chained method calls
    val b = Builder.create()
    val result = b.add(10).add(20).get_value()  // 0 + 10 = 10, 10 + 20 = 30
    exit(result)
}

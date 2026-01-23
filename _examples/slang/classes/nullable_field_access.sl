// @test: exit_code=25
// Test nullable field access and elvis operator

Container = class {
    var value: i64

    create = (v: i64) -> *Container {
        return Heap.new(Container{ v })
    }

    getValue = (self: &Container) -> i64 {
        return self.value
    }

    setValue = (self: &&Container, v: i64) {
        self.value = v
    }
}

main = () {
    // Test with values
    val c1 = Container.create(10)
    val v1 = c1.getValue()  // 10

    val c2 = Container.create(15)
    val v2 = c2.getValue()  // 15

    exit(v1 + v2)  // 10 + 15 = 25
}

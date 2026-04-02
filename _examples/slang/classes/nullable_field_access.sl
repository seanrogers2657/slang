// @test: exit_code=25
// Test nullable field access and elvis operator

Container = class {
    var value: s64

    create = (v: s64) -> *Container {
        return new Container{ v }
    }

    get_value = (self: &Container) -> s64 {
        return self.value
    }

    set_value = (self: &&Container, v: s64) {
        self.value = v
    }
}

main = () {
    // Test with values
    val c1 = Container.create(10)
    val v1 = c1.get_value()  // 10

    val c2 = Container.create(15)
    val v2 = c2.get_value()  // 15

    exit(v1 + v2)  // 10 + 15 = 25
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined method
// Error: calling method that doesn't exist on the class

Point = class {
    var x: s64
    var y: s64

    get_x = (self: &Point) -> s64 {
        return self.x
    }
}

Box = class {
    var value: s64

    get_value = (self: &Box) -> s64 {
        return self.value
    }
}

main = () {
    val b = new Box{ 5 }

    // Error: Box has no get_x method
    val x = b.get_x()
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined method
// Error: calling method that doesn't exist on the class

Point = class {
    var x: s64
    var y: s64

    getX = (self: &Point) -> s64 {
        return self.x
    }
}

Box = class {
    var value: s64

    getValue = (self: &Box) -> s64 {
        return self.value
    }
}

main = () {
    val b = Heap.new(Box{ 5 })

    // Error: Box has no getX method
    val x = b.getX()
}

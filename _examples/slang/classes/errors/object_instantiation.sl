// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot instantiate
// Error: attempting to instantiate an object (singleton)

Math = object {
    add = (a: s64, b: s64) -> s64 {
        return a + b
    }
}

main = () {
    val m = Math{}  // ERROR: cannot instantiate object
    exit(0)
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=of type 's64[] of length 3', got 's64[] of length 1'
// Arrays have a fixed length, so an elvis default must match it.
main = () {
    val a: s64[]? = [1, 2, 3]
    print(len(a ?: [0]))
}

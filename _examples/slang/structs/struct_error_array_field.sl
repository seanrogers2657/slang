// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=arrays cannot be used as a struct field
// An array type has no size, so an array field would have no layout.
S = struct {
    var arr: s64[]
}

main = () {
    val s = S{ [1, 2] }
    print(s.arr[0])
}

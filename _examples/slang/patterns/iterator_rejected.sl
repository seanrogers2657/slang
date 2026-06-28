// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot be used as a struct field
// Iterator, natural form: a cursor object that holds a reference to the
// collection. Rejected — a borrow cannot be stored in a field.
// (See iterator_ok.sl: external index-based iteration.)
Iter = struct {
    val data: &vec
    var pos: s64
}

main = () {
    var v = vec()
    push(v, 1)
    var it = Iter{ v, 0 }
    print(it.pos)
}

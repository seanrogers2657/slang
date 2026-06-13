// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot move an owned value out of a struct field
// Regression: moving an owned pointer out of a struct field left the struct
// still owning the same pointer, causing a double free. It must be rejected at
// compile time.
P = struct { var x: s64 }
Box = struct { var p: *P }

consume = (p: *P) -> s64 {
    return p.x
}

main = () {
    var b = Box{ new P{ 7 } }
    print(consume(b.p))   // Error: cannot move out of a struct field
}

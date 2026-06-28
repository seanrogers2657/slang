// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=owned pointers (*T) cannot be used as a struct field
// An owned pointer (*T) cannot be used as a struct field, so a type like Box
// below is rejected outright. This also prevents the old double-free hazard of
// moving an owned pointer out of a struct field.
P = struct { var x: s64 }
Box = struct { var p: *P }  // Error: *T cannot be a struct field

consume = (p: *P) -> s64 {
    return p.x
}

main = () {
    var b = Box{ new P{ 7 } }
    print(consume(b.p))
}

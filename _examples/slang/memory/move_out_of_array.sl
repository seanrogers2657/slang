// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=owned pointers (*T) cannot be parameters
// An owned pointer (*T) cannot be a function parameter, so a consumer like
// the one below is rejected outright. This also prevents the old double-free
// hazard of moving an owned pointer out of an array element.
P = struct { var x: s64 }

consume = (p: *P) -> s64 {  // Error: *T cannot be a parameter
    return p.x
}

main = () {
    var arr = [new P{ 1 }, new P{ 2 }]
    print(consume(arr[0]))
}

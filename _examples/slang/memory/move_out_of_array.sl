// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot move an owned value out of an array element
// Regression: moving an owned pointer out of an array element left the array
// still owning the same pointer, so both the callee and the array freed it
// (double free). It must be rejected at compile time.
P = struct { var x: s64 }

consume = (p: *P) -> s64 {
    return p.x
}

main = () {
    var arr = [new P{ 1 }, new P{ 2 }]
    print(consume(arr[0]))   // Error: cannot move out of an array element
}

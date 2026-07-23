// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Assigning an owner into an array element (arr[i] = p) would alias it: both the
// element slot and the binding 'p' would free the same allocation at scope exit
// (a double free). Use arr[i] = p.copy() or a fresh `new` instead.
P = struct { var x: s64 }

main = () {
    var arr = [new P{ 1 }, new P{ 2 }]
    var p = new P{ 9 }
    arr[0] = p   // Error: cannot alias an owner into the array
    print(arr[0].x)
}

// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// An owned-pointer array literal may only hold freshly produced elements
// (new/.copy()), never aliases of existing owners. Listing 'p' as an element
// would make both the array slot and 'p' free the same allocation. (Repeating
// the same owner, [p, p], is the clearest double free.)
P = struct { var x: s64 }

main = () {
    var p = new P{ 5 }
    var arr = [p, p]   // Error: cannot alias an owner into the array
    print(arr[0].x)
}

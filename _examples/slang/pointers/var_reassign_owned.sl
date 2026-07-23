// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Single ownership: reassigning an owned-pointer variable from another
// owned-pointer variable (a = b) would create two owners of the same
// allocation, so it is rejected. Reassign from a fresh value (new / .copy())
// instead — see heap_reassign.sl for the allowed `p = new ...` form.

Box = struct {
    val v: s64
}

main = () {
    var a = new Box{ 1 }
    var b = new Box{ 2 }
    a = b        // Error: cannot alias owned pointer 'b'
    print(a.v)
}

// @test: exit_code=0
// @test: stdout=row 0\nrow 1\nrow 2\n
// Interpolation inside a loop: each iteration allocates and frees its string,
// so the heap-balance assertion at exit must still pass.
main = () {
    for (var i = 0; i < 3; i = i + 1) {
        print("row ${i}")
    }
}

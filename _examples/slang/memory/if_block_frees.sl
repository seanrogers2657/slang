// @test: exit_code=0
// @test: stdout=5\n0\n
// Regression: an allocation made inside an if/else branch is freed at the end
// of that branch (the heap stays balanced), not leaked until function return.
Big = struct { var x: s64 }

main = () {
    if true {
        val b = new Big{ 5 }   // owned by the then-branch
        print(b.x)
    }                          // b freed here
    print(0)
}

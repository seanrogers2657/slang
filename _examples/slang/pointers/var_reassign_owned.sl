// @test: exit_code=0
// @test: stdout=2\n
// Reassigning a var of owned-pointer type from another owned-pointer
// variable transfers ownership: the source must not double-free at scope
// exit, and the old destination value must be released.

Box = struct {
    val v: s64
}

main = () {
    var a = new Box{ 1 }
    var b = new Box{ 2 }
    a = b
    print(a.v)
}

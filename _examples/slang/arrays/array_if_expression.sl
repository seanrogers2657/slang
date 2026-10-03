// @test: exit_code=0
// @test: stdout=2\n5\n
// An array-valued if expression merges array pointers in its phi.
main = () {
    val c = true
    val a = if c { [1, 2] } else { [3, 4] }
    print(a[1])
    val b = [5, 6]
    val d = if c { b } else { [7, 8] }
    print(d[0])
}

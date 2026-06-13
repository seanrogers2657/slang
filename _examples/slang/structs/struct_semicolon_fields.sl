// @test: exit_code=0
// @test: stdout=1\n2\n3\n
// Struct fields may be separated by semicolons on a single line (in addition
// to newlines). Regression test: semicolon separators previously sent the
// parser into an infinite loop.
P = struct { val x: s64; val y: s64; val z: s64 }

main = () {
    val a = P{ 1, 2, 3 }
    print(a.x)
    print(a.y)
    print(a.z)
}

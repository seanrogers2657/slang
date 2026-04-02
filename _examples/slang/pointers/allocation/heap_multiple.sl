// @test: exit_code=0
// @test: stdout=1\n2\n3\n
Num = struct {
    val v: s64
}

main = () {
    // Multiple allocations - all should be freed at exit
    val a = new Num{ 1 }
    val b = new Num{ 2 }
    val c = new Num{ 3 }

    print(a.v)
    print(b.v)
    print(c.v)
}

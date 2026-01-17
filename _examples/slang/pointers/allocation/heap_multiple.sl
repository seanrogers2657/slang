// @test: exit_code=0
// @test: stdout=1\n2\n3\n
Num = struct {
    val v: i64
}

main = () {
    // Multiple allocations - all should be freed at exit
    val a = Heap.new(Num{ 1 })
    val b = Heap.new(Num{ 2 })
    val c = Heap.new(Num{ 3 })

    print(a.v)
    print(b.v)
    print(c.v)
}

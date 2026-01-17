// @test: exit_code=0
// @test: stdout=1\n2\n
// Test: Mutually-referential structs (forward references work regardless of order)
TypeA = struct {
    val id: i64
    var other: *TypeB?
}

TypeB = struct {
    val id: i64
    var other: *TypeA?
}

main = () {
    var a = Heap.new(TypeA{ 1, null })
    var b = Heap.new(TypeB{ 2, null })

    print(a.id)  // 1
    print(b.id)  // 2
}

// @test: exit_code=0
// @test: stdout=10\n
// Test: Self-referential struct (linked list node)
Node = struct {
    val value: i64
    var next: *Node?
}

main = () {
    var head = Heap.new(Node{ 10, null })
    print(head.value)  // 10
}

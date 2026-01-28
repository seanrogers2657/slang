// @test: exit_code=0
// @test: stdout=Root:\n42\nLeft:\n10\n
// Regression test: Safe call access through nullable owned pointer (*T?)
// Tests that ?.field properly unwraps *T? to *T before accessing field

Node = struct {
    var left: *Node?
    val value: s64
}

main = () {
    val root = Heap.new(Node{ Heap.new(Node{ null, 10 }), 42 })

    print("Root:")
    print(root.value)

    // This is the key test: root.left is *Node?
    // ?.value should unwrap to *Node, then access value field
    val left_val = root.left?.value

    if left_val != null {
        print("Left:")
        // Use elvis operator to unwrap
        val v = left_val ?: 0
        print(v)
    }
}

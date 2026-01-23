// @test: exit_code=30
// Test class with pointer field to another class

Node = class {
    val value: s64
    var next: *Node?

    // Static factory
    create = (v: s64) -> *Node {
        return Heap.new(Node{ v, null })
    }

    getValue = (self: &Node) -> s64 {
        return self.value
    }

    setNext = (self: &&Node, n: *Node?) {
        self.next = n
    }
}

main = () {
    // Create a simple linked structure
    val n1 = Node.create(10)
    val n2 = Node.create(20)

    // Link them (n1 -> n2)
    n1.setNext(n2)

    // Read values
    val v1 = n1.getValue()  // 10
    val v2 = n2.getValue()  // 20

    exit(v1 + v2)  // 30
}

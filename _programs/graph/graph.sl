// A graph-like structure demonstrating self-referential types,
// ownership, heap allocation, and deep copy.

// A graph node with a value and edges to other nodes
GraphNode = struct {
    val id: i64
    var data: i64
    var edge1: *GraphNode?
    var edge2: *GraphNode?
}

// Create a new graph node on the heap
createNode = (id: i64, data: i64) -> *GraphNode {
    return Heap.new(GraphNode{ id, data, null, null })
}

// Print node info
printNode = (node: &GraphNode) {
    print(node.id)
    print(node.data)
}

main = () {
    // Create a small graph:
    //
    //     [1:100] --> [2:200]
    //        |
    //        v
    //     [3:300]

    var node3 = createNode(3, 300)
    var node2 = createNode(2, 200)
    var node1 = createNode(1, 100)

    // Connect the graph (ownership transfers)
    node1.edge1 = node2  // node1 -> node2
    node1.edge2 = node3  // node1 -> node3

    // Print and mutate root node
    printNode(node1)   // 1, 100
    node1.data = 111
    print(node1.data)  // 111

    // Deep copy the entire graph
    // copy() recursively copies all *T? fields
    var graphCopy = node1.copy()

    // Modify the copy - original should be unchanged
    graphCopy.data = 999
    print(node1.data)    // 111 (original unchanged)
    print(graphCopy.data) // 999

    // Replace an edge in the copy - original unaffected
    graphCopy.edge1 = createNode(42, 42)

    // Original graph is still intact
    print(node1.data)  // 111

    print(1)  // success
}

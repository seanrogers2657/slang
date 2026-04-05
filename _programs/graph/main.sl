// A graph-like structure demonstrating self-referential types,
// ownership, heap allocation, and deep copy.

// A graph node with a value and edges to other nodes
GraphNode = struct {
    val id: s64
    var data: s64
    var edge1: *GraphNode?
    var edge2: *GraphNode?
}

// Create a new graph node on the heap
create_node = (id: s64, data: s64) -> *GraphNode {
    return new GraphNode{ id, data, null, null }
}

// Print node info
print_node = (node: &GraphNode) {
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

    var node3 = create_node(3, 300)
    var node2 = create_node(2, 200)
    var node1 = create_node(1, 100)

    assert(node1.id == 1, "node1 id should be 1")
    assert(node1.data == 100, "node1 data should be 100")
    assert(node2.id == 2, "node2 id should be 2")
    assert(node3.id == 3, "node3 id should be 3")

    // Connect the graph (ownership transfers)
    node1.edge1 = node2  // node1 -> node2
    node1.edge2 = node3  // node1 -> node3

    // Print and mutate root node
    print_node(node1)   // 1, 100
    node1.data = 111
    assert(node1.data == 111, "node1 data should be 111 after mutation")
    print(node1.data)  // 111

    // Deep copy the entire graph
    // copy() recursively copies all *T? fields
    var graph_copy = node1.copy()
    assert(graph_copy.id == 1, "copy id should be 1")
    assert(graph_copy.data == 111, "copy data should be 111")

    // Modify the copy - original should be unchanged
    graph_copy.data = 999
    assert(node1.data == 111, "original should be unchanged after modifying copy")
    assert(graph_copy.data == 999, "copy data should be 999")
    print(node1.data)    // 111 (original unchanged)
    print(graph_copy.data) // 999

    // Replace an edge in the copy - original unaffected
    graph_copy.edge1 = create_node(42, 42)

    // Original graph is still intact
    assert(node1.data == 111, "original data should still be 111")
    print(node1.data)  // 111

    print("Graph test passed!")
}

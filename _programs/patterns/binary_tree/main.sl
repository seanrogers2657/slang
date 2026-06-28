// Binary Tree Pattern (growable-vec arena form)
// A recursive data structure under the scope-frees-it model: one scope owns a
// growable vec of node fields and each child link is an integer index into it
// (-1 = no child). Nodes are appended dynamically. This replaces recursive
// owned-pointer fields (var left: *TreeNode?) with value handles, so there are
// no owned-pointer fields and no moves. The whole tree (three vecs) frees at
// once when main returns.

// Append a node (value, left, right) to the arena and return its index.
add_node = (value: vec, left: vec, right: vec, v: s64, l: s64, r: s64) -> s64 {
    push(value, v)
    push(left, l)
    push(right, r)
    return len(value) - 1
}

main = () {
    var value = vec()
    var left = vec()
    var right = vec()

    // Build a binary tree, leaves first so children exist before their parents:
    //        4
    //       / \
    //      2   6
    //     / \ / \
    //    1  3 5  7
    val n1 = add_node(value, left, right, 1, 0 - 1, 0 - 1)
    val n3 = add_node(value, left, right, 3, 0 - 1, 0 - 1)
    val n2 = add_node(value, left, right, 2, n1, n3)
    val n5 = add_node(value, left, right, 5, 0 - 1, 0 - 1)
    val n7 = add_node(value, left, right, 7, 0 - 1, 0 - 1)
    val n6 = add_node(value, left, right, 6, n5, n7)
    val root = add_node(value, left, right, 4, n2, n6)

    assert(get(value, root) == 4, "root value should be 4")
    print("Root value:")
    print(get(value, root))

    val l = get(left, root)
    val r = get(right, root)
    assert(get(value, l) == 2, "left child should be 2")
    assert(get(value, r) == 6, "right child should be 6")
    print("Left child exists")
    print("Right child exists")

    assert(get(value, get(left, l)) == 1, "left-left grandchild should be 1")
    assert(get(value, get(right, l)) == 3, "left-right grandchild should be 3")
    assert(get(value, get(left, r)) == 5, "right-left grandchild should be 5")
    assert(get(value, get(right, r)) == 7, "right-right grandchild should be 7")
    print("Left-left grandchild exists")
    print("Left-right grandchild exists")
    print("Right-left grandchild exists")
    print("Right-right grandchild exists")

    print("Binary tree test passed!")
    // The arena (three vecs) is freed when main returns.
}

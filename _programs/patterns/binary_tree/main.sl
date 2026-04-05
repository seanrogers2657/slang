// Binary Tree Pattern
// Demonstrates: Recursive data structures with *T?, safe call access, function returns

TreeNode = struct {
    var left: *TreeNode?
    var right: *TreeNode?
    val value: s64
}

// Create a leaf node (no children)
leaf = (value: s64) -> *TreeNode {
    return new TreeNode{ null, null, value }
}

// Create a node with children (takes ownership)
node = (left: *TreeNode?, right: *TreeNode?, value: s64) -> *TreeNode {
    return new TreeNode{ left, right, value }
}

main = () {
    // Build a binary tree:
    //        4
    //       / \
    //      2   6
    //     / \ / \
    //    1  3 5  7

    val tree = node(
        node(leaf(1), leaf(3), 2),
        node(leaf(5), leaf(7), 6),
        4
    )

    // Access root value directly (tree is *TreeNode, not nullable)
    assert(tree.value == 4, "root value should be 4")
    print("Root value:")
    print(tree.value)

    // Safe call to access nullable children
    val left_val = tree.left?.value
    val right_val = tree.right?.value

    assert(left_val != null, "left child should exist")
    assert(right_val != null, "right child should exist")
    if left_val != null {
        print("Left child exists")
    }
    if right_val != null {
        print("Right child exists")
    }

    // Deeper access with chained safe calls
    val left_left_val = tree.left?.left?.value
    val left_right_val = tree.left?.right?.value
    val right_left_val = tree.right?.left?.value
    val right_right_val = tree.right?.right?.value

    assert(left_left_val != null, "left-left grandchild should exist")
    assert(left_right_val != null, "left-right grandchild should exist")
    assert(right_left_val != null, "right-left grandchild should exist")
    assert(right_right_val != null, "right-right grandchild should exist")

    if left_left_val != null { print("Left-left grandchild exists") }
    if left_right_val != null { print("Left-right grandchild exists") }
    if right_left_val != null { print("Right-left grandchild exists") }
    if right_right_val != null { print("Right-right grandchild exists") }

    print("Done - entire tree freed automatically")
    print("Binary tree test passed!")
    // All 7 nodes freed when tree goes out of scope
}

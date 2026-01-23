// Binary Tree Pattern
// Demonstrates: Recursive data structures with *T?, safe call access, function returns

TreeNode = struct {
    var left: *TreeNode?
    var right: *TreeNode?
    val value: s64
}

// Create a leaf node (no children)
leaf = (value: s64) -> *TreeNode {
    return Heap.new(TreeNode{ null, null, value })
}

// Create a node with children (takes ownership)
node = (left: *TreeNode?, right: *TreeNode?, value: s64) -> *TreeNode {
    return Heap.new(TreeNode{ left, right, value })
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
    print("Root value:")
    print(tree.value)

    // Safe call to access nullable children
    val leftVal = tree.left?.value
    val rightVal = tree.right?.value

    if leftVal != null {
        print("Left child exists")
    }
    if rightVal != null {
        print("Right child exists")
    }

    // Deeper access with chained safe calls
    val leftLeftVal = tree.left?.left?.value
    val leftRightVal = tree.left?.right?.value
    val rightLeftVal = tree.right?.left?.value
    val rightRightVal = tree.right?.right?.value

    if leftLeftVal != null { print("Left-left grandchild exists") }
    if leftRightVal != null { print("Left-right grandchild exists") }
    if rightLeftVal != null { print("Right-left grandchild exists") }
    if rightRightVal != null { print("Right-right grandchild exists") }

    print("Done - entire tree freed automatically")
    // All 7 nodes freed when tree goes out of scope
}

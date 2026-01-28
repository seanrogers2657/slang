// Linked List Pattern
// Demonstrates: *T?, building lists, safe call traversal (?.)
//
// NOTE: Full recursive traversal requires smart casts (planned feature).
// This example shows safe call chaining which is currently supported.

Node = struct {
    var next: *Node?
    val value: s64
}

// Build a list by prepending nodes
prepend = (head: *Node?, value: s64) -> *Node {
    return Heap.new(Node{ head, value })
}

main = () {
    // Build list: 5 -> 4 -> 3 -> 2 -> 1 -> null
    var list: *Node? = null
    list = prepend(list, 1)
    list = prepend(list, 2)
    list = prepend(list, 3)
    list = prepend(list, 4)
    list = prepend(list, 5)

    // Access head value using safe call
    val head_val = list?.value
    assert(head_val != null, "head value should exist")
    if head_val != null {
        print("Head value exists")
    }

    // Safe call chaining - access values through nullable pointers
    val v1 = list?.value
    val v2 = list?.next?.value
    val v3 = list?.next?.next?.value
    val v4 = list?.next?.next?.next?.value
    val v5 = list?.next?.next?.next?.next?.value
    val v6 = list?.next?.next?.next?.next?.next?.value  // null - past end

    // Verify all values exist except v6
    assert(v1 != null, "v1 should exist")
    assert(v2 != null, "v2 should exist")
    assert(v3 != null, "v3 should exist")
    assert(v4 != null, "v4 should exist")
    assert(v5 != null, "v5 should exist")
    assert(v6 == null, "v6 should be null (past end)")

    // Check which values exist
    if v1 != null { print("v1 exists") }
    if v2 != null { print("v2 exists") }
    if v3 != null { print("v3 exists") }
    if v4 != null { print("v4 exists") }
    if v5 != null { print("v5 exists") }
    if v6 == null { print("v6 is null (past end of list)") }

    print("Done - list will be freed automatically")
    print("Linked list test passed!")
    exit(0)
    // All 5 nodes freed when list goes out of scope
}

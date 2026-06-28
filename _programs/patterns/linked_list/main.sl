// Linked List Pattern (growable-vec arena form)
// The scope-frees-it idiom that replaces owned-pointer fields: one scope owns a
// growable vec of node fields and the "next pointer" is an integer index into it
// (-1 = end of list). Nodes are appended dynamically (the vec reallocates as it
// grows). No node owns another node, so there are no owned-pointer fields and no
// moves. The whole arena (both vecs) frees at once when main returns.

main = () {
    var value = vec()   // value of node i
    var next = vec()    // index of node i's successor (-1 = end)
    var head = 0 - 1

    // Build by prepending 1..5  =>  head -> 5 -> 4 -> 3 -> 2 -> 1.
    // "Allocating" is push (the vec grows as needed); "linking" writes an index.
    var i = 1
    while i <= 5 {
        push(value, i)
        push(next, head)
        head = len(value) - 1
        i = i + 1
    }

    // Head value is the last prepended (5).
    assert(get(value, head) == 5, "head value should be 5")
    print("Head value exists")

    // Traverse from the head index to the end, printing each value.
    var cur = head
    while cur != 0 - 1 {
        print(get(value, cur))
        cur = get(next, cur)
    }

    print("Linked list test passed!")
    exit(0)
    // The arena (both vecs) is freed when main returns.
}

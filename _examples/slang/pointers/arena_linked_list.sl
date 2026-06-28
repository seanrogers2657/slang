// @test: exit_code=0
// @test: stdout=10\n20\n30\n40\n50\n150\n
// Arena linked list backed by a GROWABLE vec (not a fixed array): nodes are
// appended dynamically — the vec reallocates as it grows — and the "next
// pointer" is an integer index into the arena (-1 = end). No node owns another;
// the whole arena (both vecs) frees at scope exit.

main = () {
    var value = vec()   // value of node i
    var next = vec()    // index of node i's successor (-1 = end)

    // Build a list of 5 nodes dynamically; the vecs grow from cap 0 -> 4 -> 8.
    var n = 0
    while n < 5 {
        push(value, (n + 1) * 10)
        push(next, n + 1)
        n = n + 1
    }
    set(next, 4, 0 - 1)   // last node terminates the list

    // Traverse from the head index, printing and summing.
    var total = 0
    var cur = 0
    while cur != 0 - 1 {
        print(get(value, cur))
        total = total + get(value, cur)
        cur = get(next, cur)
    }
    print(total)   // 150
}

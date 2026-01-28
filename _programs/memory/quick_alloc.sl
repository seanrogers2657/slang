// Quick memory allocation test for profiling
// Allocates 50,000 nodes with 100µs sleep to visualize arena growth
// Run with: go run cmd/slprof/main.go ./build/output

Node = struct {
    var next: *Node?
    var a: s64
    var b: s64
    var c: s64
    var d: s64
}

main = () {
    print("Allocating nodes with sleep for profiling...")

    var head: *Node? = null
    var x = 0
    var totalCount: s64 = 0

    while x < 10 {
        var count = 0
        for ; count < 1000; count = count + 1 {
            val newNode = Heap.new(Node{ head, count, count, count, count })
            head = newNode
            sleep(10 * 1000)
        }
        assert(count == 1000, "should allocate 1000 nodes per loop")
        totalCount = totalCount + count
        print("one loop done")
        sleep(20 * 1000)
        x = x + 1
    }

    assert(x == 10, "should complete 10 loops")
    assert(totalCount == 10000, "should allocate 10000 total nodes")
    assert(head != null, "head should not be null")
    print("Allocated:")
    print(totalCount)
    print("Quick alloc test passed!")
}

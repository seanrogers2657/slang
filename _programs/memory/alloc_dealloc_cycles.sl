// Allocation/deallocation cycle test
// Allocates memory, frees it, pauses, and repeats 10 times
// Run with: go run cmd/slprof/main.go ./build/output
//
// Expected behavior: Memory usage should spike during allocation,
// drop to baseline after deallocation, and repeat in a sawtooth pattern.

Node = struct {
    var next: *Node?
    var a: s64
    var b: s64
    var c: s64
    var d: s64
}

main = () {
    print("Starting 10 alloc/dealloc cycles...")

    var cycle = 0
    while cycle < 10 {
        print("Cycle:")
        print(cycle + 1)

        // Allocate 50,000 nodes (~3 MB)
        print("  Allocating...")
        var head: *Node? = null
        var count = 0
        while count < 50000 {
            val newNode = Heap.new(Node{ head, count, count, count, count })
            head = newNode
            count = count + 1
        }
        assert(count == 50000, "should allocate 50000 nodes")
        assert(head != null, "head should not be null after allocation")
        sleep(500 * 1000 * 1000)  // 500ms to observe allocation

        // Deallocate by setting to null
        print("  Deallocating...")
        head = null
        assert(head == null, "head should be null after deallocation")

        // Pause 2-3 seconds before next cycle
        print("  Pausing...")
        sleep(2500 * 1000 * 1000)  // 2.5 seconds

        cycle = cycle + 1
    }

    assert(cycle == 10, "should complete 10 cycles")
    print("Done - 10 cycles complete")
}

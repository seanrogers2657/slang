// Large memory allocation test - forces many arena allocations
// Run with: /usr/bin/time -l ./build/output
//
// Bump allocator uses 1MB arenas. Each LargeNode is 136 bytes (17 * s64),
// which fits in the 256-byte size class.
// Nodes per arena: ~4,000
//
// This test allocates 100,000 nodes to force ~25 arena allocations.
// Expected memory: ~25 MB

LargeNode = struct {
    var next: *LargeNode?
    var a: s64
    var b: s64
    var c: s64
    var d: s64
    var e: s64
    var f: s64
    var g: s64
    var h: s64
    var i: s64
    var j: s64
    var k: s64
    var l: s64
    var m: s64
    var n: s64
    var o: s64
    var p: s64
}

main = () {
    print("Allocating 100000 large nodes (should use ~25 arenas)...")

    var head: *LargeNode? = null
    var count = 0

    // Allocate 100,000 large nodes - requires many 1MB arenas
    for ; count < 100000; count = count + 1 {
        val newNode = Heap.new(LargeNode{ head, count, count, count, count, count, count, count, count, count, count, count, count, count, count, count, count })
        head = newNode
        sleep(100 * 1000)  // 0.1ms per allocation for profiling
    }

    print("Allocated nodes:")
    print(count)
    print("Deallocating...")
    // Deallocation happens when head goes out of scope
}

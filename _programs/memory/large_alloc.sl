// Large memory allocation test - forces many arena allocations
// Run with: /usr/bin/time -l ./build/output
//
// Bump allocator uses 1MB arenas. Each LargeNode is 136 bytes (17 * i64),
// which fits in the 256-byte size class.
// Nodes per arena: ~4,000
//
// This test allocates 100,000 nodes to force ~25 arena allocations.
// Expected memory: ~25 MB

LargeNode = struct {
    var next: *LargeNode?
    var a: i64
    var b: i64
    var c: i64
    var d: i64
    var e: i64
    var f: i64
    var g: i64
    var h: i64
    var i: i64
    var j: i64
    var k: i64
    var l: i64
    var m: i64
    var n: i64
    var o: i64
    var p: i64
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

// Out-of-order deallocation test
// Verifies that the free list correctly handles deallocations
// in a different order than allocations
//
// Run with: go run cmd/slprof/main.go ./build/output
// Memory should stay at ~1 MB (single arena) throughout

Point = struct {
    var x: i64
    var y: i64
}

main = () {
    print("=== Out-of-Order Deallocation Test ===")

    // Phase 1: Allocate 5 objects (A, B, C, D, E)
    print("Phase 1: Allocating A, B, C, D, E...")
    var a: *Point? = Heap.new(Point{ 1, 10 })
    var b: *Point? = Heap.new(Point{ 2, 20 })
    var c: *Point? = Heap.new(Point{ 3, 30 })
    var d: *Point? = Heap.new(Point{ 4, 40 })
    var e: *Point? = Heap.new(Point{ 5, 50 })
    sleep(500 * 1000 * 1000)  // 500ms

    // Phase 2: Free in order C, A, E, B, D (out of order!)
    print("Phase 2: Freeing C, A, E, B, D (out of allocation order)...")
    c = null  // Free C first
    a = null  // Free A second
    e = null  // Free E third
    b = null  // Free B fourth
    d = null  // Free D last
    sleep(500 * 1000 * 1000)  // 500ms

    // Phase 3: Allocate 5 new objects - should reuse freed memory from free list
    print("Phase 3: Allocating F, G, H, I, J (reusing freed slots)...")
    var f: *Point? = Heap.new(Point{ 6, 60 })
    var g: *Point? = Heap.new(Point{ 7, 70 })
    var h: *Point? = Heap.new(Point{ 8, 80 })
    var i: *Point? = Heap.new(Point{ 9, 90 })
    var j: *Point? = Heap.new(Point{ 10, 100 })
    sleep(500 * 1000 * 1000)  // 500ms

    // Phase 4: Interleaved alloc/free pattern
    print("Phase 4: Interleaved alloc/free...")
    g = null  // Free G
    i = null  // Free I
    var k: *Point? = Heap.new(Point{ 11, 110 })  // Reuse G's slot
    var l: *Point? = Heap.new(Point{ 12, 120 })  // Reuse I's slot
    sleep(500 * 1000 * 1000)  // 500ms

    // Phase 5: Many alloc/free cycles - memory must stay flat
    print("Phase 5: 10000 alloc/free cycles...")
    var count = 0
    for ; count < 10000; count = count + 1 {
        var temp: *Point? = Heap.new(Point{ count, count * 2 })
        // temp freed each iteration, reused next iteration
        sleep(100 * 1000)  // 0.1ms per cycle = 1 second total
    }
    print("Completed cycles:")
    print(count)
    sleep(500 * 1000 * 1000)  // 500ms

    // Phase 6: Verify we can still allocate after all the cycling
    print("Phase 6: Final allocations...")
    var m: *Point? = Heap.new(Point{ 100, 200 })
    var n: *Point? = Heap.new(Point{ 300, 400 })
    sleep(500 * 1000 * 1000)  // 500ms

    print("=== Test Complete ===")
}

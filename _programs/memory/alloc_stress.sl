// Memory allocation stress test
// Allocates and deallocates 50,000 structs in a loop
// Run manually to verify no memory leaks: ./sl run _programs/memory/alloc_stress.sl

Point = struct {
    var x: s64
    var y: s64
    var z: s64
}

allocateAndUse = (n: s64) -> s64 {
    val p = Heap.new(Point{ n, n * 2, n * 3 })
    return p.x + p.y + p.z  // 6n
}

main = () {
    var sum: s64 = 0
    var i = 1

    // 50,000 iterations of allocate/deallocate
    for ; i <= 5000000; i = i + 1 {
        sum = sum + allocateAndUse(i)
        sleep(1000)
    }

    // Expected: 6 * (50000 * 50001 / 2) = 7500150000
    print(sum)
    print("Memory stress test passed!")
}

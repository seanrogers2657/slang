// Memory allocation stress test
// Allocates and deallocates 50,000 structs in a loop
// Run manually to verify no memory leaks: ./sl run _programs/memory/alloc_stress.sl

Point = struct {
    var x: s64
    var y: s64
    var z: s64
}

allocateAndUse = (n: s64) -> s64 {
    val p = new Point{ n, n * 2, n * 3 }
    return p.x + p.y + p.z  // 6n
}

main = () {
    var sum: s64 = 0
    var i = 1

    // 50,000 iterations of allocate/deallocate
    for ; i <= 50000; i = i + 1 {
        sum = sum + allocateAndUse(i)
    }

    // Expected: 6 * (50000 * 50001 / 2) = 7500150000
    assert(sum == 7500150000, "sum should be 7500150000")
    assert(i == 50001, "should complete 50000 iterations")
    print(sum)
    print("Memory stress test passed!")
}

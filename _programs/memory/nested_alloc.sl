// Nested allocation stress test
// Creates 1000 nested structures to verify proper cleanup
// Run manually: ./sl run _programs/memory/nested_alloc.sl

Inner = struct {
    var value: s64
}

Outer = struct {
    var data: *Inner
    var extra: s64
}

createAndCompute = (n: s64) -> s64 {
    val inner = new Inner{ n }
    val outer = new Outer{ inner, n * 2 }
    return outer.data.value + outer.extra
}

main = () {
    var sum: s64 = 0
    var i = 1

    for ; i <= 1000; i = i + 1 {
        sum = sum + createAndCompute(i)
    }

    // Expected: 3 * (1000 * 1001 / 2) = 1501500
    assert(sum == 1501500, "sum should be 1501500")
    assert(i == 1001, "should complete 1000 iterations")
    print(sum)
    print("Nested allocation test passed!")
}

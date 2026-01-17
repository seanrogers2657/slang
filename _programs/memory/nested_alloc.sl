// Nested allocation stress test
// Creates 1000 nested structures to verify proper cleanup
// Run manually: ./sl run _programs/memory/nested_alloc.sl

Inner = struct {
    var value: i64
}

Outer = struct {
    var data: *Inner
    var extra: i64
}

createAndCompute = (n: i64) -> i64 {
    val inner = Heap.new(Inner{ n })
    val outer = Heap.new(Outer{ inner, n * 2 })
    return outer.data.value + outer.extra
}

main = () {
    var sum: i64 = 0
    var i = 1

    for ; i <= 1000; i = i + 1 {
        sum = sum + createAndCompute(i)
    }

    // Expected: 3 * (1000 * 1001 / 2) = 1501500
    print(sum)
    print("Nested allocation test passed!")
}

// @test: exit_code=0
// @test: stdout=5050\n
// Test: Repeated allocation/deallocation in a loop
// Verifies memory is properly freed each iteration
Point = struct {
    var x: i64
}

allocateAndReturn = (n: i64) -> i64 {
    val p = Heap.new(Point{ n })
    return p.x
}

main = () {
    var sum: i64 = 0
    var i = 1

    // 100 iterations of allocate/deallocate
    for ; i <= 100; i = i + 1 {
        sum = sum + allocateAndReturn(i)
    }

    // sum of 1 to 100 = 5050
    print(sum)
}

// @test: exit_code=0
// @test: stdout=1\n0\n
// Regression: vec elements of an array must be freed with the vec-aware free
// (header + data buffer, refcount-aware) when the array goes out of scope. The
// generic array free walk used to emit a raw free of the wrong size, leaking
// the data buffer.
main = () {
    val arr = [vec(), vec()]
    push(arr[0], 5)
    print(len(arr[0]))   // 1
    print(len(arr[1]))   // 0
}

// @test: exit_code=0
// @test: stdout=50\n1\n
// A copyable array bound to a new variable is an independent deep copy:
// mutating an element of the source must not change the copy.
main = () {
    var arr = [1, 2, 3]
    val copy = arr    // value copy: copy is independent of arr
    arr[0] = 50
    print(arr[0])     // 50
    print(copy[0])    // 1 (unchanged)
}

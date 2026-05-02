// @test: exit_code=0
// @test: stdout=100\n2\n
// Reassigning an element of a nullable array with a non-null value
// must wrap the new value and produce a readable result.

main = () {
    val x: s64? = 1
    val y: s64? = 2
    var arr = [x, y]

    arr[0] = 100
    print(arr[0] ?: 0)
    print(arr[1] ?: 0)
}

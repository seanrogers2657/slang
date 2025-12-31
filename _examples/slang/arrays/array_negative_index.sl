// @test: exit_code=1
// @test: stderr_contains=array index out of bounds
main = () {
    val arr = [1, 2, 3]
    val i = 0 - 1
    print(arr[i])
}

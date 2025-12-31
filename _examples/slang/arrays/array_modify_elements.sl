// @test: exit_code=0
// @test: stdout=10\n20\n30\n
main = () {
    var arr = [1, 2, 3]
    arr[0] = 10
    arr[1] = 20
    arr[2] = 30
    print(arr[0])
    print(arr[1])
    print(arr[2])
}

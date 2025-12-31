// @test: exit_code=0
// @test: stdout=10\n20\n30\n
main = () {
    val arr = [10, 20, 30]
    var i = 0
    for ; i < len(arr); i = i + 1 {
        print(arr[i])
    }
}

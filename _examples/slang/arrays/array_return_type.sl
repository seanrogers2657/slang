// @test: exit_code=0
// @test: stdout=10\n20\n30\n3\n
// Test: s64[] type annotation with len() in loop, and variable annotation
main = () {
    val arr: s64[] = [10, 20, 30]
    for (var i = 0; i < len(arr); i = i + 1) {
        print(arr[i])
    }
    print(len(arr))
}

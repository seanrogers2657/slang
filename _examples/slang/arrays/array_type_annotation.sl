// @test: exit_code=0
// @test: stdout=1\n2\n3\n
// Test: explicit T[] type annotation on array variable
main = () {
    val arr: s64[] = [1, 2, 3]
    print(arr[0])
    print(arr[1])
    print(arr[2])
}

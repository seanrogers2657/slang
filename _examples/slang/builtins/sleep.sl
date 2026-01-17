// @test: exit_code=0
// @test: stdout=before\nafter\n
// Tests the sleep builtin function
main = () {
    print("before")
    sleep(10000000) // 10ms - short enough for fast tests
    print("after")
}

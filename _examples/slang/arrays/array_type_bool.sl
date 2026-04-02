// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test: explicit bool[] type annotation
main = () {
    val flags: bool[] = [true, false]
    print(flags[0])
    print(flags[1])
}

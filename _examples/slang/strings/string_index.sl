// @test: exit_code=0
// @test: stdout=72\n101\n108\n108\n111\n
// s[i] returns the byte at index i as u8
main = () {
    val s = "Hello"
    print(s[0])
    print(s[1])
    print(s[2])
    print(s[3])
    print(s[4])
}

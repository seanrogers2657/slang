// @test: exit_code=1
// @test: stderr=panic: substring range out of bounds\nat main()\n
// An end index past the string length triggers a runtime panic
main = () {
    val s = "abcde"
    print(substr(s, 0, 6))
}

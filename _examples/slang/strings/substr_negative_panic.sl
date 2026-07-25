// @test: exit_code=1
// @test: stderr=panic: substring range out of bounds\nat main()\n
// A negative start index triggers a runtime panic
main = () {
    val s = "abcde"
    var i = 0
    i = i - 1
    print(substr(s, i, 2))
}

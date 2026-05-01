// @test: exit_code=1
// @test: stderr=panic: array index out of bounds\nat main()\n
// Out-of-bounds string index triggers a runtime panic
main = () {
    val s = "abc"
    print(s[5])
}

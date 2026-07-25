// @test: exit_code=1
// @test: stderr=panic: substring range out of bounds\nat main()\n
// A reversed range (end < start) triggers a runtime panic rather than
// silently producing an empty string
main = () {
    val s = "abcde"
    print(substr(s, 3, 1))
}

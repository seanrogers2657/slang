// @test: exit_code=0
// @test: stdout=104\n105\n
// String indexing works with variable indices
main = () {
    val s = "hi"
    for (var i: s64 = 0; i < len(s); i = i + 1) {
        print(s[i])
    }
}

// @test: exit_code=0
// @test: stdout=99\n
// Chained elvis operators using intermediate variables
main = () {
    val a: s64? = null
    val b: s64? = null
    val bDefault = b ?: 99
    val result = a ?: bDefault
    print(result)
}

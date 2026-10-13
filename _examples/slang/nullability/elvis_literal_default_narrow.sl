// @test: stdout=255\n9\n
// An elvis default literal takes the nullable's inner type.
main = () {
    val n: u8? = null
    print((n ?: 5) + 250)
    val m: s64? = null
    print(m ?: 9)
}

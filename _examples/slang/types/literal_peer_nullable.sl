// @test: stdout=3\n250\nnull\n
// A bare literal next to a nullable integer peer takes the inner type, and
// the branches unify to the nullable type.
main = () {
    val n: u8? = 3u8
    val x = if true { n } else { 5 }
    print("${x}")
    val m: u8? = null
    val y = if false { m } else { 250 }
    print("${y}")
    val z = when {
        false -> 7
        else -> m
    }
    print("${z}")
}

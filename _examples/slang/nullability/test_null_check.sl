// @test: exit_code=0
// @test: stdout=false\n
// Simple test: check that a null variable reads as null
main = () {
    val y: i64? = null
    print(y != null)  // false
}

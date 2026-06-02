// @test: exit_code=0
// @test: stdout=outer inner Ada end\n
// A nested interpolated string inside an interpolation expression.
main = () {
    val n = "Ada"
    print("outer ${ "inner ${n}" } end")
}

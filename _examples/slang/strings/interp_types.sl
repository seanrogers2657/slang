// @test: exit_code=0
// @test: stdout=n=42\nneg=-7\nflag=false\n
// Interpolating s64 (including negative) and bool values.
main = () {
    print("n=${42}")
    print("neg=${-7}")
    val flag = false
    print("flag=$flag")
}

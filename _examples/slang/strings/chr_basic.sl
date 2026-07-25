// @test: exit_code=0
// @test: stdout=A\n1\nHi\nHello, World!\ntrue\n
// chr(b) builds a one-byte string from a byte value — the inverse of s[i]
main = () {
    print(chr(65))
    print(len(chr(0)))

    val s = "Hi"
    print(chr(s[0]) + chr(s[1]))

    // Rebuilding a string byte by byte round-trips through chr/s[i]
    val src = "Hello, World!"
    var out = ""
    for (var i = 0; i < len(src); i = i + 1) {
        out = out + chr(src[i])
    }
    print(out)
    print(out == src)
}

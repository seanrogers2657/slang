// @test: exit_code=0
// @test: stdout=Hello\nWorld\nHello, World!\n0\n0\n!\n
// substr(s, start, end) slices the half-open byte range [start, end)
main = () {
    val s = "Hello, World!"
    print(substr(s, 0, 5))
    print(substr(s, 7, 12))

    // The full range reproduces the string
    print(substr(s, 0, len(s)))

    // An empty range is legal anywhere in [0, len], including at the end
    print(len(substr(s, 3, 3)))
    print(len(substr(s, len(s), len(s))))

    // The last byte
    print(substr(s, len(s) - 1, len(s)))
}

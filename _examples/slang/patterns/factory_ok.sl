// @test: exit_code=0
// @test: stdout=42\n
// Factory pattern. A factory returns a VALUE (copied to the caller), not a *T.
// "Allocate and hand back a pointer" is impossible under scope-frees-it, but
// returning a value is just as ergonomic for plain data.
Shape = struct {
    val kind: s64
    val size: s64
}

make_shape = (kind: s64, size: s64) -> Shape {
    return Shape{ kind, size }
}

main = () {
    val s = make_shape(1, 42)
    print(s.size)
}

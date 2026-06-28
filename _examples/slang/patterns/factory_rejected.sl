// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot be used as a return type
// Factory, natural form: return a heap pointer. Rejected — an owned pointer is
// freed at the end of its creating scope, so it cannot escape via return.
// (See factory_ok.sl: return a value instead.)
Shape = struct { val kind: s64  val size: s64 }

make_shape = (kind: s64, size: s64) -> *Shape {
    return new Shape{ kind, size }
}

main = () {
    val s = make_shape(1, 42)
    print(s.size)
}

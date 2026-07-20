// @test: exit_code=0
// @test: stdout=m=5\n
// Regression: embedding a struct-returning call result in a literal
// (Wrap{ make_point(5), 9 }) byte-copies the temp into the embedded region.
// The temp's heap fields transfer with the copy, so only the temp's shell must
// be freed — deep-copy fixups here would orphan the temp and leak both it and
// its buffers.
Point = struct { var name: string  var x: s64 }
Wrap = struct { var p: Point  var z: s64 }

make_point = (n: s64) -> Point {
    return Point{ "m=${n}", n }
}

main = () {
    val w = Wrap{ make_point(5), 9 }
    print(w.p.name)   // m=5
}

// @test: exit_code=0
// @test: stdout=p=1\np=1\n
// Regression: embedding an existing struct binding in another literal
// (val w = Wrap{ pt, 9 }) byte-copies the embedded region, which used to alias
// pt's string buffer into w — both then freed the same buffer at scope exit (a
// double free). The embedded region's heap fields must be deep-copied.
Point = struct {
    var name: string
    var x: s64
}
Wrap = struct {
    var p: Point
    var z: s64
}

main = () {
    val n = 1
    val pt = Point{ "p=${n}", 4 }
    val w = Wrap{ pt, 9 }
    print(w.p.name)   // p=1
    print(pt.name)    // p=1 — pt still owns its own buffer
}

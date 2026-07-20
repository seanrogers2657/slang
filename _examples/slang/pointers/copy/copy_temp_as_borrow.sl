// @test: exit_code=0
// @test: stdout=70\n
// Regression: an unbound .copy() result passed directly as a borrow argument
// has no binding to free it at scope exit; the caller must free it after the
// call. It leaked one copy per iteration.
Point = struct { var x: s64  var y: s64 }

sum_pt = (p: &Point) -> s64 {
    return p.x + p.y
}

main = () {
    val p = new Point{ 3, 4 }
    var total = 0
    for (var i = 0; i < 10; i = i + 1) {
        total = total + sum_pt(p.copy())
    }
    print(total)   // 70
}

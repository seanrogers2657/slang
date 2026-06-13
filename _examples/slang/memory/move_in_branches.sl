// @test: exit_code=0
// @test: stdout=30\n
// Regression: a value moved inside one branch of an if/else (or when) must not
// be reported as moved in the mutually-exclusive sibling branch. The ownership
// tracker used to leak the then-branch move into the else-branch analysis.
Point = struct {
    val x: s64
    val y: s64
}

consume = (p: *Point) -> s64 {
    return p.x + p.y
}

main = () {
    val p = new Point{ 10, 20 }
    val cond = true
    if cond {
        print(consume(p))   // moves p here
    } else {
        print(consume(p))   // ...and here; mutually exclusive, so this is fine
    }
}

// @test: exit_code=0
// @test: stdout=30\n
// Regression: an owned value borrowed inside one branch of an if/else (or when)
// must remain usable in the mutually-exclusive sibling branch. The ownership
// tracker used to leak then-branch usage into the else-branch analysis.
Point = struct {
    val x: s64
    val y: s64
}

consume = (p: &Point) -> s64 {
    return p.x + p.y
}

main = () {
    val p = new Point{ 10, 20 }
    val cond = true
    if cond {
        print(consume(p))   // borrows p here
    } else {
        print(consume(p))   // ...and here; mutually exclusive, so this is fine
    }
}

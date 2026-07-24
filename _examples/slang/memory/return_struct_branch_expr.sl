// @test: exit_code=0
// @test: stdout=5\n9\n
// Regression: returning an if-expression that yields a struct must hand the
// caller an owned value. The branch result is copied/owned at the branch, so
// the returned struct survives the function's scope-exit cleanup (previously
// it could return a borrow of a local that was then freed). A leak or double
// free aborts the runtime (exit 134), so exit_code=0 guards both directions.
Point = struct { val x: s64  val y: s64 }

make = () -> Point { return Point{ 5, 6 } }

pick = (c: bool) -> Point {
    return if c { make() } else { Point{ 9, 9 } }
}

main = () {
    val p = pick(true)
    print(p.x)
    val q = pick(false)
    print(q.x)
}

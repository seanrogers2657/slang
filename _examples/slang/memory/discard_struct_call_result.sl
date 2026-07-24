// @test: exit_code=0
// @test: stdout=0\n
// Regression: a struct-returning call used as a bare statement is discarded.
// The return value is heap-allocated for the caller, and with no binding to own
// it the discard cleanup must free it (previously the cleanup only handled
// string?/vec?/owned-pointer temps, leaking the struct). A double free here
// would crash, so exit_code=0 guards both directions.
Point = struct { val x: s64  val y: s64 }
Boxed = struct { val s: string }

make = () -> Point { return Point{ 1, 2 } }
make_boxed = () -> Boxed { return Boxed{ "hello" } }

main = () {
    make()
    make_boxed()
    print(0)
}

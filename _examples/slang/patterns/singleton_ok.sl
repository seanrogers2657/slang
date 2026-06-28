// @test: exit_code=0
// @test: stdout=2\n
// Singleton / shared mutable state. The single instance is owned by the top
// scope and borrowed downward. Note it is allocated with `new`: only an owned
// pointer auto-borrows into a free function's &&T param (a plain value does not).
Registry = struct { var count: s64 }

bump   = (r: &&Registry) { r.count = r.count + 1 }
report = (r: &Registry)  { print(r.count) }

main = () {
    val g = new Registry{ 0 }
    bump(g)
    bump(g)
    report(g)
}

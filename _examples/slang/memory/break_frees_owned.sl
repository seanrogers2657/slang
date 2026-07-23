// @test: exit_code=0
// @test: stdout=3\n
// Regression: an owned local allocated in a loop body must be freed when `break`
// leaves the loop (and when break sits inside a nested block that allocated its
// own owner). Without per-exit cleanup the heap is left unbalanced and the
// runtime aborts at exit. Here the heap stays balanced and we print the break
// index.
P = struct { var x: s64 }

main = () {
    var i = 0
    while i < 10 {
        val p = new P{ i }       // owned: freed each iteration AND on break
        if p.x == 3 {
            val q = new P{ 99 }  // also owned; must be freed by the break path
            break
        }
        i = i + 1
    }
    print(i)
}

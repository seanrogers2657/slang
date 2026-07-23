// @test: exit_code=0
// @test: stdout=8\n
// Regression: an owned local allocated in a loop body must be freed when
// `continue` restarts the loop, skipping the body's fall-through cleanup.
// Summing 0+1+3+4 (the iteration where p.x == 2 continues early) gives 8, and
// the heap stays balanced so the program exits cleanly.
P = struct { var x: s64 }

main = () {
    var sum = 0
    for (var i = 0; i < 5; i = i + 1) {
        val p = new P{ i }       // owned: freed each iteration AND on continue
        if p.x == 2 {
            continue
        }
        sum = sum + p.x
    }
    print(sum)
}

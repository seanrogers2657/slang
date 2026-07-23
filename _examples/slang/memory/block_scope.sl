// @test: exit_code=0
// @test: stdout=42\n1\n
// A bare { } block opens a nested scope. Heap allocated inside it is freed at
// the closing brace, before the rest of the enclosing scope runs.
Big = struct { var x: s64 }

main = () {
    val keep = 1
    {
        val scratch = new Big{ 42 }   // owned by this block
        print(scratch.x)
    }                                  // scratch freed here
    print(keep)
}

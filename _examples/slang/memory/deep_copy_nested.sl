// @test: exit_code=0
// @test: stdout=9\n9\n42\n9\n
// Regression: .copy() must deep-copy owned pointers at every nesting level.
// The recursive copy reused registers that aliased its dest/src pointers, so
// copies beyond two levels (C -> *B -> *A) produced garbage / shared state.
A = struct { var v: s64 }
B = struct { var a: *A }
C = struct { var b: *B }

main = () {
    val c1 = new C{ new B{ new A{ 9 } } }
    val c2 = c1.copy()
    print(c1.b.a.v)   // 9
    print(c2.b.a.v)   // 9 (independent copy)

    // Mutating the copy must not affect the original (proves no aliasing).
    c2.b.a.v = 42
    print(c2.b.a.v)   // 42
    print(c1.b.a.v)   // 9
}

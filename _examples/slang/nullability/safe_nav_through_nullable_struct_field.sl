// @test: exit_code=0
// @test: stdout=false\n42\ntrue\n-1\n
// Regression: safe navigation (?.) through a nullable struct-typed field used
// to crash IR validation ("FieldPtr argument must be a pointer, got Inner") —
// a nullable struct is represented as a pointer to its storage, so the unwrap
// must yield that pointer. Non-null yields the field; null short-circuits to
// null. Constructing/freeing the nullable struct field is also leak-clean now.
Inner = struct { val v: s64 }
Outer = struct { val inner: Inner? }

main = () {
    val a = Outer{ Inner{ 42 } }
    print(a.inner?.v == null)   // false
    print(a.inner?.v ?: -1)     // 42

    val b = Outer{ null }
    print(b.inner?.v == null)   // true
    print(b.inner?.v ?: -1)     // -1
}

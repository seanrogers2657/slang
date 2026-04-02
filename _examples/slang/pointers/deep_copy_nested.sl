// @test: exit_code=42
// @test: stdout=10\n10\n100\n10\n
// Test deep copy with single level of pointer nesting

Inner = struct {
    var value: s64
}

Outer = struct {
    var inner: *Inner
}

main = () {
    // Create nested structure (one level of pointer)
    val inner = new Inner{ 10 }
    val outer = new Outer{ inner }

    // Make a deep copy
    val copy = outer.copy()

    // Print original and copy values (both should be 10)
    print(outer.inner.value)  // 10
    print(copy.inner.value)   // 10 (deep copy)

    // Modify original
    outer.inner.value = 100

    // Print again - copy should be unchanged
    print(outer.inner.value)  // 100
    print(copy.inner.value)   // Should still be 10 if deep copy works

    // Verify deep copy worked
    if copy.inner.value == 10 {
        exit(42)  // Success
    }
    exit(1)  // Failed - copy was affected by original
}

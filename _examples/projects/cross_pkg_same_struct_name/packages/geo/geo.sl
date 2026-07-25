// A struct with the same name as the root package's.
Point = struct {
    val x: s64
    val y: s64
}

// Take the coordinates by value, so the test measures only the struct-name
// collision and allocates no Point of its own.
sum = (x: s64, y: s64) -> s64 {
    return x + y
}

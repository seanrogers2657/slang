// Reassigning an owned pointer to freshly produced values.
// Under the scope-frees-it model ownership never transfers (there is no move),
// but a `var` of owned-pointer type can be rebound repeatedly to a fresh `new`
// value: each reassignment frees the previous allocation, and the final value
// is freed when the variable goes out of scope. The heap stays balanced.

Point = struct {
    var x: s64
    var y: s64
}

main = () {
    var a = new Point{ 2, 3 }
    assert(a.x == 2, "initial x should be 2")
    assert(a.y == 3, "initial y should be 3")

    // Rebind `a` to a fresh owned Point built from its current value. The right
    // side must produce a new value (new / .copy()), not alias another owner;
    // the old allocation is freed by the assignment.
    var i = 0
    while i < 6 {
        a = new Point{ a.x, a.y }
        assert(a.x == 2, "x should still be 2 after rebind")
        i = i + 1
    }
    assert(a.y == 3, "y should still be 3")

    // `a` is still valid; its final allocation is freed when it goes out of scope.
    print("Owned reassignment test passed!")
}

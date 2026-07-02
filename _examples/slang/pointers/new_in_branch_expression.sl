// @test: exit_code=0
// @test: stdout=11\n2\n5\n
// Regression: `new` as the result of an if/when branch expression is legal (it
// is a fresh value, not an alias) and the binding's scope owns the allocation.
// This used to segfault: the branch's block scope freed the allocation while
// the binding outlived it, and getLastValue() could return a trailing field
// store instead of the allocation itself.
Point = struct { var x: s64  var y: s64 }

main = () {
    val q = if true { new Point{ 11, 2 } } else { new Point{ 33, 4 } }
    print(q.x)   // 11
    print(q.y)   // 2

    val r = when {
        true -> {
            new Point{ 5, 6 }   // block-valued branch, still owned by r's scope
        }
        else -> new Point{ 7, 8 }
    }
    print(r.x)   // 5
}

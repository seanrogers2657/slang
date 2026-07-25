// @test: exit_code=0
// @test: stdout=1\n2\n15\n
// Struct names are never package-mangled, so two packages that each declare a
// `Point` produce two IR struct types with the same name. Combining the
// per-package IR programs must not adopt the other program's struct list, or
// the merged program fails validation with "duplicate struct name".
geo = import "geo"

Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Point{ 1, 2 }
    print(p.x)
    print(p.y)

    val q = geo.Point{ 7, 8 }
    print(geo.sum(q.x, q.y))
}

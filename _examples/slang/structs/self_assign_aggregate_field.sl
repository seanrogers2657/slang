// @test: exit_code=0
// @test: stdout=hi1\n1\nhi1\nhi1\nyo2\nz9\n
// Regression: self-assigning an embedded aggregate field (o.inner = o.inner)
// must not free the field's heap before copying, since the source aliases that
// same storage. Freeing first hung on a string field (string copy read freed
// memory as a length and looped) and double-freed a vec field. The source is
// now deep-copied into an independent value before the old contents are freed.
Sname = struct { var name: string }
Svec = struct { var items: vec }
Mid = struct { var inner: Sname }
NestS = struct { var inner: Sname }
NestM = struct { var mid: Mid }

main = () {
    var a = NestS{ Sname{ "hi${1}" } }
    a.inner = a.inner            // self-assign, string field
    print(a.inner.name)          // hi1

    var b = Svec{ vec() }
    push(b.items, 5)
    var wrap = Mid{ Sname{ "x${0}" } }
    var vb = NestS{ Sname{ "y${0}" } }
    var vv = Svec{ vec() }
    push(vv.items, 9)
    vv = vv                      // (whole-var self-assign is a separate path)
    print(len(b.items))          // 1

    var d = NestM{ Mid{ Sname{ "hi${1}" } } }
    d.mid.inner = d.mid.inner    // deep self-assign
    print(d.mid.inner.name)      // hi1
    d.mid = d.mid                // self-assign one level up
    print(d.mid.inner.name)      // hi1

    // Cross-field assign from another struct is an independent copy.
    var o = NestS{ Sname{ "o${0}" } }
    var p = NestS{ Sname{ "yo${2}" } }
    o.inner = p.inner
    p.inner.name = "z${9}"
    print(o.inner.name)          // yo2 — o unaffected by p's mutation
    print(p.inner.name)          // z9
}

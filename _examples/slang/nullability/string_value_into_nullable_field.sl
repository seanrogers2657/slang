// @test: exit_code=0
// @test: stdout=alice\nalice\nn1\nnone\n
// Regression: constructing a struct/class whose field is string? from a string
// VALUE (a variable or interpolated string, not a literal) must copy the string
// before widening it into the box. The old order wrapped first and then ran the
// string copy on the box pointer, reading a heap address as a length and
// looping forever at scope-exit free.
C = class { var owner: string?  val id: s64 }

main = () {
    val name = "alice"       // string variable (borrow)
    val a = new C{ name, 1 }
    print(a.owner ?: "-")    // alice
    print(name)              // alice — source keeps its own buffer

    val h = "n${1}"          // interpolated (heap) string in a variable
    val b = new C{ h, 2 }
    print(b.owner ?: "-")    // n1

    val c = new C{ null, 3 }
    print(c.owner ?: "none") // none
}

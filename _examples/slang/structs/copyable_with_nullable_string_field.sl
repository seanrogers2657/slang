// @test: exit_code=0
// @test: stdout=one\ntwo\none\n
// Regression: deep-copying an aggregate with a string? field failed IR
// validation ("argument used before definition") — the copy fixups walked
// with a builder that went stale when the nullable copy split blocks. Copies
// must be independent: mutating the copy's field leaves the original intact.
Named = struct { var label: string?  val id: s64 }

main = () {
    var a = Named{ "one${""}", 1 }
    var b = a                     // deep copy incl. the string? box
    b.label = "two${""}"
    print(a.label ?: "none")      // one
    print(b.label ?: "none")      // two

    val h = new Named{ "one${""}", 2 }
    val c = h.copy()
    print(c.label ?: "none")      // one
}

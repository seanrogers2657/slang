// @test: exit_code=0
// @test: stdout=k1\n1\nk1\nnone\nc2\nnone\n3\nC:k1\n
// Safe navigation reads fields of a nullable class, as it does for structs.
C = class {
    val id: s64
    var label: string
    show = (self: &C) -> string { return "C:${self.label}" }
}
find = (k: s64) -> C? {
    if k > 0 { return C{ k, "c${k}" } }
    return null
}
main = () {
    val c: C? = C{ 1, "k${1}" }
    print(c?.label ?: "-")
    print(c?.id ?: 0)
    val s = c?.label
    print("${s}")
    val n: C? = null
    print(n?.label ?: "none")
    print(find(2)?.label ?: "-")
    print(find(0)?.label ?: "none")
    print(find(3)?.id ?: 0)
    print(c?.show() ?: "-")
}

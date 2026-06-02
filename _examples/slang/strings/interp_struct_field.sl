// @test: exit_code=0
// @test: stdout=Ada\nDr. Ada\n
// An interpolated string stored in a struct field, then re-interpolated. The
// struct owns its string buffer and frees it when freed (heap stays balanced).
Person = struct {
    val name: string
}

main = () {
    val who = "Ada"
    val p = new Person{ "${who}" }
    print(p.name)
    print("Dr. ${p.name}")
}

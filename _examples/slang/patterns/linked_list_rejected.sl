// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot be used as a class field
// Linked list, natural form: a node with a "next" pointer to another node.
// Rejected — *T cannot be a field, so nodes cannot own other nodes.
// (See composite_ok.sl / the arena examples: links are integer indices.)
ListNode = class {
    var value: s64
    var next: *ListNode?
}

main = () {
    val head = new ListNode{ 1, null }
    print(head.value)
}

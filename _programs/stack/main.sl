// Stack Data Structure using Linked List
// TRIGGERS: Bug 1 (!obj.method), Bug 2 (elvis chaining)

StackNode = struct {
    var next: *StackNode?
    val value: s64
}

Stack = class {
    var top: *StackNode?
    var size: s64

    create = () -> *Stack {
        return new Stack{ null, 0 }
    }

    push = (self: &&Stack, value: s64) {
        val node = new StackNode{ self.top, value }
        self.top = node
        self.size = self.size + 1
    }

    peek = (self: &Stack) -> s64? {
        return self.top?.value
    }

    is_empty = (self: &Stack) -> bool {
        return self.size == 0
    }

    get_size = (self: &Stack) -> s64 {
        return self.size
    }
}

main = () {
    val s = Stack.create()

    assert(s.is_empty(), "new stack should be empty")
    assert(s.get_size() == 0, "new stack size should be 0")
    assert(s.peek() == null, "peek on empty stack should be null")

    s.push(10)
    s.push(20)
    s.push(30)
    s.push(40)
    s.push(50)

    assert(s.get_size() == 5, "size should be 5")
    assert(!s.is_empty(), "stack should not be empty")

    val peeked = s.peek() ?: 0
    assert(peeked == 50, "peek should return 50")

    for (var i = 100; i < 110; i = i + 1) {
        s.push(i)
    }

    assert(s.get_size() == 15, "size should be 15")

    val new_top = s.peek() ?: 0
    assert(new_top == 109, "peek should return 109")

    // Chain safe navigation to walk the list
    val v1 = s.top?.value
    val v2 = s.top?.next?.value
    val v3 = s.top?.next?.next?.value

    assert(v1 != null, "v1 should exist")
    assert(v2 != null, "v2 should exist")
    assert(v3 != null, "v3 should exist")

    print("Stack size:")
    print(s.get_size())
    print("Stack test passed!")
}

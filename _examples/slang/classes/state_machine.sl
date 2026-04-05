// @test: exit_code=0
// @test: stdout=1\n2\n3\n1\n
// State machine using class methods and when expressions

Door = class {
    var state: s64

    create = () -> *Door {
        return new Door{ 1 }
    }

    unlock = (self: &&Door) -> bool {
        if self.state != 1 { return false }
        self.state = 2
        return true
    }

    open_door = (self: &&Door) -> bool {
        if self.state != 2 { return false }
        self.state = 3
        return true
    }

    close_and_lock = (self: &&Door) -> bool {
        if self.state != 3 { return false }
        self.state = 1
        return true
    }
}

main = () {
    val door = Door.create()
    print(door.state)

    assert(door.open_door() == false, "can't open locked door")
    assert(door.state == 1, "should still be locked")

    assert(door.unlock(), "should unlock")
    print(door.state)

    assert(door.unlock() == false, "can't unlock twice")

    assert(door.open_door(), "should open")
    print(door.state)

    assert(door.close_and_lock(), "should close and lock")
    print(door.state)

    for (var i = 0; i < 5; i = i + 1) {
        assert(door.unlock(), "cycle unlock")
        assert(door.open_door(), "cycle open")
        assert(door.close_and_lock(), "cycle lock")
    }
    assert(door.state == 1, "should end locked after cycles")
}

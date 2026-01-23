// @test: exit_code=60
// Test methods with side effects (modifying multiple fields)

State = class {
    var x: s64
    var y: s64
    var z: s64

    create = () -> *State {
        return Heap.new(State{ 0, 0, 0 })
    }

    // Method modifying multiple fields
    setAll = (self: &&State, v: s64) {
        self.x = v
        self.y = v
        self.z = v
    }

    // Method with sequential modifications
    increment = (self: &&State) {
        self.x = self.x + 1
        self.y = self.y + 2
        self.z = self.z + 3
    }

    // Method using and modifying
    swapXY = (self: &&State) {
        val temp = self.x
        self.x = self.y
        self.y = temp
    }

    sum = (self: &State) -> s64 {
        return self.x + self.y + self.z
    }
}

main = () {
    val s = State.create()

    s.setAll(10)    // x=10, y=10, z=10
    s.increment()   // x=11, y=12, z=13
    s.swapXY()      // x=12, y=11, z=13
    s.increment()   // x=13, y=13, z=16

    // sum = 13 + 13 + 16 = 42
    s.setAll(20)    // x=20, y=20, z=20
    // sum = 60

    exit(s.sum())
}

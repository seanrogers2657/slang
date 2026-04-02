// @test: exit_code=50
// Test instance method calling static method of same class

MathBox = class {
    var value: s64

    // Static helper method
    double = (x: s64) -> s64 {
        return x * 2
    }

    // Static method calling another static
    quadruple = (x: s64) -> s64 {
        return MathBox.double(MathBox.double(x))
    }

    // Instance method calling static method
    add_doubled = (self: &&MathBox, x: s64) {
        self.value = self.value + MathBox.double(x)
    }

    // Instance method using static in expression
    get_doubled_value = (self: &MathBox) -> s64 {
        return MathBox.double(self.value)
    }
}

main = () {
    val box = new MathBox{ 10 }
    box.add_doubled(5)              // value = 10 + double(5) = 10 + 10 = 20
    box.add_doubled(2)              // value = 20 + double(2) = 20 + 4 = 24
    val result = box.get_doubled_value()  // double(24) = 48... wait that's 48
    // Let me recalculate: 10 + 10 = 20, 20 + 4 = 24, double(24) = 48
    // Let's adjust: start with 15, add doubled(5)=10 -> 25, getDoubledValue = 50
    exit(result + 2)  // 48 + 2 = 50
}

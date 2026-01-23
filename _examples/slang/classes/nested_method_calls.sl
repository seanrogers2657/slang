// @test: exit_code=64
// Test nested method calls - method(method(method()))

Transformer = class {
    var offset: i64

    create = (offset: i64) -> *Transformer {
        return Heap.new(Transformer{ offset })
    }

    // Method to be nested
    transform = (self: &Transformer, x: i64) -> i64 {
        return x + self.offset
    }

    // Static version for nesting
    add = (a: i64, b: i64) -> i64 {
        return a + b
    }

    // Method that calls other methods
    compute = (self: &Transformer, x: i64) -> i64 {
        return self.transform(self.transform(x))
    }
}

main = () {
    val t = Transformer.create(10)

    // Nested instance method calls
    // transform(transform(transform(4))) = transform(transform(14)) = transform(24) = 34
    val r1 = t.transform(t.transform(t.transform(4)))

    // Nested static method calls
    // add(add(5, 10), add(5, 10)) = add(15, 15) = 30
    val r2 = Transformer.add(Transformer.add(5, 10), Transformer.add(5, 10))

    exit(r1 + r2)  // 34 + 30 = 64
}

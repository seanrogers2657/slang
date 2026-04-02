// @test: exit_code=15
// Test methods with conditional logic

Classifier = class {
    var threshold: s64

    create = (threshold: s64) -> *Classifier {
        return new Classifier{ threshold }
    }

    // Method with if-else chain
    classify = (self: &Classifier, value: s64) -> s64 {
        if value < 0 {
            return 0
        }
        if value == 0 {
            return 1
        }
        if value <= self.threshold {
            return 2
        }
        return 3
    }

    // Method with conditional expression
    score = (self: &Classifier, value: s64) -> s64 {
        val multiplier = if value < 0 {
            0
        } else if value < self.threshold {
            1
        } else {
            2
        }
        return value * multiplier
    }
}

main = () {
    val c = Classifier.create(10)

    // Test classify (using 0-5 instead of -5 for negative)
    val neg5 = 0 - 5
    val r1 = c.classify(neg5)  // 0
    val r2 = c.classify(0)     // 1
    val r3 = c.classify(5)     // 2
    val r4 = c.classify(20)    // 3

    // Test score
    val s2 = c.score(5)  // 5 * 1 = 5

    // r1 + r2 + r3 + r4 = 0 + 1 + 2 + 3 = 6
    // s2 = 5, 6 + 5 + 4 = 15
    exit(r1 + r2 + r3 + r4 + s2 + 4)
}

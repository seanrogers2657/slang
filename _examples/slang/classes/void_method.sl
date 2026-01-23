// @test: exit_code=30
// Test void methods (methods with no return value)

Logger = class {
    var lastValue: i64

    create = () -> *Logger {
        return Heap.new(Logger{ 0 })
    }

    // Void method with explicit void return type
    log = (self: &&Logger, value: i64) {
        self.lastValue = value
    }

    // Void method that does multiple things
    reset = (self: &&Logger) {
        self.lastValue = 0
    }

    // Void static method
    staticLog = (value: i64) {
        // Just does something without returning
        val temp = value * 2
    }

    getLast = (self: &Logger) -> i64 {
        return self.lastValue
    }
}

main = () {
    val logger = Logger.create()

    // Call void methods
    logger.log(10)
    val v1 = logger.getLast()  // 10

    logger.log(20)
    val v2 = logger.getLast()  // 20

    logger.reset()
    val v3 = logger.getLast()  // 0

    // Call static void method (just verify it doesn't crash)
    Logger.staticLog(100)

    exit(v1 + v2 + v3)  // 10 + 20 + 0 = 30
}

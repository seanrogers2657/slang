// @test: exit_code=30
// Test void methods (methods with no return value)

Logger = class {
    var last_value: s64

    create = () -> *Logger {
        return new Logger{ 0 }
    }

    // Void method with explicit void return type
    log = (self: &&Logger, value: s64) {
        self.last_value = value
    }

    // Void method that does multiple things
    reset = (self: &&Logger) {
        self.last_value = 0
    }

    // Void static method
    static_log = (value: s64) {
        // Just does something without returning
        val temp = value * 2
    }

    get_last = (self: &Logger) -> s64 {
        return self.last_value
    }
}

main = () {
    val logger = Logger.create()

    // Call void methods
    logger.log(10)
    val v1 = logger.get_last()  // 10

    logger.log(20)
    val v2 = logger.get_last()  // 20

    logger.reset()
    val v3 = logger.get_last()  // 0

    // Call static void method (just verify it doesn't crash)
    Logger.static_log(100)

    exit(v1 + v2 + v3)  // 10 + 20 + 0 = 30
}

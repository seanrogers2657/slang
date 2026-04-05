import "logger"

validate = (value: s64) -> s64 {
    if value < 0 {
        logger.log_value(-1)
        return 0
    }
    logger.log_value(value)
    return value
}

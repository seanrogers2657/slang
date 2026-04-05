import "config"

// This runs during init — config.db_port must already be initialized
val connection_port = config.db_port + 1

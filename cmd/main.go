package main

import (
	"clients-test/internal/logger"
)

var logPrefix = "MAIN"

func main() {
	logger.InfoWithPrefix(logPrefix, "Starting Notifications service")
}
